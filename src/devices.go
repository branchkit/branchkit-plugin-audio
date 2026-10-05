package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/branchkit/plugin-sdk-go"
)

// The two device collections. Bare names because other plugins read them
// (voice floats `audio_inputs` as a list you can say); the platform reserves
// `plugin.*` for collections nobody else may read.
const (
	outputsCollection = "audio_outputs"
	inputsCollection  = "audio_inputs"
)

// deviceRecord is one name you can say for one device. A device has several:
// its own name, a shorter one when that is unambiguous, and any the user
// adds. Each is its own record because the collection is keyed by what you
// SAY, the way `apps` is keyed by alias, and the value the matcher hands the
// action is the device's uid, so the handler never guesses from a name.
type deviceRecord struct {
	Spoken string `json:"spoken"`
	UID    string `json:"uid"`
	// Name is the device's display name, so a record still says which device
	// it is while that device is unplugged. A user alias outlives the device
	// it names: the next time it is plugged in, the alias works again.
	Name string `json:"name,omitempty"`
}

// direction is which way a device carries sound, and which collection and
// platform device type go with it.
type direction struct {
	collection string
	deviceType string // native.set_audio_device's device_type
	has        func(branchkit.AudioDevice) bool
}

var (
	output = direction{outputsCollection, "output", func(d branchkit.AudioDevice) bool { return d.IsOutput }}
	input  = direction{inputsCollection, "input", func(d branchkit.AudioDevice) bool { return d.IsInput }}
)

// spokenForm is the form a name is stored and compared in. Lowercase and
// trimmed only: the platform normalizes spoken keys for the engine (format
// characters, punctuation) on its side, and stored records keep the text as
// written.
func spokenForm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// shortNamePrefixes are the parts of a device name that say which machine it
// is on rather than which device it is. Only a prefix that leaves something
// behind is stripped, and only when nothing else ends up with the same short
// name (deviceRecords checks that).
var shortNamePrefixes = []string{
	"macbook air ", "macbook pro ", "macbook ", "imac ", "mac mini ", "mac studio ", "mac pro ",
	"built-in ", "external ",
}

// shortName derives the name a person actually says for a device:
// "MacBook Air Speakers" → "speakers", "External Headphones" → "headphones",
// and, for the names Windows gives, "Speakers (Realtek(R) Audio)" → "speakers".
// Returns "" when there is nothing shorter to say.
func shortName(name string) string {
	full := spokenForm(name)
	s := full
	if i := strings.Index(s, " ("); i > 0 {
		s = strings.TrimSpace(s[:i])
	}
	for _, p := range shortNamePrefixes {
		if strings.HasPrefix(s, p) && len(s) > len(p) {
			s = strings.TrimSpace(strings.TrimPrefix(s, p))
			break
		}
	}
	if s == full {
		return ""
	}
	return s
}

// isAutoAggregate reports whether d is an aggregate device macOS created for
// one process's own use — named and identified "CADefaultDeviceAggregate-
// <pid>-<n>", as a voice-processing audio unit makes. Nobody picks it as an
// output, and because its name carries a process id, every relaunch would add
// new, unsayable words to the speech vocabulary, which never forgets a word.
func isAutoAggregate(d branchkit.AudioDevice) bool {
	const prefix = "CADefaultDeviceAggregate"
	return strings.HasPrefix(d.UID, prefix) || strings.HasPrefix(d.Name, prefix)
}

