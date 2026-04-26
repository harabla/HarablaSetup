# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

Reference + configuration for an Elgato Stream Deck MK.2 that's shared between
a work PC and a gaming PC via a USB hub. The primary deliverable is a
self-contained reference page at `docs/index.html` that opens directly from the
filesystem (`file://`) on a sidescreen during gaming sessions.

There is **no build step**, no package manager, no tests. The reference page is
plain HTML / CSS / JS, viewed by double-clicking the file or:

```bash
open docs/index.html        # macOS
start docs\index.html       # Windows
```

## Architecture

The reference page is **data-driven**. All bindings live as JSON inside
`<script type="application/json">` blocks in `docs/index.html`. `docs/app.js`
reads those blocks via `document.getElementById(...).textContent` + `JSON.parse`
and renders into containers in the HTML shell. This is a deliberate choice over
`fetch('data.json')` because Chrome/Edge block `fetch` on `file://`; inline
JSON keeps the page double-clickable while still giving clean diffs when a
binding changes.

Three data domains, each its own `<script id="data-...">` block:

- **Stream Deck grids** (`data-streamdeck-{home,pubg,iracing,streaming}`) — 3×5
  cell layouts. Each cell has `label`, `glyph`, `cat` (colour token), `type`,
  optional `macro` and `notes`.
- **Input devices** (`data-pubg-keyboard`, `data-pubg-scimitar`,
  `data-iracing-wheel`) — physical-layout JSON with category tags.
- **Meta** (`data-meta`) — colour category dictionary (used both for keyboard
  semantics and Stream Deck colour codes — there is intentional overlap, e.g.
  `blue` means "iRacing/launch" on a Stream Deck cell *and* "movement" on the
  keyboard) and the setup checklist.

The page uses a **tab UI**: only one `<section>` is visible at a time
(`section.active` shown, others `display: none`). `app.js#setTab()` is the
single source of truth — it updates the URL hash, persists to `localStorage`
(`gaming-ref:active-tab`), resets the search state, and closes the detail panel.
Initial tab priority: URL hash → localStorage → first section.

Interactivity built into `app.js`:

1. **Search** (`#search`) — substring match against `data-searchable` attribute
   set on every rendered cell/key/button. Non-matches dimmed via `body.search-dim`.
2. **Click-to-detail** — every Stream Deck cell, keyboard key, and Scimitar
   button shows a slide-out panel (`#detail`) on click.
3. **Car-class toggle** — GT3 / LMP / Formula buttons re-render the wheel
   layout from `data-iracing-wheel`'s `byClass` fields. Differing rows get a
   gold highlight.
4. **Macro copy** — pit autochat strings have a Copy button using
   `navigator.clipboard.writeText`.

A **validation pass** runs on load (`app.js`) that warns in the console for any
binding referencing an unknown category — catches typos like `"ambr"` before
they cause a silent uncoloured render.

## When bindings change

1. Edit the relevant `<script type="application/json">` block in
   `docs/index.html`. Comment banners (`<!-- ====== STREAM DECK :: HOME ====== -->`)
   mark each block.
2. **For iRacing macros only**, also update the matching `.txt` file in
   `profiles/gaming/macros/`. The `.txt` files are the version-controlled
   source-of-truth for the autochat strings; the JSON in `index.html` mirrors
   them. Both must stay in sync.
3. Refresh the browser. Open DevTools → Console to confirm the validation pass
   prints `data validated: N categories, all bindings OK`.

## Conventions baked into the data

- **Glyphs are plain Unicode** (`▶ ● ■ ⏺ ⏹ ✂ ⊘`), not emoji — they render
  identically across OSes. Don't introduce emoji.
- **Colour tints** are pre-computed hex (`--tint-blue` etc. in `styles.css`),
  not `color-mix()` — keeps rendering consistent on older browsers.
- **`999l`** in iRacing fuel macros means "fill to max" — iRacing caps at tank
  capacity per car, so this works across GT3 / LMP / Formula without knowing
  the tank size.
- **`row` field on Scimitar buttons** (`hard` / `mid` / `easy` / `bot`)
  controls difficulty badges in the rendered grid. M7–M9 are "easy" because
  they're the natural thumb-rest row.
- **Empty Stream Deck cells** use `"cat": "empty", "type": "empty"` — they
  render as dashed placeholders and are non-interactive.

## Hardware split (load-bearing assumption)

Three control surfaces, three jobs, **no overlap**:

- **Stream Deck**: pre-game, between-game, pit strategy, streaming.
- **Fanatec ClubSport F1 V2.5 wheel**: iRacing in-car only.
- **Corsair Scimitar (12 thumb buttons)**: PUBG in-game only.

Suggestions that move in-game actions onto the Stream Deck or in-car actions
off the wheel are violating the design — flag them rather than implementing.

## Planned: work mode

The current page is gaming-only. A "Mode: Gaming / Work" switch above the tab
bar is planned for when the work profile gets structured (currently
`profiles/work/` only holds icons + a PDF). When adding it: the chrome change
is small (mode switch + filter tabs by mode), data shape gains a top-level
mode group. Don't duplicate `index.html` per mode — single file, single
source.

## What's not here

- No tests, no linter, no CI.
- No `package.json` — by design. Adding one is a real decision; ask before doing it.
- No icons yet — `profiles/gaming/icons/{home,iracing,pubg,streaming}/` are
  empty placeholder folders awaiting 144×144 PNGs.
- No actual Stream Deck profile export — the JSON in `index.html` is the
  human-readable spec; mapping it onto a real `.streamDeckProfile` export is
  future work.
