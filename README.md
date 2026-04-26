# HarablaSetup

Personal gaming-rig OS for one specific rig: Stream Deck MK.2 + Fanatec
ClubSport SW Formula V2.5 + Corsair Scimitar + 3 monitors + Quest VR via
Virtual Desktop. Reference docs + live verification + always-on diagnostics
+ recovery tooling, in one repository.

See [`VISION.md`](./VISION.md) for the full design and rationale.

## Quick start

**On any device — view the reference:**

```bash
open docs/index.html        # macOS / file://
start docs\index.html       # Windows
```

Works offline. No build step, no dependencies. The page is the spec for
the whole rig.

**On the gaming PC — run the live tray:**

```cmd
:: Build once (Go required)
cd tray
go build -o bin\tray.exe .

:: Run
.\bin\tray.exe
```

Tray sits in the system tray. Browser at `http://localhost:8765` shows live
state. Game launches → auto-verify settings → auto-spawn telemetry capture
→ session report on exit.

## Repo layout

```
HarablaSetup/
├─ docs/                  static HTML/CSS/JS — reference + live data UI
├─ tray/                  Go HTTP server + tray (Mac + Windows builds)
│  ├─ api/                JSON endpoints
│  ├─ probe/              system state (mock on Mac, real on Windows)
│  ├─ exec/               whitelisted action dispatcher
│  ├─ verify/             settings drift diff engine
│  ├─ parse/              INI / iRacing-controls / registry parsers
│  ├─ watch/              game-launch watcher
│  ├─ notify/             cross-platform desktop notifications
│  └─ config/             rig-config.json loader + writer
├─ scripts/               PowerShell — health-check, monitor, generate-report, bundle
├─ bundle/                captured-from-PC configs deployed on fresh PCs
├─ profiles/              iRacing pit macros + work-PC stubs
├─ rig-config.example.json  template — copy to rig-config.json (gitignored) and fill in paths
└─ VISION.md              project vision + design + roadmap
```

## The five modes

The docs site groups everything into five operating modes (see VISION.md):

| Mode | Purpose |
|---|---|
| 🏠 Overview | What's running right now · health summary · vision link |
| 📚 Reference | Wheel · Stream Deck profiles · keyboard / mouse / scimitar · architecture |
| 🎯 Tune | Sensitivity converter · game library · "add a game" generator |
| 🩺 Verify | Settings drift · mouse hardware · scimitar mapping · keyboard chatter · pre-game routine |
| 📊 Diagnose | Live process / display / vJoy / health · session reports |
| 🔧 Deploy | PC setup walkthrough · `bundle.ps1` capture · planned `setup.ps1` restore |

## Editing the reference

All bindings live as JSON inside `docs/index.html` (in
`<script type="application/json">` blocks). Edit the JSON, refresh, commit.
The diff shows exactly what changed.

For iRacing pit macros, also update the matching `.txt` file in
`profiles/gaming/macros/` so the autochat strings stay version-controlled.

## Per-PC configuration

Copy `rig-config.example.json` to `rig-config.json` (gitignored), fill in
paths to your installs, monitor IDs, etc. The tray + scripts read this file
as the single source of truth for per-PC paths.

## Capturing the rig (one-time, on the working PC)

After you've done the manual first-time setup once:

```cmd
.\scripts\bundle.ps1
git add bundle/
git commit -m "Update bundle from <hostname>"
git push
```

This captures Stream Deck profiles, the Joystick Gremlin XML, iRacing
controls.cfg, and generates display + audio .bat files from rig-config.
Future fresh-PC deploys read `bundle/` to skip ~6 hours of manual rebuild.

## Status

Works end-to-end on Mac with mock data; Windows binary cross-compiles
clean. The actual rig-validation (real Windows probes, runtime telemetry,
PC-only flows) happens when you sit at the PC. See
[`VISION.md`](./VISION.md#status) for the full status block.
