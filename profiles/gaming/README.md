# Gaming profile

Stream Deck MK.2 profile for the gaming PC, plus the supporting layouts on the
Fanatec wheel and Corsair Scimitar mouse.

## Hardware split

Three control surfaces, three jobs. No overlap.

| Device | Job | Examples |
| --- | --- | --- |
| **Stream Deck MK.2** (15 keys) | Pre-game, between-game, pit strategy, streaming. | Launch sims, switch EQ, OBS scenes, pit macros. |
| **Fanatec ClubSport F1 V2.5 wheel** | iRacing in-car only. | Brake bias, fuel mix, TC/ABS/Engine map, DRS, ERS, pit limiter. |
| **Corsair Scimitar (12 thumb buttons)** | PUBG in-game only. | Throwables, seat switch, scope zero, heals, ping. |

In a shooter your hand never leaves the mouse, so PUBG actions live on the
Scimitar — the Stream Deck is only used before queueing. In iRacing your hands
never leave the wheel, so driving stays on the wheel — the Stream Deck is only
used in the pits or before a session.

## Switching between work and gaming profiles

The Stream Deck MK.2 is shared between the work PC and the gaming PC via a USB
hub that routes the device to one machine at a time. Switching the hub causes
the Stream Deck to load whichever profile is bound on that machine — no manual
profile change required inside the Stream Deck app.

## Icon spec

- 144×144 px, PNG
- Dark background (matches Stream Deck default)
- Bold label at the top
- Symbol/glyph below
- Colour-coded by category (see below)

### Colour categories

| Colour | Use |
| --- | --- |
| Blue | iRacing actions, sim launches |
| Red | OBS stop/end, throwables, mic mute |
| Green | PUBG launch, "go" actions, healing |
| Amber | Pit strategy, caution, scope zero |
| Purple | ERS / overtake, ping, recording |
| Teal | Audio / volume |

## Plugins

Stream Deck plugins required for this profile (all from the Stream Deck Store
unless noted).

| Plugin | Source | Purpose |
| --- | --- | --- |
| iRaceIT | Stream Deck Store | Live iRacing telemetry, pit/black-box actions |
| SuperMacro | Stream Deck Store | Send pit autochat strings via keypress macros |
| Win Tools | Stream Deck Store | App Audio Mixer for per-app volume (Discord) |
| OBS Studio (bundled) | Stream Deck software | Scene switching, recording, replay buffer |

System actions used (no plugin needed):

- **Hotkey** — for SteelSeries GG EQ profile switches
- **Open** — for app launches (iRacing, Crew Chief, SimHub, Trading Paints)

## Folder layout

```
gaming/
├── icons/         144×144 PNGs grouped by section
├── macros/        iRacing autochat strings, one per file
└── scripts/       SuperMacro / AHK helpers (later)
```

## When bindings change

1. Update the relevant entry in `docs/data/*` (inline JSON in `docs/index.html`).
2. If it's an iRacing macro, also update the matching `.txt` file in `macros/`.
3. Rebind in the Stream Deck app (the doc is the source of truth, the app
   mirrors it).
4. Commit — the diff shows exactly what changed.
