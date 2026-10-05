package main

import (
	_ "embed"
	"encoding/json"

	"github.com/branchkit/plugin-sdk-go"
)

//go:embed settings.css
var settingsCSS string

func main() {
	p := branchkit.NewPlugin()
	h := newHost(p)

	// Registrars come from actions_gen.go, generated from plugin.json, so no
	// action string is spelled here and a params type cannot drift from the
	// manifest.
	HandleVolumeUp(p, h.handleVolumeUp)
	HandleVolumeDown(p, h.handleVolumeDown)
	HandleMute(p, h.handleMute)
	HandleUnmute(p, h.handleUnmute)
	HandleSetOutput(p, h.handleSetOutput)
	HandleSetInput(p, h.handleSetInput)

	p.SettingsCSS(settingsCSS)
	p.SettingsTab("sound", h.renderSoundTab)
	branchkit.HandleCommand(p, "set_volume", h.handleSetVolumeCommand)
	branchkit.HandleCommand(p, "set_mute", h.handleSetMuteCommand)
	branchkit.HandleCommand(p, "set_device", h.handleSetDeviceCommand)
	branchkit.HandleCommand(p, "device_alias_add", h.handleDeviceAliasAdd)
	branchkit.HandleCommand(p, "device_alias_remove", h.handleDeviceAliasRemove)

	// The device collections track the live device set: published once RPC
	// is up, again on hotplug, and again on wake, because devices come and go
	// while the machine sleeps (a dock unplugged, headphones carried out of
	// range) and the listener behind audio_devices.changed is not running to
	// see it.
	p.OnReady(h.pushDeviceCollections)
	p.On("_platform.audio_devices.changed", func(json.RawMessage) { h.pushDeviceCollections() })
	p.On("_platform.system.did_wake", func(json.RawMessage) { h.pushDeviceCollections() })

	p.Run()
}
