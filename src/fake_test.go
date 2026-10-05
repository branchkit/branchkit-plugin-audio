package main

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/branchkit/plugin-sdk-go"
)

// fakePlatform stands in for the platform: a device list, the volume, and the
// two device collections with the user band composed over the plugin's own
// records the way the platform's collection overrides compose them —
// `add` puts a record in the user band, `remove` suppresses an id, and
// `restore` drops ALL user-band state for an id (its additions and its
// suppressions alike).
type fakePlatform struct {
	devices []branchkit.AudioDevice
	volume  float64
	muted   bool

	base    map[string]map[string]deviceRecord // collection → id → plugin's record
	added   map[string]map[string]deviceRecord // collection → id → user-added record
	removed map[string]map[string]bool         // collection → id → suppressed

	setDevices []branchkit.NativeSetAudioDeviceRequest
	overrides  []branchkit.OverridesApplyRequest
}

func newFake(devices ...branchkit.AudioDevice) *fakePlatform {
	return &fakePlatform{
		devices: devices,
		base:    map[string]map[string]deviceRecord{},
		added:   map[string]map[string]deviceRecord{},
		removed: map[string]map[string]bool{},
	}
}

func (f *fakePlatform) NativeAudioDevices() ([]branchkit.AudioDevice, error) { return f.devices, nil }

func (f *fakePlatform) NativeSetAudioDevice(req branchkit.NativeSetAudioDeviceRequest) error {
	f.setDevices = append(f.setDevices, req)
	return nil
}

func (f *fakePlatform) NativeVolume() (*branchkit.NativeVolumeResponse, error) {
	return &branchkit.NativeVolumeResponse{Volume: f.volume, IsMuted: f.muted}, nil
}

func (f *fakePlatform) NativeSetVolume(req branchkit.NativeSetVolumeRequest) error {
	f.volume = req.Volume
	return nil
}

func (f *fakePlatform) NativeMute(req branchkit.NativeMuteRequest) error {
	f.muted = req.Muted
	return nil
}

func (f *fakePlatform) Replace(name string, entries []branchkit.CollectionPutEntry, _ branchkit.ReplaceScope, _ ...branchkit.ReplaceOption) (branchkit.ReplaceResult, error) {
	m := map[string]deviceRecord{}
	for _, e := range entries {
		var r deviceRecord
		if err := json.Unmarshal(e.Payload, &r); err != nil {
			return branchkit.ReplaceResult{}, err
		}
		m[e.ID] = r
	}
	f.base[name] = m
	return branchkit.ReplaceResult{}, nil
}

func (f *fakePlatform) ListAll(name string) ([]branchkit.CollectionRecord, error) {
	composed := map[string]deviceRecord{}
	for id, r := range f.base[name] {
		composed[id] = r
	}
	for id, r := range f.added[name] {
		composed[id] = r
	}
	for id := range f.removed[name] {
		delete(composed, id)
	}
	ids := make([]string, 0, len(composed))
	for id := range composed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]branchkit.CollectionRecord, 0, len(ids))
	for _, id := range ids {
		raw, _ := json.Marshal(composed[id])
		out = append(out, branchkit.CollectionRecord{ID: id, Payload: raw})
	}
	return out, nil
}

func (f *fakePlatform) OverridesApply(req branchkit.OverridesApplyRequest) (*branchkit.OverridesApplyResponse, error) {
	f.overrides = append(f.overrides, req)
	c := req.Collection
	if f.added[c] == nil {
		f.added[c] = map[string]deviceRecord{}
		f.removed[c] = map[string]bool{}
	}
	switch req.Action {
	case "add":
		var r deviceRecord
		if err := json.Unmarshal(req.Fields, &r); err != nil {
			return nil, err
		}
		f.added[c][r.Spoken] = r
	case "remove":
		f.removed[c][*req.ID] = true
	case "restore":
		delete(f.added[c], *req.ID)
		delete(f.removed[c], *req.ID)
	}
	return &branchkit.OverridesApplyResponse{}, nil
}

// userBandSize is how many entries the user band holds across both
// collections: the measure of "only what genuinely differs from what ships".
func (f *fakePlatform) userBandSize() int {
	n := 0
	for _, c := range f.added {
		n += len(c)
	}
	for _, c := range f.removed {
		n += len(c)
	}
	return n
}

func namesOf(t *testing.T, h *Host, dir direction) map[string]string {
	t.Helper()
	names, err := h.loadNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, r := range names {
		out[r.Spoken] = r.UID
	}
	return out
}

func mustRender(t *testing.T, c branchkit.HTMLComponent) string {
	t.Helper()
	html, err := branchkit.RenderComponent(c)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return html
}
