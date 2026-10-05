package main

import (
	"github.com/branchkit/plugin-sdk-go"

	"strings"
	"testing"
)

// The tab lists the names that select each device as the collection holds
// them — the user's own aliases included — and only those can be removed.
func TestRenderSoundTab_ShowsSayableNames(t *testing.T) {
	f := newFake(speakers, headset, builtInMic)
	f.volume = 0.5
	h := newHost(f)
	h.pushDeviceCollections()
	if err := h.addDeviceAlias(headset.UID, "cans"); err != nil {
		t.Fatal(err)
	}
	html, err := h.renderSoundTab(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"set output speakers",
		"set output cans",
		"set input cans",
		"set input microphone",
		"50%",
		`title="Make External Headphones the default output"`,
		`title="Stop &#34;cans&#34; naming External Headphones"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("sound tab missing %q", want)
		}
	}
	if strings.Contains(html, `Stop &#34;external headphones&#34;`) {
		t.Error("a device's own name must not offer removal")
	}
}

func TestDeviceViews_ShortestNameFirst(t *testing.T) {
	names := deviceRecords(nil, output)
	names = append(names,
		deviceRecord{Spoken: "external headphones", UID: headset.UID},
		deviceRecord{Spoken: "headphones", UID: headset.UID},
	)
	v := deviceViews([]branchkit.AudioDevice{headset}, names, output)
	if len(v) != 1 || len(v[0].Say) != 2 || v[0].Say[0] != "headphones" {
		t.Fatalf("views = %+v", v)
	}
	if !v[0].Removable["headphones"] || v[0].Removable["external headphones"] {
		t.Errorf("removable = %v", v[0].Removable)
	}
}

func TestSoundTempl_NoDevices(t *testing.T) {
	html := mustRender(t, Sound(soundSettingsData{VolumePlus: 7}))
	if strings.Contains(html, "Output devices") || strings.Contains(html, "Input devices") {
		t.Error("expected no device sections when there are no devices")
	}
}

func TestSoundTempl_MuteState(t *testing.T) {
	html := mustRender(t, Sound(soundSettingsData{Muted: true}))
	if !strings.Contains(html, `class="segmented-btn is-active" aria-pressed="true"`) {
		t.Errorf("expected the On button pressed when muted; got %s", html)
	}
}
