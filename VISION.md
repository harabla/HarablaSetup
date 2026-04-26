# Vision

## What this is

A personal gaming-rig OS for one specific rig — Stream Deck MK.2, Fanatec
ClubSport SW Formula V2.5, Corsair Scimitar, three monitors, Quest in VR via
Virtual Desktop. Combines reference docs, settings verification, live
diagnostics, and recovery tooling in one repository.

The system runs on the gaming PC (a Go HTTP server in the system tray) and is
also browsable as static docs from any device. It watches the rig, alerts on
drift, captures session telemetry, and can rebuild the whole setup on a fresh
PC in roughly an hour.

## Problem

A serious sim-racing + competitive-shooter rig has 20+ moving parts. Without a
system, you:

- **Forget bindings** between sessions and waste time relearning.
- **Don't notice drift** — Windows updates change audio routing, iRacing
  resets a setting, iCUE silently swaps profiles. Things feel "off" but you
  can't say why.
- **Can't diagnose stutters** — was it OBS? A Discord update? Spotify? GPU
  thermal throttling? You shrug and shipping happens later.
- **Lose hours rebuilding** after Windows reinstall or new PC. Memory is the
  only documentation.
- **Wing it on new games** — set sensitivity by feel, fiddle with graphics
  for a week, never get a baseline.

## North star

A single repository where you can:

1. Open on any device and look up what M7 does.
2. See, at a glance, whether everything is configured correctly right now.
3. Know what's eating CPU during a stutter — without having alt-tabbed.
4. Add a new game by filling in one JSON block and have monitoring,
   sensitivity recommendations, and verification work for it immediately.
5. Reinstall Windows, run one PowerShell, get the same rig back in an hour.

## Audience

One user. One rig. One PC (95% of the time — Mac is for design when you can't
be at the rig). No teammates, no league management, no shared profiles.
Personal toolbox, not a product.

This single-user lock is a feature: it lets the system be opinionated, hard-
coded where convenient, and skip abstraction the rest of the way.

## The five modes

The system has five distinct modes, used at different rates and triggered
differently. They drive the navigation.

| Mode | Frequency | Trigger | Purpose |
|---|---|---|---|
| **Reference** | Every session | You open the page | Look up bindings, layouts, settings — "what does M7 do?" |
| **Tune** | When trying new things | You're configuring | Sensitivity calculator, "add a game" wizard, settings recommendations |
| **Verify** | Auto on game launch + on-demand | Game process appears | Read game files, diff vs expected, flag drift |
| **Diagnose** | Always-on while gaming | Game process running | Capture telemetry continuously, surface anomalies, generate session report |
| **Deploy** | Rare | Fresh PC / reinstall | Install all tools, deploy bundle, bring rig online |

### Reference

The 80% case. You open the page on a sidescreen during a session, on a phone
before queueing, on the Mac while designing. Must be glanceable, complete,
and never wrong.

Contains:
- Per-game pages: bindings on every input surface (Stream Deck profile,
  keyboard, mouse, scimitar buttons, wheel) for that game.
- Per-input pages: every Stream Deck profile, the wheel layout with photo
  overlay, keyboard layouts, Scimitar button maps.
- Architectural docs: how vJoy + Gremlin + iRFFB chain works, why MPS-L picks
  ENC-L's context, what FanaLab profiles do.

### Tune

When adding a game or changing settings. Examples:
- Add a new shooter — calculate suggested in-game sens from your existing
  PUBG cm/360° baseline (DPI 800, sens 32, ~41 cm/360°).
- ADS / scope sens conversion (links to mouse-sensitivity.com for advanced).
- Change wheel FFB strength — log the change, note which car/track, see if
  next session feels different.
