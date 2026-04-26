package watch

import (
	"context"
	"testing"
	"time"

	"github.com/hkbla/streamdeck-config/tray/config"
)

func TestExeNames(t *testing.T) {
	cases := []struct {
		def  config.GameDef
		want []string
	}{
		{config.GameDef{Exe: "iRacingSim64DX11.exe"}, []string{"iRacingSim64DX11"}},
		{config.GameDef{Exe: []interface{}{"a.exe", "b.exe"}}, []string{"a", "b"}},
		{config.GameDef{Sim: `C:\foo\bar.exe`, UI: `C:\baz\ui.exe`}, []string{"bar", "ui"}},
		{config.GameDef{}, []string{}},
	}
	for _, c := range cases {
		got := exeNames(c.def)
		if len(got) != len(c.want) {
			t.Errorf("exeNames(%+v) = %v, want %v", c.def, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("exeNames(%+v)[%d] = %q, want %q", c.def, i, got[i], c.want[i])
			}
		}
	}
}

func TestGameRunning(t *testing.T) {
	procSet := map[string]struct{}{
		"iracingsim64dx11": {},
		"discord":          {},
	}
	cases := []struct {
		def  config.GameDef
		want bool
	}{
		{config.GameDef{Exe: "iRacingSim64DX11.exe"}, true},
		{config.GameDef{Exe: "Notepad.exe"}, false},
		{config.GameDef{Sim: `C:\iRacing\iRacingSim64DX11.exe`}, true}, // matches via Sim
		{config.GameDef{}, false},
	}
	for _, c := range cases {
		got := gameRunning(c.def, procSet)
		if got != c.want {
			t.Errorf("gameRunning(%+v): got %v, want %v", c.def, got, c.want)
		}
	}
}

func TestWatcher_Lifecycle(t *testing.T) {
	cfg := &config.Config{
		Games: map[string]config.GameDef{
			"FakeGame": {Exe: "iracingui.exe"}, // matches mock that pretends iracingui is always running
		},
	}
	w := New(cfg)
	w.interval = 50 * time.Millisecond

	events := w.Subscribe()
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	w.Start(ctx)

	select {
	case ev := <-events:
		if ev.Game != "FakeGame" || ev.Kind != "launch" {
			t.Errorf("expected FakeGame launch, got %+v", ev)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("no event received within 300ms — watcher didn't fire")
	}

	states := w.States()
	if !states["FakeGame"].Running {
		t.Error("FakeGame should be marked running")
	}
}
