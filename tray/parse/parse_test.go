package parse

import (
	"path/filepath"
	"testing"
)

func TestParseINI_iRacingApp(t *testing.T) {
	got, err := ParseINI(filepath.Join("testdata", "iracing-app.ini"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := map[string]string{
		"Graphics.MultiSamples":  "4",
		"Graphics.MaxQuality":    "1",
		"Graphics.MirrorHigh":    "1",
		"Graphics.ShadowMaps":    "2",
		"Graphics.TextureQuality": "2",
		"Mirrors.MirrorScale":    "1.0",
		"VR.PixelDensity":        "1.4",
	}
	for k, want := range cases {
		if got[k] != want {
			t.Errorf("key %q: got %q, want %q", k, got[k], want)
		}
	}
}

func TestParseINI_PUBGUnreal(t *testing.T) {
	got, err := ParseINI(filepath.Join("testdata", "pubg-gameusersettings.ini"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Spot-check a key from each section, including the dotted-key Unreal style
	cases := map[string]string{
		"/Script/Engine.GameUserSettings.ResolutionSizeX":  "2560",
		"ScalabilityGroups.sg.TextureQuality":              "3",
		"ScalabilityGroups.sg.AntiAliasingQuality":         "2",
		"/Script/TslGame.TslGameUserSettings.AimSensitivity": "32.000000",
	}
	for k, want := range cases {
		if got[k] != want {
			t.Errorf("key %q: got %q, want %q", k, got[k], want)
		}
	}
}

func TestParseIRacingControls(t *testing.T) {
	got, err := ParseIRacingControls(filepath.Join("testdata", "iracing-controls.cfg"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cases := map[string]string{
		"Controls.Throttle":            "Fanatec axis 1",
		"Controls.Brake Bias increase": "vJoy_71",
		"Controls.Brake Bias decrease": "vJoy_72",
		"Controls.Black Box click":     "vJoy_56",
	}
	for k, want := range cases {
		if got[k] != want {
			t.Errorf("key %q: got %q, want %q", k, got[k], want)
		}
	}
}

func TestStripInlineComment(t *testing.T) {
	cases := map[string]string{
		"value":              "value",
		"value ; comment":    "value",
		"value # comment":    "value",
		"rgb(255;128;0)":     "rgb(255;128;0)", // no preceding whitespace, preserved
		"path\\#anchor":      "path\\#anchor",
		"value with  ; tail": "value with",
	}
	for in, want := range cases {
		got := stripInlineComment(in)
		if got != want {
			t.Errorf("stripInlineComment(%q) = %q, want %q", in, got, want)
		}
	}
}
