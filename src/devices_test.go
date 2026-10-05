package main

import (
	"strings"
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

var (
	speakers   = branchkit.AudioDevice{UID: "BuiltInSpeakerDevice", Name: "MacBook Air Speakers", IsOutput: true, IsDefaultOutput: true}
	builtInMic = branchkit.AudioDevice{UID: "BuiltInMicrophoneDevice", Name: "MacBook Air Microphone", IsInput: true, IsDefaultInput: true}
	headset    = branchkit.AudioDevice{UID: "usb-headset", Name: "External Headphones", IsOutput: true, IsInput: true}
	aggregate  = branchkit.AudioDevice{UID: "CADefaultDeviceAggregate-80574-0", Name: "CADefaultDeviceAggregate-80574-0", IsOutput: true}
)

func TestShortName(t *testing.T) {
	for name, want := range map[string]string{
		"MacBook Air Speakers":        "speakers",
		"MacBook Pro Microphone":      "microphone",
		"Built-in Output":             "output",
		"External Headphones":         "headphones",
		"Speakers (Realtek(R) Audio)": "speakers",
		"Sony WH-1000XM5":             "",
		"AirPods Pro":                 "",
		"MacBook":                     "",
	} {
		if got := shortName(name); got != want {
			t.Errorf("shortName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestIsAutoAggregate(t *testing.T) {
	for _, tc := range []struct {
		d    branchkit.AudioDevice
		want bool
	}{
		{aggregate, true},
		{branchkit.AudioDevice{Name: "Aggregate", UID: "CADefaultDeviceAggregate-1-0"}, true},
		{speakers, false},
		{branchkit.AudioDevice{Name: "My Aggregate Device", UID: "~:AMS2_Aggregate:0"}, false},
	} {
		if got := isAutoAggregate(tc.d); got != tc.want {
			t.Errorf("isAutoAggregate(%q, %q) = %v, want %v", tc.d.Name, tc.d.UID, got, tc.want)
		}
	}
}

func spokenToUID(records []deviceRecord) map[string]string {
	out := map[string]string{}
	for _, r := range records {
		out[r.Spoken] = r.UID
	}
	return out
}

func TestDeviceRecords_NamesAndShortNames(t *testing.T) {
	got := spokenToUID(deviceRecords([]branchkit.AudioDevice{speakers, builtInMic, headset, aggregate}, output))
	want := map[string]string{
		"macbook air speakers": speakers.UID,
		"speakers":             speakers.UID,
		"external headphones":  headset.UID,
		"headphones":           headset.UID,
	}
	if len(got) != len(want) {
		t.Fatalf("outputs = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("outputs[%q] = %q, want %q", k, got[k], v)
		}
	}
	// The headset carries sound both ways, so it is an input too.
	inputs := spokenToUID(deviceRecords([]branchkit.AudioDevice{speakers, builtInMic, headset}, input))
	if inputs["headphones"] != headset.UID || inputs["microphone"] != builtInMic.UID || inputs["speakers"] != "" {
		t.Errorf("inputs = %v", inputs)
	}
}

// A short name two devices share would pick one arbitrarily, so neither
// gets it; each keeps its full name.
func TestDeviceRecords_AmbiguousShortNameDropped(t *testing.T) {
	a := branchkit.AudioDevice{UID: "a", Name: "Speakers (Realtek(R) Audio)", IsOutput: true}
	b := branchkit.AudioDevice{UID: "b", Name: "Speakers (USB Audio)", IsOutput: true}
	got := spokenToUID(deviceRecords([]branchkit.AudioDevice{a, b}, output))
	if _, ok := got["speakers"]; ok {
		t.Errorf("an ambiguous short name was published: %v", got)
	}
	if got["speakers (realtek(r) audio)"] != "a" || got["speakers (usb audio)"] != "b" {
		t.Errorf("full names missing: %v", got)
	}
}

// A short name that is another device's full name would make that device's
// own name ambiguous.
func TestDeviceRecords_ShortNameCollidingWithFullName(t *testing.T) {
	a := branchkit.AudioDevice{UID: "a", Name: "MacBook Pro Speakers", IsOutput: true}
	b := branchkit.AudioDevice{UID: "b", Name: "Speakers", IsOutput: true}
	got := spokenToUID(deviceRecords([]branchkit.AudioDevice{a, b}, output))
	if got["speakers"] != "b" {
		t.Errorf(`"speakers" should be device b's own name, got %v`, got)
	}
}

func TestResolveDevice(t *testing.T) {
	devices := []branchkit.AudioDevice{speakers, headset, builtInMic}
	names := []deviceRecord{
		{Spoken: "cans", UID: headset.UID, Name: headset.Name},
		{Spoken: "desk speakers", UID: "unplugged", Name: "Desk Speakers"},
	}
	for _, tc := range []struct {
		want    string
		dir     direction
		uid     string
		errPart string
	}{
		{want: headset.UID, dir: output, uid: headset.UID},
		{want: "external headphones", dir: output, uid: headset.UID},
		{want: "External Headphones", dir: input, uid: headset.UID},
		{want: "cans", dir: output, uid: headset.UID},
		{want: "air speakers", dir: output, uid: speakers.UID},
		{want: "desk speakers", dir: output, errPart: "Desk Speakers is not connected"},
		{want: "unplugged", dir: output, errPart: "not connected"},
		{want: "kazoo", dir: output, errPart: `no output device called "kazoo"`},
		{want: "", dir: output, errPart: "no output device named"},
		// A microphone is not an output, whatever its name.
		{want: builtInMic.UID, dir: output, errPart: "no output device"},
		// "a" is in both output names: refused, naming both.
		{want: "a", dir: output, errPart: "could be any of"},
	} {
		d, err := resolveDevice(devices, names, tc.want, tc.dir)
		if tc.errPart != "" {
			if err == nil || !strings.Contains(err.Error(), tc.errPart) {
				t.Errorf("resolve(%q, %s): err = %v, want containing %q", tc.want, tc.dir.deviceType, err, tc.errPart)
			}
			continue
		}
		if err != nil || d.UID != tc.uid {
			t.Errorf("resolve(%q, %s) = %q, %v; want %q", tc.want, tc.dir.deviceType, d.UID, err, tc.uid)
		}
	}
}

func TestSetDevice_SwitchesByResolvedUID(t *testing.T) {
	f := newFake(speakers, headset)
	h := newHost(f)
	h.pushDeviceCollections()
	if err := h.setDevice("headphones", output); err != nil {
		t.Fatal(err)
	}
	if len(f.setDevices) != 1 || f.setDevices[0].UID != headset.UID || f.setDevices[0].DeviceType != "output" {
		t.Fatalf("set device calls = %+v", f.setDevices)
	}
}

// The action reports a failure instead of claiming success.
func TestSetOutputAction_ReturnsError(t *testing.T) {
	h := newHost(newFake(speakers))
	if _, err := h.handleSetOutput(SetOutputParams{Device: "kazoo"}, nil); err == nil {
		t.Fatal("expected an error for a device that does not exist")
	}
}

func TestVolumeStep_UnmutesAndClamps(t *testing.T) {
	f := newFake()
	f.volume, f.muted = 0.97, true
	h := newHost(f)
	if _, err := h.handleVolumeUp(nil); err != nil {
		t.Fatal(err)
	}
	if f.muted || f.volume != 1 {
		t.Errorf("after volume up: volume %v muted %v, want 1 and unmuted", f.volume, f.muted)
	}
	f.volume = 0.03
	if _, err := h.handleVolumeDown(nil); err != nil {
		t.Fatal(err)
	}
	if f.volume != 0 {
		t.Errorf("after volume down: volume %v, want 0", f.volume)
	}
}

func TestVolumeSteps(t *testing.T) {
	for _, tc := range []struct {
		vol              float64
		cur, minus, plus int
	}{
		{0.5, 50, 43, 57},
		{0.03, 3, 0, 10},
		{0.98, 98, 91, 100},
		{0.576, 58, 51, 65},
	} {
		cur, minus, plus := volumeSteps(tc.vol)
		if cur != tc.cur || minus != tc.minus || plus != tc.plus {
			t.Errorf("volumeSteps(%v) = %d,%d,%d; want %d,%d,%d", tc.vol, cur, minus, plus, tc.cur, tc.minus, tc.plus)
		}
	}
}
