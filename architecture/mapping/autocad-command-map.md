# AutoCAD command → Oblikovati action map

The Command Window (M26) accepts AutoCAD-style command words and aliases and dispatches
them to Oblikovati's existing commands and built-in actions. This page is the human-readable
companion to the machine source of truth, the built-in vocabulary in
[`app/cmdline/vocabulary.go`](../../app/cmdline/vocabulary.go). A test
(`TestEveryVocabularyWordResolves`) asserts every word here resolves to a real registered
command id or built-in action id, so this table and the code cannot drift apart silently.

## How it resolves

A typed word resolves through the binding engine in this order (`Bindings.ResolveAlias`):

1. a **user-defined alias** (Keymap editor) — always wins;
2. the **built-in AutoCAD vocabulary** below — but only to an action that exists in the
   session;
3. for a **single character**, that key's keyboard shortcut (so one-letter shortcuts still
   run when typed and committed with Enter, e.g. `V` → toggle work-plane visibility).

The first word listed for each action is its **canonical name** — the word echoed when the
action is triggered by a keyboard chord (e.g. Ctrl+Z echoes `UNDO`).

Context still applies: a word maps to one action, and that action's own enable rule gates it.
Typing a 2D `FILLET` in a part with no open sketch reports that the command is disabled — use
the 3D `FILLETEDGE` there. This is why the 2D and 3D edge blends have distinct words.

## Sketch — draw (2D geometry)

| AutoCAD word(s) | Oblikovati action |
| --- | --- |
| `LINE`, `L` | `Sketch.Line` |
| `CIRCLE`, `C` | `Sketch.Circle` |
| `ARC`, `A` | `Sketch.Arc` |
| `RECTANG`, `RECTANGLE`, `REC` | `Sketch.Rectangle` |
| `POINT`, `PO` | `Sketch.Point` |
| `POLYGON`, `POL` | `Sketch.Polygon` |
| `ELLIPSE`, `EL` | `Sketch.Ellipse` |
| `SPLINE`, `SPL` | `Sketch.Spline` |
| `SLOT` | `Sketch.Slot` |
| `TEXT`, `MTEXT`, `T` | `Sketch.Text` |
| `DIMENSION`, `DIM` | `Sketch.Dimension` |
| `AUTODIMENSION`, `AUTODIM` | `Sketch.AutoDimension` |
| `PROJECT`, `PROJECTGEOMETRY` | `Sketch.Project` |
| `SKETCH2D`, `NEWSKETCH` | `Sketch.Create2D` |
| `SKETCH3D` | `Sketch.Create3D` |
| `FINISHSKETCH`, `FINISH` | `Sketch.Finish` |

## Sketch — modify (2D editing)

| AutoCAD word(s) | Oblikovati action |
| --- | --- |
| `FILLET`, `F` | `Sketch.Fillet` |
| `CHAMFER`, `CHA` | `Sketch.Chamfer` |
| `TRIM`, `TR` | `Sketch.Trim` |
| `EXTEND`, `EX` | `Sketch.Extend` |
| `OFFSET`, `O` | `Sketch.Offset` |
| `MIRROR`, `MI` | `Sketch.Mirror` |
| `MOVE`, `M` | `Sketch.Move` |
| `COPY`, `CO`, `CP` | `Sketch.Copy` |
| `ROTATE`, `RO` | `Sketch.Rotate` |
| `SCALE`, `SC` | `Sketch.Scale` |
| `STRETCH`, `S` | `Sketch.Stretch` |
| `BREAK`, `BR` | `Sketch.Split` |
| `ARRAYRECT` | `Sketch.RectangularPattern` |
| `ARRAYPOLAR` | `Sketch.CircularPattern` |

## Part — create (solid features)

| AutoCAD word(s) | Oblikovati action |
| --- | --- |
| `EXTRUDE`, `EXT`, `E` | `Create.Extrude` |
| `REVOLVE`, `REV` | `Create.Revolve` |
| `SWEEP`, `SW` | `Create.Sweep` |
| `LOFT` | `Create.Loft` |
| `HELIX`, `COIL` | `Create.Coil` |
| `RIB` | `Create.Rib` |
| `EMBOSS` | `Create.Emboss` |
| `DECAL` | `Create.Decal` |

## Part — modify (solid editing)

| AutoCAD word(s) | Oblikovati action |
| --- | --- |
| `HOLE` | `Modify.Hole` |
| `BOSS` | `Modify.Boss` |
| `FILLETEDGE` | `Modify.Fillet` |
| `CHAMFEREDGE` | `Modify.Chamfer` |
| `SHELL` | `Modify.Shell` |
| `THREAD` | `Modify.Thread` |
| `THICKEN` | `Modify.Thicken` |
| `DRAFT`, `TAPER` | `Modify.Draft` |
| `OFFSETFACE` | `Modify.FaceOffset` |
| `DELETEFACE` | `Modify.DeleteFace` |
| `REPLACEFACE` | `Modify.ReplaceFace` |

## Surfaces

| AutoCAD word(s) | Oblikovati action |
| --- | --- |
| `SURFPATCH`, `PATCH` | `Surface.Patch` |
| `SURFTRIM` | `Surface.Trim` |
| `SURFEXTEND` | `Surface.Extend` |
| `SURFOFFSET` | `Surface.Offset` |
| `RULESURF`, `RULED` | `Surface.Ruled` |
| `SURFSTITCH`, `STITCH` | `Surface.Stitch` |
| `SCULPT` | `Surface.Sculpt` |
| `MIDSURFACE` | `Surface.MidSurface` |

## Work planes (datums)

AutoCAD has no direct equivalent, so these use Oblikovati-natural words.

| Word(s) | Oblikovati action |
| --- | --- |
| `WORKPLANE`, `PLANE` | `WorkPlane.Offset` |
| `MIDPLANE` | `WorkPlane.Midplane` |
| `PLANE3P` | `WorkPlane.ThreePoints` |
| `TANPLANE` | `WorkPlane.Tangent` |
| `NORMALPLANE` | `WorkPlane.NormalToAxis` |

## Application / editing

| AutoCAD word(s) | Oblikovati action | Default chord |
| --- | --- | --- |
| `SAVE`, `QSAVE` | `file.save` | Ctrl+S |
| `UNDO`, `U` | `edit.undo` | Ctrl+Z |
| `REDO`, `MREDO` | `edit.redo` | Ctrl+Y |
| `CANCEL` | `tool.cancel` | Esc |

## Adding to the map

Add a row to the relevant group in `app/cmdline/vocabulary.go` (first word = canonical
name; words must be unique across the whole table — `DefaultVocabulary` panics on a
duplicate). Point it at a **registered** command id or a reserved built-in action id, then
mirror the row here. `TestEveryVocabularyWordResolves` and `TestVocabularyActionsAreRealCommands`
keep the two in sync.
