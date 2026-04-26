# scripts/

PowerShell scripts that operate the rig on Windows. Read `rig-config.json`
for paths so nothing is hard-coded.

## What's here

| File | Purpose |
|---|---|
| `_lib.ps1` | Shared helpers (`Get-RigConfig`, `Test-CheckPath`, `Test-Process`, `Test-VJoyDevice`, env-var expansion) |
| `health-check.ps1` | Probes every tool / config / process. Outputs JSON to stdout, optionally HTML report with `-Html`. |
| `monitor-iracing.ps1` | Wraps an iRacing session: snapshots `app.ini` + OpenXR cfg + recent setups, starts HWiNFO + PresentMon + process poller, launches game, waits for exit, generates report. |
| `generate-report.ps1` | Companion to `monitor-iracing.ps1`. Reads `processes.csv`, `presentmon.csv`, `*.snapshot` and emits a self-contained `report.html` with Chart.js frametime chart. |

## Setup once

1. Copy `../rig-config.example.json` to `../rig-config.json` (in repo root).
2. Edit the JSON — fill in your install paths.
3. Open PowerShell with `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned`
   (one-time), or run scripts with `-ExecutionPolicy Bypass`.

## Running

```powershell
# Health check (prints JSON + writes HTML to <logs>/health-<ts>.html)
powershell -NoProfile -File scripts\health-check.ps1 -Html

# Quiet (no progress text)
powershell -NoProfile -File scripts\health-check.ps1 -Quiet

# Wrap an iRacing session
powershell -NoProfile -File scripts\monitor-iracing.ps1
# (game launches, do your thing, exit game; report opens automatically)
```

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
