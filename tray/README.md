# Stream Deck Config — Tray

Local HTTP server + tray icon for the gaming rig.

## What it does

- Serves the `docs/` site at `http://localhost:8765`
- Exposes `/api/*` endpoints for live system state (vJoy, processes,
  monitors, audio sessions, health checks)
- Sits in the system tray with quick actions (open browser, run health
  check, quit)
- Spawns the PowerShell scripts in `../scripts/` for installs and monitoring

The HTML page in `docs/` makes optional `fetch('/api/...')` calls to enrich
the static reference with live data. If the tray isn't running, the page
still works as a static reference — the live elements just don't appear.

## Build

```bash
# On Mac (development)
cd tray
go build -o bin/tray .
./bin/tray -addr 127.0.0.1:8765

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o bin/tray.exe .
# Copy to PC, run tray.exe
```

## Layout

```
tray/
├─ main.go              entry point, tray UI, HTTP server bootstrap
├─ browser_*.go         per-OS open-in-browser shim
├─ go.mod               module definition
├─ api/
│  └─ api.go            HTTP route handlers, JSON shapes
└─ probe/
   ├─ types.go          Process, Display, VJoyState, Check
   ├─ probe_other.go    !windows: mock data for development on Mac
   └─ probe_windows.go  windows: real probes (TODO — currently stubs)
```

## Mock data on Mac

`probe_other.go` returns plausible mock data shaped like real Windows
output. This lets us build and style the live-data UI on Mac without the
PC connected. When we move to Windows, only `probe_windows.go` needs
fleshing out — handlers and types stay identical.

## Endpoints

| Method | Path | Returns |
|---|---|---|
| GET | `/api/state` | full snapshot (processes, displays, vJoy, health) |
| GET | `/api/processes` | top-50 processes by RAM |
| GET | `/api/displays` | monitor list with active state |
| GET | `/api/vjoy` | vJoy device state |
| GET | `/api/health` | full health check map |

## Future

- POST endpoints for actions (toggle monitor, restart Gremlin, run script)
- WebSocket for live updates (replace polling)
- Read `rig-config.json` for per-PC paths
- Wire HTML page to consume these endpoints