- Sensitivity database imported from
  [`Wareya/Sensitivity-Database`](https://github.com/Wareya/Sensitivity-Database)
  (community-maintained game multipliers, MIT-licensed).

### Verify

Three things, all automatic where possible:

1. **Settings drift** — Read game / OS config files, parse, diff against
   expected values. Covers **graphics, key bindings, mouse sens, and
   OS-level mouse settings** — anything stored in plaintext or registry.

   Per game, `rig-config.json` declares one or more `settings_files` entries,
   each with a `category` (so drifts can be grouped by what they affect):

   ```json
   "iRacing": {
     "settings_files": [
       { "category": "graphics", "format": "ini",
         "path": "%USERPROFILE%\\Documents\\iRacing\\app.ini",
         "expected": { "Graphics.MultiSamples": 4 } },
       { "category": "controls", "format": "iracing-controls",
         "path": "%USERPROFILE%\\Documents\\iRacing\\controls.cfg",
         "expected": { "Brake Bias increase": "vJoy_71" } }
     ]
   }
   ```

   Plus a `system` block for OS-level settings:

   ```json
   "system": {
     "settings_files": [
       { "category": "mouse", "format": "registry",
         "path": "registry:HKCU\\Control Panel\\Mouse",
         "expected": { "MouseSpeed": "0" } }   // EPP off
     ]
   }
   ```

   Realistic readability matrix:

   | Source | Graphics | Binds | Mouse sens | Format | Notes |
   |---|---|---|---|---|---|
   | iRacing | `app.ini` ✓ | `controls.cfg` ✓ | n/a (wheel) | INI + custom | works today |
   | PUBG | `GameUserSettings.ini` ✓ | `Input.ini` ✓ | `GameUserSettings.ini` ✓ | INI (Unreal) | works today |
   | CS2 / Apex | ✓ | ✓ | ✓ | KV format | when added |
   | Most Unreal games | ✓ | ✓ | ✓ | INI | when added |
   | Windows mouse | n/a | n/a | `HKCU\Control Panel\Mouse` ✓ | registry | OS-level |
   | iCUE Scimitar | n/a | proprietary binary | n/a | iCUE SDK | use press-each-button test instead |
   | FanaLab profiles | n/a | partial | n/a | proprietary | future, separate concern |

   **"Snapshot current as baseline" workflow** — you tweak settings on
   purpose. The system shouldn't nag. Each drift in the Verify UI has an
   `[Accept as new baseline]` button that writes the actual value back into
   `rig-config.json`'s `expected` block. Expected = "what you decided was
   right last time", not a frozen golden image.

2. **Hardware behaviour** — Active tests for the input devices:
   - **Mouse**: in-browser polling-rate test (count `pointermove` events/sec),
     acceleration detection (drag-test for `Enhance Pointer Precision`),
     smoothing/jitter visual, link-out to MouseTester for hardware-level
     malfunction-speed and click-latency tests.
   - **Keyboard**: NKRO test (press multiple keys, all register), chatter
     test (rapid press, no doubles).
   - **Scimitar**: press-each-button verification — page captures the keystroke
     each Scimitar button sends, diffs against the mapping declared in the
     PUBG keyboard JSON. Reveals iCUE profile mismatches without parsing
     iCUE files.
   - **Wheel**: joy.cpl-style live input view, calibration drift check.

3. **Tool installs** — every path in `rig-config.json` resolves to a file
   that exists. vJoy devices configured per spec. HidHide whitelist correct.

**Triggered automatically on game launch** — tray watches for known game
exes (iRacingSim64DX11.exe, TslGame.exe, etc.) and fires verify the moment
one appears. Results surface as:
- **Toast notification** (Windows native via Go syscall) — non-blocking.
- **Stream Deck cell** — dynamic colour via BarRaider Web Requests plugin
  polling our `/api/health/summary` endpoint. Green = all clean, amber =
  drift, red = fail.

Plus on-demand from the docs page (Verify tab + button on Status).

### Diagnose

Always-on while gaming. Tray detects game process → automatically starts:
- **HWiNFO64** with CSV sensor logging (CPU/GPU/RAM/temps, 1s sample)
- **PresentMon** for frame timestamps (~0.3% CPU)
- **Process poller** capturing top-15 by RAM every 5s

Game exits → loggers stop, HTML session report auto-generates and lands in
`C:\Logs\<game>\<timestamp>\report.html`. Report includes:
- Avg/P99/stutter-count frame times
- Top processes by CPU spike (correlates with stutter timestamps)
- Snapshot of game's settings at session start
- VR-specific metrics from OpenXR Toolkit log if applicable

**Total overhead: <1% CPU, ~110MB RAM** — verified negligible on any modern
gaming PC.

The Status tab in the docs gives a live view (top processes, vJoy state,
display state, current health-check results) refreshed every 10s while open.

### Deploy

Two halves: **capture** the working rig once, **restore** it on any future PC.

**Capture (`bundle.ps1`)** — runs on the working PC after first-time setup.
Writes everything portable into `bundle/`:
- Stream Deck profiles — exports each as `.streamDeckProfile` (zip of the
  profile folder under `%APPDATA%\Elgato\StreamDeck\ProfilesV2\`).
- Stream Deck plugin list — folder names from `Plugins\` → `plugins.txt`.
- Joystick Gremlin profile — XML from `%APPDATA%\Joystick Gremlin\` (or
  Documents).
- iRacing controls.cfg — copied to `bundle/iRacing/controls.cfg.expected`.
- Display toggle .bat files — generated from `rig-config.json` displays
  block (one per monitor + presets all-on / all-off / vr-race / work).
- Audio control .bat files — generated from `rig-config.json` audio block
  (Discord/Spotify/iRacing volume up/down via SoundVolumeView).

The user runs it (`.\scripts\bundle.ps1`), reviews `git status`, commits.
FanaLab profiles are exported manually (vendor format); they get dropped
into `bundle/FanaLab/`.

**Restore (`setup.ps1`)** — runs on a fresh PC. Inverse of capture:
- `winget` installs for ~5 mainstream apps (Discord, Spotify, Steam, OBS,
  Process Lasso).
- GitHub release downloads for ~7 tools (HidHide, Joystick Gremlin,
  PresentMon, Crew Chief, iRFFB 2022, etc.).
- Portable .exe extraction (SoundVolumeView, MultiMonitorTool, HWiNFO).
- Imports each `bundle/StreamDeck/*.streamDeckProfile` via the Stream Deck
  app's import hook.
- Drops `bundle/Gremlin/*.xml` into `%APPDATA%\Joystick Gremlin\`.
- Drops `bundle/iRacing/controls.cfg.expected` to
  `Documents\iRacing\controls.cfg`.
- Drops `bundle/scripts/*.bat` into `C:\Tools\<tool>\`.
- Pauses for ~5 vendor-bound installs (FanaLab, iRacing, Stream Deck app,
  Trading Paints, Virtual Desktop) with on-screen instructions and a
  "press enter when done" prompt.
- Pauses for OAuth re-auth (Spotify, Discord plugins).
- Runs `health-check.ps1 -Html` and opens the report.

Target: **first PC = ~7 hours manual; capture once; every PC after = ~30
minutes** (mostly download time + 5 vendor installer clicks + a few
re-auths).

What's NOT bundle-able (tradeoffs documented in `bundle/README.md`):
- Plugin OAuth tokens — re-auth per PC.
- Plugin local audio device bindings — per-PC device names.
- Stream Deck plugin binaries — Marketplace handles install + updates.
- FanaLab driver — vendor.
- iRacing membership — account-bound.

## Architecture

### Repo layout

```
streamdeck-config/
├─ docs/                  static HTML/CSS/JS — reference + live data UI
├─ tray/                  Go HTTP server + system tray (Mac + Windows builds)
│  ├─ api/                JSON endpoints (state, config, action, health)
│  ├─ probe/              mock + windows real probes
│  ├─ exec/               whitelisted action dispatcher
│  ├─ config/             rig-config.json loader
│  └─ watch/              [planned] game-launch watcher + auto-monitor trigger
├─ scripts/               PowerShell — health-check, monitor-iracing, generate-report
├─ bundle/                captured-from-PC configs deployed on fresh PC
│  ├─ StreamDeck/         exported .streamDeckProfile files + plugin list
│  ├─ Gremlin/            fanatec-iracing.xml
│  ├─ FanaLab/            vendor-format profile exports
│  ├─ iRacing/            controls.cfg.expected
│  └─ scripts/            generated .bat files (display + audio)
├─ profiles/              iRacing pit macros (.txt source-of-truth)
├─ data/                  [planned] sensitivity DB, expected-settings baselines
├─ rig-config.json        per-PC paths (gitignored, templated)
└─ rig-config.example.json
```

### Data flow

```
   ┌──────────────┐                          ┌──────────────┐
   │  game files  │  ─── verify reads ──→    │ docs / Verify│
   │  (app.ini …) │                          │   tab        │
   └──────────────┘                          └──────────────┘
          ▲                                          ▲
          │                                          │
   ┌──────┴───────┐    auto on launch        ┌──────┴───────┐
   │  game exe    │  ─── triggers ────→      │  tray watch  │
   │  (running)   │                          │  + monitor   │
   └──────────────┘                          └──────────────┘
                                                     │
                                                     ▼
                                             ┌──────────────┐
                                             │ HTML report  │
                                             │ + telemetry  │
                                             └──────────────┘
```

The tray is the orchestrator. The docs page consumes its API. PowerShell does
the heavy Windows-specific work. `rig-config.json` is the single source of
truth for paths.

## Principles

**Operational:**
- *Static-first*: `docs/` works on `file://` with no server. Tray data is
  enhancement, not requirement.
- *Mock-on-dev*: developing on Mac never blocks on the PC.
- *Single source of truth*: bindings live in `docs/index.html` JSON, paths
  live in `rig-config.json`, canonical configs live in `bundle/`. No
  duplication.
- *Whitelist actions*: tray never accepts arbitrary commands. Every action
  named and validated.
- *Graceful degradation*: each layer works without the layer above.
- *Verify reads from source*: settings come from the game's actual files,
  not from manual user spec. Spec describes expected; reality drives
  display.

**Design:**
- *No build step* — pure HTML/CSS/JS, single-file Go, plain PowerShell.
  Revisit only if a feature genuinely requires it.
- *Inline JSON* for binding data — diff-friendly, works on file://. Split
  to external files only when size becomes a problem.
- *Plain Unicode glyphs*, not emoji or images.

**UX:**
- *Reference > novelty* — sidescreen-during-race is the primary use.
- *Drift surfaces itself* — auto-verify on game launch + toast + Stream
  Deck cell. You don't have to remember to check.
- *Latched state visible* — when a monitor is off, when iCUE has the
  wrong profile, when a service crashed — the page says so.
- *No surprise actions* — clicking surfaces a detail; firing requires
  explicit affordance.

## Multi-game

Each game is a JSON block in `rig-config.json`. `settings_files` is an array
of categorised entries — graphics, controls, mouse, etc. — so the same diff
engine handles everything you want to verify:

```json
"iRacing": {
  "exe": ["iRacingSim64DX11.exe", "iRacingUI.exe"],
  "ui_path": "...",
  "settings_files": [
    {
      "category": "graphics", "format": "ini",
      "path": "%USERPROFILE%\\Documents\\iRacing\\app.ini",
      "expected": { "Graphics.MultiSamples": 4, "Graphics.MaxQuality": 1 }
    },
    {
      "category": "controls", "format": "iracing-controls",
      "path": "%USERPROFILE%\\Documents\\iRacing\\controls.cfg",
      "expected": { "Brake Bias increase": "vJoy_71",
                    "Throttle":            "Fanatec axis 1" }
    }
  ],
  "monitoring": { "auto": true, "wrapper": "monitor-iracing.ps1" },
  "sensitivity": { "in_game": 0.55, "fov": 75, "type": "sim-racing" }
},
"PUBG": {
  "exe": ["TslGame.exe"],
  "settings_files": [
    {
      "category": "graphics", "format": "ini",
      "path": "%LOCALAPPDATA%\\TslGame\\Saved\\Config\\WindowsNoEditor\\GameUserSettings.ini",
      "expected": { "ScalabilityGroups.sg.TextureQuality": 3 }
    },
    {
      "category": "controls", "format": "ini",
      "path": "%LOCALAPPDATA%\\TslGame\\Saved\\Config\\WindowsNoEditor\\Input.ini",
      "expected": { "ActionMappings.Crouch": "C" }
    },
    {
      "category": "mouse", "format": "ini",
      "path": "%LOCALAPPDATA%\\TslGame\\Saved\\Config\\WindowsNoEditor\\GameUserSettings.ini",
      "expected": { "AimSensitivity": 32, "ScopeSensitivity": 20 }
    }
  ],
  "monitoring": { "auto": true, "wrapper": "monitor-pubg.ps1" },
  "sensitivity": { "dpi": 800, "in_game_hipfire": 32, "in_game_scope": 20,
                   "type": "fps-tac" }
}
```

Adding a game = drop in a block, write a per-game page in `docs/` (using a
template), optionally write a `monitor-<game>.ps1` (or share the generic
one). Verification, monitoring, sensitivity recommendations, and drift
detection all just work for the new game.

Currently configured: iRacing, PUBG. Future: any game the user adds.

## Success criteria

You can:

1. Sit at the PC, plug in the wheel, double-click `tray.exe`. Browser opens
   to live status. Pre-flight pills are accurate. Health all green.
2. Open the page on a phone before a race, scan the wheel layout for a
   binding you've forgotten.
3. Launch iRacing. Tray detects it, reads `app.ini` + `controls.cfg`, diffs
   against expected. Toast says "all settings match expected" — graphics,
   binds, FFB strength, everything. Stream Deck health cell stays green.
   Monitoring starts silently in the background.
4. Race in VR using MPS context modes for ENC-L without thinking about it.
5. Hit a stutter. After the race, open `report.html`. See exactly which
   process spiked when, what your graphics settings were.
6. Notice the mouse feels off. Open Verify tab. Two layers help:
   - **Hardware tests**: polling rate widget shows 125Hz instead of 1000Hz
     → check driver / Windows polling rate setting → re-test → green.
   - **Settings drift**: PUBG `AimSensitivity` actual is 28, expected was 32.
     Either accept new value as baseline (intentional change) or restore.
7. After a long session your iRacing FFB strength was tweaked. Open Verify,
   see "FFB drift" entry → click [Accept as baseline] → expected gets
   updated; no nag next session.
8. Add a new shooter. Tune tab calculates suggested sens from your PUBG
   baseline. Drop a JSON block in rig-config with paths + expected. New
   game gets verify + monitoring + sensitivity recommendations
   automatically.
9. Reinstall Windows. Clone repo, run `setup.ps1`, restore `bundle/`.
   Racing again within an hour with the same rig.

If those nine flows feel friction-free, the system works. Anything that
doesn't serve them is a candidate for cutting.

## Status

**Built (works on Mac with mock data + Windows binaries cross-compiled):**

*Reference / Tune / Verify / Diagnose surfaces:*
- Static reference site (wheel + 6 Stream Deck profiles + FanaLab + macros)
- 6-tab nav: Overview · Reference · Tune · Verify · Diagnose · Deploy
- Sensitivity converter (cm/360°), 14-game library, add-game JSON generator
- Mouse hardware tests in browser (polling, accel detection, jitter, Windows accel snippet)
- Scimitar mapping verification (press-each-button capture + persistence)
- Keyboard chatter / NKRO test (chatter detection, peak-NKRO tracking)
- Pre-game routine cards (per-game checklists, localStorage progress)
- Settings drift cards with [Accept-as-baseline] (in-memory + disk persist)

*Tray + scripts (the runtime):*
- Tray with 9 JSON endpoints (state, processes, displays, vjoy, health,
  health/summary, config, action POST, games, verify, /VISION.md, /README.md)
- Mock-on-Mac probes, whitelist-based action dispatcher
- rig-config.json loader (env expansion + dev defaults)
- Game-launch watcher (2.5s poll, fires verify + auto-monitor on transition)
- Toast notifications (gen2brain/beeep — cross-platform)
- INI parser + registry reader + diff engine + verify package
- Disk persistence for baseline updates

*PowerShell scripts:*
- health-check.ps1 (real Windows probes via registry + process queries)
- monitor-iracing.ps1 (full session capture: HWiNFO + PresentMon + poller)
- generate-report.ps1 (HTML report with Chart.js frametime chart)
- bundle.ps1 (capture working PC's portable configs into bundle/)
- setup.ps1 (8-phase deploy: preflight → winget → github → portable →
  vendor → restore → rigconfig → healthcheck. Idempotent, -DryRun
  supported, per-phase skip/only flags, logs to C:\Logs\setup\)

*Deploy scaffolding:*
- bundle/ directory with README documenting structure + capture workflow
- rig-config.example.json template

*Build:*
- Cross-compile: 9 MB Mac binary + 9.6 MB Windows .exe from same source

**Next (in roughly this order):**

*PC-required to validate / progress:*
1. **Run `bundle.ps1` once** on the working PC, commit `bundle/` outputs
   (Stream Deck profiles, Gremlin XML, controls.cfg, generated .bat
   files). Until this happens, fresh-PC deploys still rebuild Stream
   Deck profiles + Gremlin XML by hand.
2. **Run `setup.ps1` on a fresh PC** to validate the install flow.
   Likely surfaces: winget package ID corrections, GitHub release asset
   patterns, vendor URL changes. Fix as found.
3. **Real Windows probes** in `probe_windows.go` — replace mocks with
   registry reads (vJoy / HidHide / mouse settings), `EnumDisplayDevices`,
   PerfCounter for live CPU. ~2 hours sitting at the PC.
4. **Wheel calibration check** widget on Verify (live joy.cpl-style view —
   needs Windows joystick API).

*Mac-shippable polish:*
5. **iRacing telemetry hook** (live FFB clipping in Status via SimHub
   shared memory or iRSDK direct). ~3 hours.
6. **Settings change log** in Tune tab — annotated history of intentional
   tweaks with timestamps. ~1 hour.
7. **ADS / scope sens converter** in Tune (FOV-corrected math for tactical
   FPS, embedded vs link-out trade-off). ~2 hours.
8. **Auto-discovery of paths** — probe Steam manifest + registry to
   pre-populate `rig-config.json`. ~2 hours.

**Out of scope:**
- Stream Deck custom plugin (BarRaider Web Requests is enough)
- Tauri / Electron desktop wrapper (browser tab is right shape)
- Multi-user / cloud sync / sharing
- Game guides / drop locations / strategic content beyond your own settings
- Content creation tooling beyond triggering OBS
- iCUE binary profile parsing (use press-each-button verification instead)
