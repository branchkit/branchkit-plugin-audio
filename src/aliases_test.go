package main

import (
	"strings"
	"testing"
)

// An alias the user adds is a record in the collection, so it can be said,
// and it names the device in every direction the device carries sound.
func TestAddDeviceAlias_SayableBothWays(t *testing.T) {
	f := newFake(speakers, headset)
	h := newHost(f)
	h.pushDeviceCollections()

	if err := h.addDeviceAlias(headset.UID, "  Cans "); err != nil {
		t.Fatal(err)
	}
	if got := namesOf(t, h, output)["cans"]; got != headset.UID {
		t.Errorf("outputs[cans] = %q, want the headset", got)
	}
	if got := namesOf(t, h, input)["cans"]; got != headset.UID {
		t.Errorf("inputs[cans] = %q, want the headset", got)
	}
	// And it resolves when spoken.
	if err := h.setDevice(headset.UID, output); err != nil {
		t.Fatal(err)
	}
}

// The hotplug replace only touches the plugin's records, so a user alias
// survives the device being unplugged and plugged back in.
func TestAddDeviceAlias_SurvivesReplace(t *testing.T) {
	f := newFake(speakers, headset)
	h := newHost(f)
	h.pushDeviceCollections()
	if err := h.addDeviceAlias(headset.UID, "cans"); err != nil {
		t.Fatal(err)
	}
	f.devices = nil
	h.pushDeviceCollections()
	if err := h.setDevice("cans", output); err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Errorf("unplugged: err = %v, want 'not connected'", err)
	}
	f.devices = append(f.devices, speakers, headset)
	h.pushDeviceCollections()
	if err := h.setDevice("cans", output); err != nil {
		t.Errorf("plugged back in: %v", err)
	}
}

func TestAddDeviceAlias_RefusesAnotherDevicesName(t *testing.T) {
	f := newFake(speakers, headset)
	h := newHost(f)
	h.pushDeviceCollections()
	err := h.addDeviceAlias(headset.UID, "speakers")
	if err == nil || !strings.Contains(err.Error(), "MacBook Air Speakers") {
		t.Fatalf("err = %v, want a refusal naming the other device", err)
	}
	if f.userBandSize() != 0 {
		t.Errorf("a refused alias left user-band state: %d entries", f.userBandSize())
	}
}

func TestAddDeviceAlias_EmptyRefused(t *testing.T) {
	h := newHost(newFake(headset))
	if err := h.addDeviceAlias(headset.UID, "   "); err == nil {
		t.Fatal("expected an empty alias to be refused")
	}
}

// Removing a name the user added deletes it outright: the user band ends
// empty, not holding an addition and a removal on top of each other.
func TestRemoveDeviceAlias_UserAddedLeavesNothing(t *testing.T) {
	f := newFake(speakers, headset)
	h := newHost(f)
	h.pushDeviceCollections()
	if err := h.addDeviceAlias(headset.UID, "cans"); err != nil {
		t.Fatal(err)
	}
	if err := h.removeDeviceAlias(headset.UID, "cans"); err != nil {
		t.Fatal(err)
	}
	if _, ok := namesOf(t, h, output)["cans"]; ok {
		t.Error("removed alias still in outputs")
	}
	if f.userBandSize() != 0 {
		t.Errorf("user band holds %d entries after add+remove, want 0", f.userBandSize())
	}
}

// A short name the plugin publishes is suppressed when removed, and putting
// it back clears the suppression rather than storing a private copy.
func TestRemoveThenAddShortName_RoundTripsToNothing(t *testing.T) {
	f := newFake(speakers, headset)
	h := newHost(f)
	h.pushDeviceCollections()
	if err := h.removeDeviceAlias(headset.UID, "headphones"); err != nil {
		t.Fatal(err)
	}
	if _, ok := namesOf(t, h, output)["headphones"]; ok {
		t.Fatal("short name still published after removal")
	}
	if err := h.addDeviceAlias(headset.UID, "headphones"); err != nil {
		t.Fatal(err)
	}
	if namesOf(t, h, output)["headphones"] != headset.UID {
		t.Fatal("short name not back after re-adding")
	}
	if f.userBandSize() != 0 {
		t.Errorf("user band holds %d entries, want 0: the name is back the way it shipped", f.userBandSize())
	}
}

func TestRemoveDeviceAlias_OwnNameRefused(t *testing.T) {
	f := newFake(headset)
	h := newHost(f)
	h.pushDeviceCollections()
	if err := h.removeDeviceAlias(headset.UID, "External Headphones"); err == nil {
		t.Fatal("expected the device's own name to be refused")
	}
}
