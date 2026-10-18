---
id: "bsvzj521xa"
title: "Code generation export"
state: "todo"
due: "2028-10-18"
priority: ""
tag: ["feature"]
repeat: ""
effort: ""
postpone: 0
created: "2026-10-18"
closed: ""
---

Export a drawing as source code for a TUI framework (Bubble Tea + Lip Gloss,
Ratatui, Textual).

## Plan

1. Static: emit the drawing as a string constant or styled blocks.
2. Semantic: tag shapes as widgets (panel, label, button, list, input) in the
   editor and map them to real framework widgets.

## Acceptance

- Generated code compiles and renders the same layout as the drawing.

Needs color and ANSI export (v2) first.
