package main

import "github.com/branchkit/plugin-sdk-go"

// Per-action handlers. Each returns its error: a failed action that reports
// success is invisible to whoever ran it — the person who spoke, a script's
// dispatch() result, a keybind — and to every report built on dispatch.
//
// Param structs (SetOutputParams, …) are generated into actions_gen.go from
// plugin.json's action_types; edit the manifest and run branchkit-gen, never
// the structs.

func (h *Host) handleVolumeUp(_ *branchkit.OnActionRequest) (any, error) {
	return nil, h.stepVolume(volumeStep)
}

func (h *Host) handleVolumeDown(_ *branchkit.OnActionRequest) (any, error) {
	return nil, h.stepVolume(-volumeStep)
}

func (h *Host) handleMute(_ *branchkit.OnActionRequest) (any, error) {
	return nil, h.setMuted(true)
}

func (h *Host) handleUnmute(_ *branchkit.OnActionRequest) (any, error) {
	return nil, h.setMuted(false)
}

func (h *Host) handleSetOutput(p SetOutputParams, _ *branchkit.OnActionRequest) (any, error) {
	return nil, h.setDevice(p.Device, output)
}

func (h *Host) handleSetInput(p SetInputParams, _ *branchkit.OnActionRequest) (any, error) {
	return nil, h.setDevice(p.Device, input)
}
