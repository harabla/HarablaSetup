package verify

import (
	"testing"

	"github.com/harabla/HarablaSetup/tray/config"
)

func newTestConfig() *config.Config {
	return &config.Config{
		Games: map[string]config.GameDef{
			"iRacing": {
				SettingsFiles: []config.SettingsFile{
					{
						Category: "graphics", Format: "ini",
						Path: "../parse/testdata/iracing-app.ini",
						Expected: map[string]string{
							"Graphics.MultiSamples":  "4",  // matches
							"Graphics.MaxQuality":    "1",  // matches
							"Graphics.MirrorHigh":    "0",  // mismatch (file says 1)
							"Graphics.NonExistentKey": "x", // missing
						},
					},
					{
						Category: "controls", Format: "iracing-controls",
						Path: "../parse/testdata/iracing-controls.cfg",
						Expected: map[string]string{
							"Controls.Brake Bias increase": "vJoy_71", // matches
							"Controls.Throttle":            "Fanatec axis 1", // matches
						},
					},
				},
			},
		},
	}
}

func TestGame_HappyAndDrifts(t *testing.T) {
	cfg := newTestConfig()
	res := Game(cfg, "iRacing")
	if res.Target != "iRacing" {
		t.Errorf("target: got %q, want iRacing", res.Target)
	}
	if len(res.Files) != 2 {
		t.Fatalf("files: got %d, want 2", len(res.Files))
	}
	// Graphics file should have 2 drifts (mismatch + missing) and 2 OKs
	gfx := res.Files[0]
	if gfx.Status != "drift" {
		t.Errorf("gfx status: got %q, want drift", gfx.Status)
	}
	if len(gfx.Drifts) != 2 {
		t.Fatalf("gfx drifts: got %d, want 2", len(gfx.Drifts))
	}
	// Controls file should be all-OK
	ctrl := res.Files[1]
	if ctrl.Status != "ok" {
		t.Errorf("ctrl status: got %q, want ok", ctrl.Status)
	}
	if len(ctrl.Drifts) != 0 {
		t.Errorf("ctrl drifts: got %d, want 0", len(ctrl.Drifts))
	}
	// Aggregate counts
	if res.DriftCount != 2 {
		t.Errorf("DriftCount: got %d, want 2", res.DriftCount)
	}
	if res.OkCount != 4 { // 2 ok in gfx + 2 ok in ctrl
		t.Errorf("OkCount: got %d, want 4", res.OkCount)
	}
}

func TestGame_FileMissing(t *testing.T) {
	cfg := &config.Config{
		Games: map[string]config.GameDef{
			"FakeGame": {
				SettingsFiles: []config.SettingsFile{
					{Category: "graphics", Format: "ini", Path: "/nonexistent.ini",
						Expected: map[string]string{"x": "y"}},
				},
			},
		},
	}
	res := Game(cfg, "FakeGame")
	if res.MissingFiles != 1 {
		t.Errorf("MissingFiles: got %d, want 1", res.MissingFiles)
	}
	if res.Files[0].Status != "missing" {
		t.Errorf("status: got %q, want missing", res.Files[0].Status)
	}
}

func TestSnapshotBaseline(t *testing.T) {
	cfg := newTestConfig()
	// Snapshot the mismatching MirrorHigh into expected
	err := SnapshotBaseline(cfg, "iRacing", "../parse/testdata/iracing-app.ini", "Graphics.MirrorHigh", "1")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	// Re-verify; that drift should be gone (NonExistentKey still drifts)
	res := Game(cfg, "iRacing")
	gfx := res.Files[0]
	if len(gfx.Drifts) != 1 {
		t.Errorf("after snapshot, gfx drifts: got %d, want 1 (only NonExistentKey)", len(gfx.Drifts))
	}
}

func TestAll(t *testing.T) {
	cfg := newTestConfig()
	all := All(cfg)
	if len(all) != 1 { // 1 game, no system block
		t.Errorf("all results: got %d, want 1", len(all))
	}
}
