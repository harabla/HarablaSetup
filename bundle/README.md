# bundle/

Captured per-rig configuration files. Committed to the repo so a fresh PC
deploys in minutes instead of hours. Run `scripts/bundle.ps1` on a working
PC to (re-)populate this directory; commit the result.

## Layout

```
bundle/
├─ StreamDeck/
│  ├─ Home.streamDeckProfile           ← exported profiles, one per name
│  ├─ iRacing.streamDeckProfile
│  ├─ PUBG.streamDeckProfile
│  ├─ Streaming.streamDeckProfile
│  ├─ Audio.streamDeckProfile
│  ├─ Display.streamDeckProfile
│  └─ plugins.txt                       ← list of plugin IDs to install via Marketplace
├─ Gremlin/
│  └─ fanatec-iracing.xml               ← Joystick Gremlin profile (vJoy mappings)
├─ FanaLab/
│  └─ profiles.zip                      ← FanaLab profile exports (manual export step)
├─ iRacing/
│  └─ controls.cfg.expected             ← iRacing controls.cfg snapshot
└─ scripts/
   ├─ display-toggle-top-left.bat       ← generated from rig-config.json display IDs
   ├─ display-toggle-top-right.bat
   ├─ display-toggle-ultra.bat
   ├─ display-all-on.bat
   ├─ display-all-off.bat
   ├─ display-vr-race.bat
   ├─ display-work.bat
   ├─ discord-up.bat                    ← generated for SoundVolumeView per-app vol
   ├─ discord-down.bat
   ├─ spotify-up.bat
   ├─ spotify-down.bat
   ├─ iracing-up.bat
   └─ iracing-down.bat
```

## How to populate

1. On your working PC (after the manual first-time setup), open PowerShell
   in the repo root.
2. Run:

   ```powershell
   .\scripts\bundle.ps1
   ```

   This captures Stream Deck profiles + plugin list + Gremlin XML + iRacing
   controls.cfg + generates the .bat files from rig-config.json.

3. FanaLab profiles export manually (FanaLab → Settings → Export profile)
   → drop the resulting files into `bundle/FanaLab/`.

4. Commit the changes:

   ```powershell
   git add bundle/
   git commit -m "Update bundle from PC <hostname> on <date>"
   git push
   ```

## How a fresh PC deploys from this

1. Clone the repo.
2. Run `scripts/setup.ps1` (when shipped — currently planned).
   It will:
   - winget install / GitHub-release download the tools listed in
     `bundle/StreamDeck/plugins.txt` and the rig dependencies
   - Stream Deck app: import each `*.streamDeckProfile` via the app's
     CLI / API
   - Joystick Gremlin: copy `Gremlin/fanatec-iracing.xml` to
     `%APPDATA%\Joystick Gremlin\` and load it
   - iRacing: copy `iRacing/controls.cfg.expected` to
     `Documents\iRacing\controls.cfg` (after iRacing has been launched once
     so the dir exists)
   - SoundVolumeView / MultiMonitorTool: copy the .bat files into
     `C:\Tools\<tool>\`
   - Pause for vendor-bound steps (FanaLab profiles import, iRacing /
     Discord / Spotify / Trading Paints / Virtual Desktop installs)
   - Run `health-check.ps1 -Html` and open the report

3. Re-auth Spotify, Discord plugins (one click each).

4. Done in ~30 minutes vs ~7 hours of manual building.

## What's NOT in the bundle (and why)

| Item | Why not |
|---|---|
| Plugin OAuth tokens (Spotify, Discord) | Per-PC security — re-auth takes 10s |
| Plugin local audio device bindings (Volume Controller) | Per-PC device names differ |
| Stream Deck plugin binaries | Marketplace handles install + updates |
| FanaLab driver | Hardware vendor — install separately |
| iRacing membership | Account-bound, not portable |

## Updating the bundle

Whenever you intentionally change a profile or binding:

1. Re-run `scripts/bundle.ps1` to refresh the captured files.
2. Commit the diff.

The bundle is the canonical "this is the rig as configured today". Drift
detection (Verify mode) compares current state against `expected` values
in `rig-config.json`; the bundle covers the file-level configs that the
diff doesn't cover.
