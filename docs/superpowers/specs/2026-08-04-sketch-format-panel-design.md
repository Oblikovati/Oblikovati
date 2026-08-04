# Sketch Format panel (#2015)

Date: 2026-08-04
Issue: [#2015](https://github.com/Oblikovati/Oblikovati/issues/2015) — *Bug: Sketch, Missing
format panel and tools*

Depends on [#2014](2026-08-04-sketch-entity-creation-parity-design.md): the creation modes hook
into the `commitRecipe` seam that work introduces, so this branch stacks on it.

## Problem

The Sketch tab's Format panel is missing most of its tools, and the 3D Sketch tab has no Format
panel at all. The issue lists six items. What the survey found:

| Item | State today | Work |
|---|---|---|
| Construction | command exists, selection-only | add creation mode |
| Centerline | command exists, selection-only | add creation mode |
| Driven Dimensions | `SetDriven` exists, no command | command + creation mode |
| Center Point | nothing; `Point` carries no flags | flag + tool mode + distinct render |
| Show Format | nothing | display toggle suppressing overrides |
| Line Type / Colour / Thickness | **nothing at any layer** | model + persistence + DWG/DXF + 3 ribbon controls |

Two corrections to the issue's framing:

**Construction and Centerline are not missing — they are half-built.** Both exist in
`app/sketch_format.go` but only convert the current selection. The issue's own wording ("or
creates new geometry as…") describes a creation mode neither has.

**Show Format's polarity is inverted from its name.** The behaviour documentation states it
twice: toggling Show Format *on* suppresses formatting overrides and draws with default
attributes; toggling it *off* shows user formatting again. The implementation follows the
documented behaviour and names the internal state `suppressFormatOverrides` for what it does,
keeping "Show Format" only as the button label.

### Scope boundary: Sketch Only

The reference documentation also describes a **Sketch Only** toggle on the Format panel, but that
passage is from the *drawing* sketch environment. The part-sketch Format panel — which is what
the issue's screenshot shows and what this work targets — does not carry it. Out of scope, by
environment rather than by judgement.

### Panel contents per tab

| Item | 2D | 3D |
|---|---|---|
| Construction | yes | yes |
| Driven Dimensions | yes | yes |
| Show Format | yes | yes |
| Line Type / Colour / Thickness | yes | yes |
| Centerline | yes | no |
| Center Point | yes | no |

Centerline (a revolve/mirror axis) and Center Point (a hole-centre marker on a planar sketch) have
no meaning in a 3D sketch. The reference contract independently supports the rest applying in 3D:
`Sketch3D` exposes `OverrideColor`.

The canonical ribbon map at `architecture/mapping/inventor-ribbon-structure.md` records only
`[Construction, Centerline]` for this panel, so it is an abbreviated capture rather than an
authority; the issue's screenshot is the reference for panel contents.

## Design

### Format overrides

Per-entity overrides, matching the reference contract, where the properties sit on the concrete
entity types (`SketchCircle.OverrideColor`, `.LineType`, `.LineWeight`) rather than on the entity
base:

```go
// model/sketch/format.go
type EntityFormat struct {
    LineType   string      // "" ⇒ Default (inherit the sketch's line type)
    Color      types.Color // Source == AutomaticColorSource ⇒ Default
    LineWeight float64     // 0 ⇒ Default
}
```

Stored as `map[ID]EntityFormat` on the `Sketch`, with three accessors — `EntityFormat(id)`,
`SetEntityFormat(id, f)`, `ClearEntityFormat(id)`.

Default is expressed at two levels, and both are needed. Absence from the map means the entity has
no overrides at all — the common case. Within a stored format, each field can independently be
unset, so an entity can override its colour while inheriting its line type.

For the colour field that "unset" marker is the existing `AutomaticColorSource`, not a zero value:
`types.Color` already carries a `Source` enum whose members include the automatic (by-class)
colour, and the zero `Color` has `Source == 0`, which is not a member of that enum. Reusing the
existing marker keeps one meaning of "automatic" across the codebase instead of adding a second.

**Why a side table rather than fields on `entityBase`.** Absence means Default, which models the
semantics exactly and needs no sentinel values. An unstyled sketch — the overwhelmingly common
case — costs nothing. And nothing is added to `Point`, which is arena-allocated specifically to
stay small because point count dominates large DWG imports. Fields on `entityBase` would grow
every curve whether or not it carries an override, and would still leave `Point` needing separate
treatment since it has no `entityBase`.

Two obligations the side table carries, both easy to miss:

- **Pruning.** `DeleteEntities` already prunes orphan points and their constraints; it must drop
  format entries too. Otherwise a deleted entity's format leaks, and a later entity reusing the
  id would inherit it.
- **Copy propagation.** Pattern, mirror and block-instance copies duplicate entities, and the
  copies must carry the source's format. `copy_constraints.go` already holds a registry for this
  class of "carry X across a copy" concern; the format hooks in there.

### DWG/DXF round-trip

Import resolves each entity's BYLAYER through the file's layer table and stores the resolved
values as explicit per-entity overrides; export writes them per entity.

The drawing looks identical on re-export. Layer *structure* is deliberately not preserved —
exported entities carry explicit values rather than BYLAYER. This is the accepted trade: modelling
layers would mean a document-level layer table, `SketchEntity.Layer`, a ribbon layer picker and
importer/exporter work on both formats, for a part sketch with no style-management UI. The
exporter's doc comment records the loss so it is not discovered later as a surprise.

### Creation modes

All four toggles share one rule, which is what the issue's "changes selected … or creates new
geometry as" wording describes: **with geometry selected the button converts the selection; with
nothing selected it toggles a creation mode** applying to what is drawn next. One helper
implements it for all four so they cannot drift apart.

```go
// app/sketch_format_modes.go
type sketchFormatModes struct {
    construction bool // new geometry is construction
    centerline   bool // new lines are centerlines
    centerPoint  bool // the Point tool places centre points
    drivenDim    bool // new dimensions are driven
}
```

Modes are applied at one seam — `applyFormatModes(ents []Entity)`, called after geometry is
created. Every 2D creation already funnels through `commitRecipe` (#2014), so that is a single
call site; the 3D tools take the same call in their commits. A recipe's own construction geometry
(a centre rectangle's diagonals) is already flagged, so re-flagging under construction mode is a
harmless no-op.

**Driven dimensions interact with #2014's redundancy handling.** `applyRecipeFields` may already
demote a redundant dimension to driven. Both paths set the same flag, so the interaction is
benign, but the combination is tested rather than assumed: a locked field under driven mode must
produce exactly one driven dimension, not an error.

### Center Point

`Point` gains a single `centerPoint bool` — deliberately not an `entityBase`, to keep the
arena-allocated struct small for the same reason the format table is a side table. It renders as
a distinct glyph and round-trips in the `.obk`.

**Nothing consumes it yet.** The assembly hole takes an explicit 3D centre and there is no part
Hole feature, so this is a marker awaiting a consumer. Recorded here because the panel button will
otherwise look like it wires something up.

### Show Format

A persisted session toggle beside `HeadsUpDisplay` and `RelaxMode` in the sketch application
options, which are its siblings. When on, the renderer skips the override lookup and uses default
attributes — one branch, in one place.

### Rendering

`app.SketchEntityPattern` already resolves a pattern per entity (centerline → center pattern,
construction → dashed, otherwise the sketch-level override). It gains the per-entity override at
the front of that chain and returns colour and weight alongside the pattern. The Show Format
toggle short-circuits the override lookup.

### Ribbon control

The three lists need a control that does not exist: the ribbon has four button styles and no
dropdown. The nearest precedent is the split-button *variant* flyout, but that lists commands and
can neither show a current value nor draw a preview — and a preview is the point of these
controls, since a line type is picked by seeing its dash pattern.

So this adds a fifth style, `SelectionListButton`, with a value provider:

```go
WithSelectionList(items func() []ListItem, current func() int, choose func(int))
```

`ListItem` carries a label plus a preview kind — dash pattern, colour swatch, or weight sample.
All three draw from primitives the head already has (`linetype.Builtin` patterns, `DrawLine` with
a width), so there is no asset work.

### File split

`app/sketch_format.go` is 64 lines and would pass the 500-line limit with this work, so it becomes
three files by responsibility:

- `app/sketch_format.go` — the panel's command definitions
- `app/sketch_format_modes.go` — the dual selection/mode behaviour and the mode state
- `app/sketch_format_style.go` — the three lists

### API

One consolidated PR in `/Oblikovati.API`:

- `types.SketchEntityFormat{LineType SketchLineType; OverrideColor Color; LineWeight float64}`,
  reusing the existing `SketchLineType` whose empty value already means "inherit" — matching
  Default exactly.
- `types.ButtonStyle` += `SelectionListButton`.
- `wire` + `client`: `sketch.getEntityFormat` / `sketch.setEntityFormat`, plus get/set for the
  four creation modes and Show Format.

## Testing

| Layer | Gate |
|---|---|
| `model/sketch` | format set/get/clear; **pruning on entity delete**; **propagation across pattern/mirror/block copies** |
| `model/sketch` | the centre-point flag survives a round trip |
| `app` | the dual rule — with a selection it converts, with none it toggles the mode — one table across all four toggles |
| `app` | creation modes mark new geometry, including through the recipe path |
| `app` | driven mode combined with #2014's redundancy demotion yields exactly one driven dimension |
| `persistence` | format map, centre points and Show Format round-trip the `.obk` |
| exchange | DWG/DXF: BYLAYER resolves to explicit overrides on import and re-exports identically |
| `head/ui` | in-window render tests for the three list controls and their previews |
| live | MCP: set formats, screenshot, toggle Show Format, screenshot |

Coverage above 80% and duplication below 3% before any PR, with the full local suite, lint and
markdownlint run first. New head widgets take ≤6-method consumer interfaces so the audit I5
session-coupling ratchet holds.

## Build order

1. Format model, pruning and copy propagation (`model/sketch`) — pure headless
2. The four toggles and their creation modes
3. Show Format
4. Persistence (`.obk`, `yamlcodec`, application options)
5. DWG/DXF round-trip
6. `SelectionListButton` ribbon control and the three lists
7. API surface
8. Live verification

## Out of scope

- **Sketch Only** — a drawing-sketch Format panel item, not a part-sketch one.
- **A layer model.** Deliberately traded away; see the DWG/DXF section.
- **A Hole feature consuming centre points.** Center Point ships as a marker.