// deviceRecords builds the shipped records for one direction: every device's
// own name, plus its short name when no other device in that direction has
// the same short name or full name. A short name two devices share would
// match one of them arbitrarily, so neither gets it.
//
// Two devices with the same full name (two identical USB headsets) keep only
// the first: a record id is the spoken name, and one name can only mean one
// device. The other is still reachable by its uid from a keybind or script.
func deviceRecords(devices []branchkit.AudioDevice, dir direction) []deviceRecord {
	var devs []branchkit.AudioDevice
	for _, d := range devices {
		if dir.has(d) && !isAutoAggregate(d) && spokenForm(d.Name) != "" {
			devs = append(devs, d)
		}
	}

	// How many devices each candidate name could mean.
	claims := map[string]int{}
	for _, d := range devs {
		claims[spokenForm(d.Name)]++
		if s := shortName(d.Name); s != "" && s != spokenForm(d.Name) {
			claims[s]++
		}
	}

	var out []deviceRecord
	seen := map[string]bool{}
	add := func(spoken string, d branchkit.AudioDevice) {
		if spoken == "" || seen[spoken] {
			return
		}
		seen[spoken] = true
		out = append(out, deviceRecord{Spoken: spoken, UID: d.UID, Name: d.Name})
	}
	for _, d := range devs {
		add(spokenForm(d.Name), d)
	}
	for _, d := range devs {
		if s := shortName(d.Name); s != "" && claims[s] == 1 {
			add(s, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Spoken < out[j].Spoken })
	return out
}

// pushDeviceCollections publishes the current devices as speakable names.
// Wired to OnReady, hotplug and wake: a collection backed by live OS state
// re-publishes whenever that state can have changed. The replace is
// idempotent (identical records are skipped platform-side), so a burst of
// device events does not churn the grammar.
//
// The replace covers only this plugin's own records. A name the user added
// lives in the user band and survives every replace, including one where its
// device is unplugged.
func (h *Host) pushDeviceCollections() {
	devices, err := h.plugin.NativeAudioDevices()
	if err != nil {
		branchkit.Logf("audio", "audio devices: %v", err)
		return
	}
	for _, dir := range []direction{output, input} {
		records := deviceRecords(devices, dir)
		entries := make([]branchkit.CollectionPutEntry, 0, len(records))
		for _, r := range records {
			raw, err := json.Marshal(r)
			if err != nil {
				branchkit.Logf("audio", "%s: marshal %q: %v", dir.collection, r.Spoken, err)
				return
			}
			entries = append(entries, branchkit.CollectionPutEntry{ID: r.Spoken, Payload: raw})
		}
		if _, err := h.plugin.Replace(dir.collection, entries, branchkit.ScopeCollection()); err != nil {
			branchkit.Logf("audio", "replace %s: %v", dir.collection, err)
		}
	}
}

// loadNames reads a device collection as it stands, with the user's
// additions and removals applied. A failed read is an error, not an empty
// list: the two mean different things to every caller.
func (h *Host) loadNames(dir direction) ([]deviceRecord, error) {
	records, err := h.plugin.ListAll(dir.collection)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir.collection, err)
	}
	out := make([]deviceRecord, 0, len(records))
	for _, rec := range records {
		var r deviceRecord
		if err := json.Unmarshal(rec.Payload, &r); err != nil || r.Spoken == "" || r.UID == "" {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// resolveDevice finds the device a request means. `want` is usually a uid —
// that is the value a spoken name resolves to — but a keybind or script may
// pass a name, so names are accepted too, most exact first:
//
//  1. a connected device's uid;
//  2. a connected device's name, whole;
//  3. a name in the collection (a short name or one the user added);
//  4. part of exactly one connected device's name.
//
// A partial name that fits more than one device is refused, naming them,
// rather than picking whichever the OS listed first. A name that resolves to
// a device that is not connected says so.
func resolveDevice(devices []branchkit.AudioDevice, names []deviceRecord, want string, dir direction) (branchkit.AudioDevice, error) {
	var devs []branchkit.AudioDevice
	for _, d := range devices {
		if dir.has(d) && !isAutoAggregate(d) {
			devs = append(devs, d)
		}
	}
	if strings.TrimSpace(want) == "" {
		return branchkit.AudioDevice{}, fmt.Errorf("no %s device named", dir.deviceType)
	}
	for _, d := range devs {
		if d.UID == want {
			return d, nil
		}
	}
	spoken := spokenForm(want)
	for _, d := range devs {
		if spokenForm(d.Name) == spoken {
			return d, nil
		}
	}
	for _, r := range names {
		if r.Spoken != spoken && r.UID != want {
			continue
		}
		for _, d := range devs {
			if d.UID == r.UID {
				return d, nil
			}
		}
		label := r.Name
		if label == "" {
			label = r.Spoken
		}
		return branchkit.AudioDevice{}, fmt.Errorf("%s is not connected", label)
	}
	var partial []branchkit.AudioDevice
	for _, d := range devs {
		if strings.Contains(spokenForm(d.Name), spoken) {
			partial = append(partial, d)
		}
	}
	switch len(partial) {
	case 1:
		return partial[0], nil
	case 0:
		return branchkit.AudioDevice{}, fmt.Errorf("no %s device called %q", dir.deviceType, want)
	default:
		names := make([]string, len(partial))
		for i, d := range partial {
			names[i] = d.Name
		}
		return branchkit.AudioDevice{}, fmt.Errorf("%q could be any of: %s", want, strings.Join(names, ", "))
	}
}

// setDevice makes the device `want` names the default for one direction.
func (h *Host) setDevice(want string, dir direction) error {
	devices, err := h.plugin.NativeAudioDevices()
	if err != nil {
		return fmt.Errorf("list audio devices: %w", err)
	}
	// A failed read of the names only loses step 3 of the lookup; the device
	// names themselves still resolve, so log it and carry on.
	names, err := h.loadNames(dir)
	if err != nil {
		branchkit.Logf("audio", "set %s: %v", dir.deviceType, err)
	}
	d, err := resolveDevice(devices, names, want, dir)
	if err != nil {
		return err
	}
	if err := h.plugin.NativeSetAudioDevice(branchkit.NativeSetAudioDeviceRequest{DeviceType: dir.deviceType, UID: d.UID}); err != nil {
		return fmt.Errorf("switch %s to %s: %w", dir.deviceType, d.Name, err)
	}
	return nil
}
