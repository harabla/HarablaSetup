# streamdeck-config

Stream Deck MK.2 reference and configuration. The deck is shared between a
work PC and a gaming PC via a USB hub.

## Layout

```
streamdeck-config/
├── profiles/
│   ├── work/        Work-PC profile (icons + reference PDF, no structure imposed)
│   └── gaming/      Gaming-PC profile (this is where active development is)
└── docs/
    └── index.html   Self-contained reference page — open in any browser
```

## View the reference

Double-click `docs/index.html`, or:

```bash
open docs/index.html        # macOS
start docs\index.html       # Windows
xdg-open docs/index.html    # Linux
```

Works fully offline. No build step, no dependencies.

## Editing bindings

The reference page is data-driven — bindings live as JSON inside
`docs/index.html` (in `<script type="application/json">` blocks). Edit the JSON,
refresh the page, commit. The diff shows exactly what changed.

For iRacing macros, also update the matching `.txt` file in
`profiles/gaming/macros/` so the autochat strings stay version-controlled.
