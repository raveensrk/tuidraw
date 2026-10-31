# Overview

What tuidraw is for, who it serves, and where it stops.

## Problem

General design tools work in pixels, so a mockup never quite matches what a terminal can show.
ASCII editors either stop at plain boxes and lines, lack color, or are hard to use.
tuidraw sits between them: a lightweight canvas where every shape lands on the character grid, so the design you draw is the screen you will build.

## Users

| User | Commitment |
|---|---|
| The author, designing their own terminal UIs | Primary user; the app exists because no existing program fit their needs |
| Anyone else who installs it | Welcome under the MIT license, with no support owed |

> [!NOTE]
> tuidraw is vibe coded: built by prompting Claude Opus 5.5 at low effort in Claude Code, with a person steering, testing and reviewing.

## Scope

| Area | In scope |
|---|---|
| Canvas | Infinite in every direction; one cell is one terminal character |
| Shapes | Rect, oval, circle, triangle, diamond, polygon, line or arrow, text, brush stroke, group |
| Borders | Five styles that join where shapes touch or cross |
| Editing | Select, move, resize, restyle, group, reorder, duplicate, delete, undo, redo |
| Color | 16 ANSI colors for ink and background, opaque fill, eraser |
| Input | Mouse, plus a keyboard cursor that does everything the mouse does |
| Files | Editable project file, plain text export, ANSI color export |
| Clipboard | Copy and paste shapes; copies also reach the system clipboard as text |
| Delivery | One binary with no runtime |

## Non-goals

- Real-time collaboration.
- Truecolor or 256-color palettes.
- Image export.
- Animation.
- Running the mockups as live interfaces.
- Layout of wide characters such as CJK and emoji.

Exporting a drawing as framework source code is planned future work, tracked in [its task file](../task/2026_10_18_code_generation_export.md), and is outside a rebuild.

## Success criteria

- A design drawn in tuidraw can be built as that exact screen.
- The author reaches for it over the alternatives.

## Rebuild exactness

| Must match exactly | May be equivalent |
|---|---|
| Project file format | Color values of the editor's own interface |
| Border, arrow and outline runes | Help page wording and pager internals |
| Text and ANSI exports | Internal structure and names |
| Key bindings | Status message wording |

Previous: [Cover](README.md) | Next: [Architecture](02-architecture.md)
