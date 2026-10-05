package main

import (
	"encoding/json"
	"fmt"

	"github.com/branchkit/plugin-sdk-go"
)

// The alias editor: names the user adds for a device, and names they take
// away.
//
// An alias is a record in the device collection like any other, written as a
// platform OVERRIDE in the user band (`_user`), never by this plugin into its
// own records. Three things follow from that. The name is in the grammar, so
// it can be said: matching a spoken name against the collection is the only
// path to the action, and a name kept anywhere else is unreachable. It
// survives the replace that runs on every hotplug, which only touches this
// plugin's records. And it is the user's, visible and removable in the
// Collections page like every other edit.
//
// Both edits RESTORE FIRST. `restore` drops all user-band state for one id,
// so removing a name the user added deletes it outright, and adding back a
// shipped name the user had removed un-suppresses it instead of storing a
// private copy that would stop tracking what ships. The user band then holds
// only what genuinely differs from what this plugin publishes.

// addDeviceAlias makes `alias` a name for the device, in every direction the
// device carries sound: an alias added to a headset names it as an output and
// as an input.
func (h *Host) addDeviceAlias(uid, rawAlias string) error {
	alias := spokenForm(rawAlias)
	if alias == "" {
		return fmt.Errorf("an alias needs a name")
	}
	d, err := h.connectedDevice(uid)
	if err != nil {
		return err
	}
	for _, dir := range []direction{output, input} {
		if !dir.has(d) {
			continue
		}
		names, err := h.loadNames(dir)
		if err != nil {
			return err
		}
		// A name already meaning another device would make one of the two
		// unreachable by voice. Refuse it, naming the device that has it.
		if other, ok := findName(names, alias); ok && other.UID != uid {
			label := other.Name
			if label == "" {
				label = "another device"
			}
			return fmt.Errorf("%q already names %s", alias, label)
		}
		if err := userBandOverride(h.plugin, dir.collection, "restore", alias, nil); err != nil {
			branchkit.Logf("audio", "alias restore %q (may be a no-op): %v", alias, err)
		}
		if names, err := h.loadNames(dir); err == nil {
			if r, ok := findName(names, alias); ok && r.UID == uid {
				continue
			}
		}
		if err := userBandOverride(h.plugin, dir.collection, "add", "",
			deviceRecord{Spoken: alias, UID: uid, Name: d.Name}); err != nil {
			return fmt.Errorf("could not add %q: %w", alias, err)
		}
	}
	return nil
}

// removeDeviceAlias stops `alias` naming the device, in every direction.
// A device's own name cannot be removed this way: it is what the Sound tab
// lists the device by, and the only name a fresh install knows.
func (h *Host) removeDeviceAlias(uid, rawAlias string) error {
	alias := spokenForm(rawAlias)
	for _, dir := range []direction{output, input} {
		names, err := h.loadNames(dir)
		if err != nil {
			return err
		}
		r, ok := findName(names, alias)
		if !ok || r.UID != uid {
			continue
		}
		if spokenForm(r.Name) == alias {
			return fmt.Errorf("%q is the device's own name", alias)
		}
		if err := userBandOverride(h.plugin, dir.collection, "restore", alias, nil); err != nil {
			branchkit.Logf("audio", "alias restore %q (may be a no-op): %v", alias, err)
		}
		// Gone: it was the user's own, and restoring deleted it.
		if names, err := h.loadNames(dir); err == nil {
			if _, still := findName(names, alias); !still {
				continue
			}
		}
		// Still there, so this plugin publishes it (a short name). Suppress it.
		if err := userBandOverride(h.plugin, dir.collection, "remove", alias, nil); err != nil {
			return fmt.Errorf("could not remove %q: %w", alias, err)
		}
	}
	return nil
}

func findName(names []deviceRecord, spoken string) (deviceRecord, bool) {
	for _, r := range names {
		if r.Spoken == spoken {
			return r, true
		}
	}
	return deviceRecord{}, false
}

// connectedDevice returns the connected device with this uid.
func (h *Host) connectedDevice(uid string) (branchkit.AudioDevice, error) {
	devices, err := h.plugin.NativeAudioDevices()
	if err != nil {
		return branchkit.AudioDevice{}, fmt.Errorf("list audio devices: %w", err)
	}
	for _, d := range devices {
		if d.UID == uid {
			return d, nil
		}
	}
	return branchkit.AudioDevice{}, fmt.Errorf("that device is no longer connected")
}

// userBandOverride relays one user gesture into the user band of one of this
// plugin's collections: by record id, or with a whole record to add.
func userBandOverride(p platform, collection, action, id string, record any) error {
	tenant := "_user"
	var idPtr *string
	if id != "" {
		idPtr = &id
	}
	var raw json.RawMessage
	if record != nil {
		b, err := json.Marshal(record)
		if err != nil {
			return err
		}
		raw = b
	}
	_, err := p.OverridesApply(branchkit.OverridesApplyRequest{Action: action, Collection: collection, Fields: raw, ID: idPtr, Tenant: &tenant})
	return err
}
