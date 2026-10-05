# BranchKit Audio

Volume, mute, and which audio devices you hear and speak through, by name. A
plugin for [BranchKit](https://github.com/branchkit), an accessibility plugin
platform for the desktop. MIT licensed.

BranchKit is pre-launch: the app is not publicly released yet.

## What you can say

The phrases come from `commands.json`. Any command can also be bound to a key
in Settings → Keybinds.

| Say | Does |
|---|---|
| `volume up` / `volume down` | Raise or lower the output volume, unmuting it if it was muted |
| `mute audio` / `unmute audio` | Mute or unmute the output |
| `set output <device>` / `switch output <device>` | Make a device the output |
| `set input <device>` / `switch input <device>` | Make a device the input |

A device answers to its own name, to a shorter one when no other device
would share it ("speakers" for "MacBook Air Speakers", "headphones" for
"External Headphones", "speakers" for "Speakers (Realtek(R) Audio)"), and to
any name you add on the Sound tab. The Sound tab lists every name each device
answers to, written as the phrase to say.

## Actions

| Action | Does |
|---|---|
| `audio.volume_up` / `audio.volume_down` | Step the output volume by about 7%, unmuting first |
| `audio.mute` / `audio.unmute` | Mute or unmute the output |
| `audio.set_output` / `audio.set_input` | Make a device the default. `device` is the device's id, its name, or any name it answers to |

Every action reports failure: a device that is not connected, a name that
could mean two devices, or a refused platform call comes back as an error,
never as success.

## Collections

`audio_outputs` and `audio_inputs` hold one record per name you can say,
`{spoken, uid, name}`. The matcher hands the action the device's `uid`, so the
action never guesses from a name.

- This plugin publishes each connected device's own name and short name, and
  publishes again on hotplug and on wake (devices come and go while the
  machine sleeps, when nothing is watching).
- A name you add is a user-band record, written through the platform's
  collection overrides. It survives every republish, including while its
  device is unplugged: plug the device back in and the name works again.
- Removing a name you added deletes it. Removing a short name this plugin
  publishes suppresses it, and adding it back clears the suppression. Your
  edits hold only what differs from what the plugin publishes.

Other plugins may read both collections: they are what `<device:audio_outputs>`
captures from.

## Settings

**Sound**: volume, mute, the output and input devices with the names each
answers to, a button to make a device the default, and an editor for names.

## Permissions

| Privilege | Why |
|---|---|
| `audio` | Read and set volume, mute and audio devices |

No network, no files, no input: the manifest declares none, so the sandbox
gives it none.

## Platform support

Every action works on macOS, Linux and Windows; the platform implements the
volume and device operations underneath on each.

## Reading this as an example

- `src/devices.go` publishes a collection backed by live OS state and
  re-publishes it on the events that can change it.
- `src/aliases.go` is the pattern for letting a person edit a collection a
  plugin publishes: user-band overrides, restore before add or remove.
- `src/host.go` puts every platform call behind a small interface, so
  `src/fake_test.go` can test the plugin's logic without a running app.

## Build

Go, [plugin-sdk-go](https://github.com/branchkit/plugin-sdk-go). The settings
tab is a [templ](https://templ.guide) template; the generated Go is committed,
so `templ` is needed only when you edit `sound.templ`.

```bash
cd src && go build -o ../audio-plugin . && go test ./...
```

## License

MIT. See [LICENSE](LICENSE).
