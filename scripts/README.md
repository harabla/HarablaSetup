# scripts/

PowerShell scripts that operate the rig on Windows. Read `rig-config.json`
for paths so nothing is hard-coded.

## What's here

| File | Purpose |
|---|---|
| `_lib.ps1` | Shared helpers (`Get-RigConfig`, `Test-CheckPath`, `Test-Process`, `Test-VJoyDevice`, env-var expansion) |
| `setup.ps1` | One-shot deploy: install everything (winget + GitHub + portable) + restore from `bundle/` + verify. Idempotent, phased, supports `-DryRun`. |
| `bundle.ps1` | Inverse of `setup.ps1` — capture this PC's Stream Deck profiles + Gremlin XML + iRacing controls.cfg + generate .bat files into `bundle/`. Run on the working PC, commit the result. |
| `health-check.ps1` | Probes every tool / config / process. Outputs JSON to stdout, optionally HTML report with `-Html`. |
| `monitor-iracing.ps1` | Wraps an iRacing session: snapshots `app.ini` + OpenXR cfg + recent setups, starts HWiNFO + PresentMon + process poller, launches game, waits for exit, generates report. |
| `generate-report.ps1` | Companion to `monitor-iracing.ps1`. Reads `processes.csv`, `presentmon.csv`, `*.snapshot` and emits a self-contained `report.html` with Chart.js frametime chart. |

## Two-PC workflow

```
┌──────────────────────────────┐         ┌──────────────────────────────┐
│ PC A — already configured    │         │ PC B — fresh                 │
│                              │         │                              │
│ scripts/bundle.ps1           │  push   │                              │
│   captures profiles, XML,    ├────────▶│                              │
│   controls.cfg, .bat scripts │  bundle │                              │
│ git commit + push            │         │                              │
└──────────────────────────────┘         │                              │
                                         │ git clone                    │
                                         │ scripts/setup.ps1            │
                                         │   installs everything,       │
                                         │   restores from bundle/      │
                                         └──────────────────────────────┘
```

## Setup once

1. Copy `../rig-config.example.json` to `../rig-config.json` (in repo root).
2. Edit the JSON — fill in your install paths.
3. Open PowerShell with `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned`
   (one-time), or run scripts with `-ExecutionPolicy Bypass`.

## Running

```powershell
# First-PC manual setup, then capture for re-deploy
powershell -NoProfile -File scripts\bundle.ps1
git add bundle/ && git commit -m "Update bundle from $env:COMPUTERNAME" && git push

# Fresh-PC deploy (run from repo root, ideally in elevated PS for HidHide etc)
powershell -NoProfile -File scripts\setup.ps1                 # full run
powershell -NoProfile -File scripts\setup.ps1 -DryRun         # preview only
powershell -NoProfile -File scripts\setup.ps1 -OnlyPhase restore  # just restore from bundle/
powershell -NoProfile -File scripts\setup.ps1 -SkipPhase vendor   # skip browser-prompt phase

# Health check (prints JSON + writes HTML to C:\Logs\health-<ts>.html)
powershell -NoProfile -File scripts\health-check.ps1 -Html

# Wrap an iRacing session manually (the tray watcher does this auto on launch)
powershell -NoProfile -File scripts\monitor-iracing.ps1
# (game launches, do your thing, exit game; report opens automatically)
```

`setup.ps1` phases (run in order, individually skippable):

1. `preflight` — admin? PS version? internet? winget? repo root?
2. `winget` — Discord, Spotify, Steam, OBS, Process Lasso, HWiNFO64, Stream Deck app
3. `github` — HidHide, Joystick Gremlin, Crew Chief, iRFFB 2022 (latest releases)
4. `portable` — SoundVolumeView, MultiMonitorTool, PresentMon (extract to `C:\Tools\`)
5. `vendor` — opens browser to Fanatec, iRacing, Trading Paints, Virtual Desktop, OpenXR Toolkit, SteelSeries GG; prompts for "press Enter when done"
6. `restore` — copies bundle/ files into Stream Deck profiles dir, Gremlin AppData, iRacing Documents, `C:\Tools\<tool>\` for .bat files
7. `rigconfig` — creates `rig-config.json` from example if missing
8. `healthcheck` — runs `health-check.ps1 -Html`, opens the report

All phases are idempotent: re-running skips work that's already done.
Logs to `C:\Logs\setup\setup-<timestamp>.log`.

## Wired to the tray

The Go tray's `POST /api/action` accepts:
- `scripts.health` → spawns `health-check.ps1`
- `scripts.monitor.iracing` → spawns `monitor-iracing.ps1`

Both run with `-NoProfile -ExecutionPolicy Bypass` from the tray binary.

## Wired to the Stream Deck

| Stream Deck cell | Action |
|---|---|
| iRacing 📊 (iRacing profile) | System: Open → `powershell.exe -NoProfile -File <repo>\scripts\monitor-iracing.ps1` |
| Display Top-L / Top-R / Ultra | System: Open → tray endpoint, OR direct `MultiMonitorTool /switch` (see PC Setup tab) |

## Adding a new monitored game

1. Add the game's exe + settings paths to `rig-config.json` under `games.<name>`.
2. Copy `monitor-iracing.ps1` to `monitor-<game>.ps1`, swap the iRacing-specific
   bits (UI path, process names to wait on, settings-file names).
3. Add a `scripts.monitor.<game>` action to `tray/exec/exec.go`.
4. Add a Stream Deck cell.
