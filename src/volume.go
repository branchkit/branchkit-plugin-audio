package main

import (
	"fmt"

	"github.com/branchkit/plugin-sdk-go"
)

const volumeStep = 0.07 // ~7% per step

// getVolume returns the output volume (0.0–1.0) and whether it is muted.
func (h *Host) getVolume() (float64, bool, error) {
	resp, err := h.plugin.NativeVolume()
	if err != nil {
		return 0, false, fmt.Errorf("read volume: %w", err)
	}
	return resp.Volume, resp.IsMuted, nil
}

// setVolume sets the output volume, clamped to 0.0–1.0.
func (h *Host) setVolume(vol float64) error {
	return h.plugin.NativeSetVolume(branchkit.NativeSetVolumeRequest{Volume: clampVolume(vol)})
}

func clampVolume(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// stepVolume moves the volume one step and unmutes, the way a keyboard's
// volume keys do: changing the volume of a muted output and still hearing
// nothing is the surprise, not the unmute.
func (h *Host) stepVolume(delta float64) error {
	vol, muted, err := h.getVolume()
	if err != nil {
		return err
	}
	if muted {
		if err := h.setMuted(false); err != nil {
			return err
		}
	}
	return h.setVolume(vol + delta)
}

func (h *Host) setMuted(muted bool) error {
	return h.plugin.NativeMute(branchkit.NativeMuteRequest{Muted: muted})
}
