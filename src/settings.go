package main

import (
	"sort"

	"github.com/branchkit/plugin-sdk-go"
)

type soundSettingsData struct {
	Volume      int
	VolumeMinus int
	VolumePlus  int
	Muted       bool
	Outputs     []deviceView
	Inputs      []deviceView
}

// deviceView is one row of the Sound tab.
type deviceView struct {
	UID       string
	Name      string
	IsDefault bool
	// Say is every name that selects the device, as the collection holds
	// them: its own name, its short name, the user's aliases. Read from the
	// collection, not rebuilt here, so the tab shows exactly what can be said
	// — including the user's edits, after a restart as much as before.
	Say []string
	// Removable is the subset of Say the user can take away: everything but
	// the device's own name.
	Removable map[string]bool
}

func (h *Host) renderSoundTab(_ *branchkit.RenderSettingsRequest) (string, error) {
	vol, muted, err := h.getVolume()
	if err != nil {
		branchkit.Logf("audio", "sound tab: %v", err)
	}
	devices, err := h.plugin.NativeAudioDevices()
	if err != nil {
		branchkit.Logf("audio", "sound tab: list audio devices: %v", err)
	}
	data := soundSettingsData{Muted: muted}
	data.Volume, data.VolumeMinus, data.VolumePlus = volumeSteps(vol)
	for _, dir := range []direction{output, input} {
		names, err := h.loadNames(dir)
		if err != nil {
			branchkit.Logf("audio", "sound tab: %v", err)
		}
		views := deviceViews(devices, names, dir)
		if dir.deviceType == "output" {
			data.Outputs = views
		} else {
			data.Inputs = views
		}
	}
	return branchkit.RenderComponent(Sound(data))
}

// volumeSteps returns the volume as a percentage and the percentages one step
// down and up, clamped, for the stepper buttons.
func volumeSteps(vol float64) (cur, minus, plus int) {
	cur = int(vol*100 + 0.5)
	step := int(volumeStep * 100)
	return cur, max(cur-step, 0), min(cur+step, 100)
}

// deviceViews joins the connected devices of one direction with the names
// that select them.
func deviceViews(devices []branchkit.AudioDevice, names []deviceRecord, dir direction) []deviceView {
	var out []deviceView
	for _, d := range devices {
		if !dir.has(d) || isAutoAggregate(d) {
			continue
		}
		v := deviceView{UID: d.UID, Name: d.Name, Removable: map[string]bool{}}
		if dir.deviceType == "output" {
			v.IsDefault = d.IsDefaultOutput
		} else {
			v.IsDefault = d.IsDefaultInput
		}
		own := spokenForm(d.Name)
		for _, r := range names {
			if r.UID != d.UID {
				continue
			}
			v.Say = append(v.Say, r.Spoken)
			if r.Spoken != own {
				v.Removable[r.Spoken] = true
			}
		}
		// The shortest first: it is the one a person reaches for.
		sort.SliceStable(v.Say, func(i, j int) bool { return len(v.Say[i]) < len(v.Say[j]) })
		out = append(out, v)
	}
	return out
}

// --- Settings commands ---

type setVolumeRequest struct {
	Volume int `json:"volume"`
}

func (h *Host) handleSetVolumeCommand(req *setVolumeRequest) error {
	return h.setVolume(float64(req.Volume) / 100.0)
}

type setMuteRequest struct {
	Muted bool `json:"muted"`
}

func (h *Host) handleSetMuteCommand(req *setMuteRequest) error {
	return h.setMuted(req.Muted)
}

type setDeviceRequest struct {
	UID        string `json:"uid"`
	DeviceType string `json:"device_type"`
}

func (h *Host) handleSetDeviceCommand(req *setDeviceRequest) error {
	dir := output
	if req.DeviceType == "input" {
		dir = input
	}
	return h.setDevice(req.UID, dir)
}

type deviceAliasRequest struct {
	UID   string `json:"uid"`
	Alias string `json:"alias"`
}

func (h *Host) handleDeviceAliasAdd(req *deviceAliasRequest) error {
	return h.addDeviceAlias(req.UID, req.Alias)
}

func (h *Host) handleDeviceAliasRemove(req *deviceAliasRequest) error {
	return h.removeDeviceAlias(req.UID, req.Alias)
}
