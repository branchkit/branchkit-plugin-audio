package main

import "github.com/branchkit/plugin-sdk-go"

// platform is every call this plugin makes on the platform, so tests can
// stand in for it. *branchkit.Plugin satisfies it.
type platform interface {
	NativeAudioDevices() ([]branchkit.AudioDevice, error)
	NativeSetAudioDevice(branchkit.NativeSetAudioDeviceRequest) error
	NativeVolume() (*branchkit.NativeVolumeResponse, error)
	NativeSetVolume(branchkit.NativeSetVolumeRequest) error
	NativeMute(branchkit.NativeMuteRequest) error
	ListAll(name string) ([]branchkit.CollectionRecord, error)
	Replace(name string, entries []branchkit.CollectionPutEntry, scope branchkit.ReplaceScope, opts ...branchkit.ReplaceOption) (branchkit.ReplaceResult, error)
	OverridesApply(branchkit.OverridesApplyRequest) (*branchkit.OverridesApplyResponse, error)
}

// Host is what every handler needs. Handlers are methods on it, so a
// handler's dependencies are visible in its signature.
type Host struct {
	plugin platform
}

func newHost(p platform) *Host { return &Host{plugin: p} }
