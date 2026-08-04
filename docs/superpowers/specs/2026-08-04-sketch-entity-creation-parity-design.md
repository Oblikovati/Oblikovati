# Interactive sketch-entity creation parity (#2014)

Date: 2026-08-04
Issue: [#2014](https://github.com/Oblikovati/Oblikovati/issues/2014) — *Bug: Inconsistent
behaviour when creating sketch entities*

## Problem

Creating sketch geometry by clicking and dragging does not behave like the reference MCAD
application. The issue names five gaps: no entity preview, no construction lines, no
constraints, no dimension constraints, and no in-place dimension / parameter input.

Measured against the current code, the root cause is narrower and more serious than the
symptom list: **each shape is defined in more than one place, and no definition is
complete.**

- `sketch.AddPolygon` (`model/sketch/composite.go:119`) builds a correct, rigid hexagon —
  construction circumcircle, `PointOnCircle` per vertex, `EqualLength` per edge pair.
- `PolygonTool.Commit` (`app/sketch_geometry_tools.go:281`) reimplements the same polygon
  from scratch and applies no constraints at all.
- `app/sketch_preview.go` holds a third, cruder definition for the rubber band.

Three definitions, three behaviours. Every symptom in the issue follows from that.

### Measured baseline

DOF and redundancy measured through each tool's `Commit` on a fresh sketch:

| Shape | vars | DOF now | DOF target | Intrinsic parameters |
|---|---|---|---|---|
| Line | 4 | 3 | 3 | correct today (line inference works) |
| Circle (centre-radius) | 3 | 3 | 3 | correct today |
| Three-point circle | 3 | 3 | 3 | correct today |
| Arc (both variants) | 6 | 5 | 5 | correct today |
| Ellipse | 5 | 5 | 5 | correct today |
| Spline (both variants) | 2n | 2n | 2n | correct today |
| Two-point rectangle | 8 | **8** | **4** | corner x,y + width + height |
| Three-point rectangle | 8 | **8** | **5** | + base angle |
| Centre rectangle | 8 → 10 | **8** | **4** | centre x,y + width + height |
| Polygon (hexagon) | 12 → 15 | **12** | **4** | centre x,y + circumradius + rotation |
| Straight slot | 12 | **10** | **5** | centre x,y + angle + length + width |
| Arc slot (both variants) | 16 | **12** | **6** | centre x,y + radius + start + sweep + width |
| Text anchor, Point | 2 | 2 | 2 | correct today |

A committed two-point rectangle has **zero** constraints: drag a corner and it shears into
an arbitrary quadrilateral. The polygon is worse — twelve free points.

### Other current-state facts

- **Drag-create does not exist.** `handleViewportClick` (`head/ui/chrome_viewport.go:629`)
  fires on `native.IsItemClicked`, i.e. `ImGui::IsItemClicked`, i.e. mouse *press*. No
  release handler exists for creation tools, so press-drag-release places one point and the
  entity appears on a later unrelated press.
- **Preview covers 5 of 17 tools** — `PreviewPolyline` exists only for Line, Rectangle,
  Circle, Polygon, Spline, and returns a single flat polyline that cannot express
  construction or witness lines.
- **Auto-constraints fire for lines only** — `ApplyLineInference` is called from exactly two
  places, both in `LineTool.Commit` (`app/sketch_tools.go:156,160`).
- **The `#790` HUD is the wrong shape for this.** It is one panel beside the cursor with a
  fixed generic field pair (X/Y for the first point, Length/Angle after). It has no
  per-entity semantics, no witness-line placement, no lock state, and `HUDCommit` feeds a
  *point* to the command engine — it never creates a dimension.
- **Four settings are declared but dead.** `DisplayConstraintsOnCreation`,
  `EditDimensionsWhenCreated`, `OverConstrainedBehavior` and `PersistInferredConstraints`
  exist in `types/sketch_settings.go`, persist to `.obk` and round-trip in tests, but are
  read by zero lines of behaviour.

## Target behaviour

Derived from the six reference images on the issue. The visual language is consistent:

| Element | Style |
|---|---|
| Entity being created | solid violet |
| Construction geometry | dashed violet |
| Dimension witness / extension lines | fine dotted |
| Inferred-constraint glyph | yellow dot (coincident), yellow star-cross (midpoint/centre) |
| In-place value box, active | blue fill, white text |
| In-place value box, inactive | white fill, dark text |
| In-place value box, locked | white fill, dark text, padlock icon |

The defining frame is the two-point-rectangle image: the bottom half shows width typed and
**locked** with a padlock while height tracks the cursor; the top half shows the result —
that locked value became a **persistent driving dimension**, and the untyped direction got
none.

**The contract: typing a value during creation creates a real dimension constraint; leaving
a field to track the cursor creates nothing.**

## Design

### Layering

The centre of the design is one type: a `Recipe` in `model/sketch` declaring everything a
shape *is*.

```go
type Recipe struct {
    Points      []math.Point2       // indexed; everything below refers by index
    Entities    []RecipeEntity      // kind, point indices, Construction flag
    Constraints []RecipeConstraint  // kind + entity/point indices
    Fields      []RecipeField       // label, unit, live value, dimension to create, witness anchors
}
```

One builder per shape (`RectangleRecipe`, `SlotRecipe`, `CenterRectangleRecipe`, …) and one
`(*Sketch).Apply(Recipe)` that materialises it. Three consumers, one definition:

- **preview** renders the recipe (solid / dashed / dotted + glyphs + field boxes);
- **`Commit`** applies the same recipe;
- **`AddConstrained*`** constructors are `Apply(XRecipe(…))`, and the wire's composite path
  calls those.

Preview/commit divergence becomes structurally impossible, and the duplicate polygon
implementation is deleted.

| Layer | Owns | New files |
|---|---|---|
| `model/sketch` (GPL) | what a shape *is*: geometry, construction, constraints, dimensionable quantities | `recipe.go`, `recipe_apply.go`, `recipe_<family>.go`, `composite_constrained.go` |
| `app` (GPL) | interaction: drag-create state machine, field typing/lock, recipe assembly from clicks + cursor | `sketch_placement.go`, `sketch_placement_fields.go`, rewritten `sketch_preview.go` |
| `head/ui` (GPL, cgo) | painting only: line styles, glyphs, in-place boxes, padlock | `sketch_placement_overlay.go`, generalised `sketch_hud.go` |
| `Oblikovati.API` (Apache-2.0) | reference-parity settings | `types.HeadsUpDisplayOptions`, wire + client |

The raw `Add*` primitives stay untouched, so importers, pattern copies and add-ins that
state their own constraints keep the unconstrained path. This is the opt-in boundary that
keeps the procedural-add-in redundancy trap closed: an add-in building a fully explicit
profile calls the raw constructor and receives no inferred duplicates.

### Shape catalogue

Six shapes are already correct and need only preview and fields. Six are floppy and gain
constraints:

| Shape | Constraints added |
|---|---|
| Two-point rectangle | `Horizontal` ×2, `Vertical` ×2 |
| Three-point rectangle | `Perpendicular` ×3 |
| Centre rectangle | `Horizontal` ×2, `Vertical` ×2, `Midpoint`(centre, diagonal) |
| Polygon | none new — delete the duplicate in `PolygonTool.Commit`, route through `sketch.AddPolygon` |
| Straight slot | `Tangent` ×4, `EqualRadius` ×1 |
| Arc slots | `Tangent` ×4, `EqualRadius` on the caps, `Concentric` on the offset arcs |

The slot deliberately omits a parallel-sides constraint: parallelism follows from four
tangencies plus equal radii, so adding it would be redundant.

The two slot families are the arithmetic worth checking up front, since both must land on
their target with zero redundancy:

- straight slot — 12 vars, 2 circularity (present today) + 4 tangency + 1 equal-radius =
  7 equations, DOF 5;
- arc slot — 16 vars, 4 circularity (present today) + 4 tangency + 1 equal-radius +
  1 concentric = 10 equations, DOF 6.

**Construction-geometry rule:** persist construction geometry only when it anchors a
constraint or carries a dimension the tool's definition needs.

- centre rectangle → two diagonals (they anchor the centre via `Midpoint`)
- all three slots → a centreline (it carries the length dimension)
- polygon → the circumcircle (already the case; this is the in-repo precedent for the rule)
- everything else → none. The dashed diameter line in the circle reference image is a
  *dimension witness*, not geometry; the control-vertex spline's control polygon is
  preview-only.

**Fields per tool** (Tab cycles them):

| Tool | Fields |
|---|---|
| Line | Length, Angle |
| Rectangle two-point / centre | Width, Height |
| Rectangle three-point | Length, Angle → Width |
| Circle (both variants) | Diameter |
| Arc (both variants) | Radius → Sweep angle |
| Slot (straight) | Length, Angle → Width |
| Slot (arc, both variants) | Radius, Sweep → Width |
| Ellipse | Major, Angle → Minor |
| Polygon | Diameter, Angle |
| Text | Height, Angle |
| Point, splines | none (pointer input X/Y only) |
| Fillet / Chamfer | Radius / Distance |

Locked fields become dimensions through the existing `AddDistance` / `AddRadius` /
`AddDiameter` / `AddAngle` constructors.

**Unit-strictness trap.** The parameter engine is unit-strict: a locked 10 mm must be
emitted as the expression `"10 mm"`. A bare `"10"` would silently mean 10 cm (the kernel
length unit). Field-to-expression formatting carries the document's preferred unit
explicitly, and a test pins the emitted string.

### Interaction

**Drag-create.** One state machine in `app/sketch_placement.go` replaces the press-only
`sketchClick` path for `PlaneClickTool`s:

- **press** — place a point (as today); remember the pixel, clear the moved flag
- **move while held** — set moved once past a 4 px slop (mirroring `orbitPivotClickSlop`)
- **release** — if moved, place a second point at the release position and auto-commit if
  the tool is ready; if not moved, do nothing

Click-click and click-drag-release become the same path: a drag means the second point
arrived on release instead of on the next press. Tools needing three or more points (arc,
three-point rectangle, ellipse, arc slots) get drag for points 1→2 and click-click for the
rest with no special-casing, matching the reference application.

Wiring is a new `updateSketchPlacement(s)` in `handleViewportSelection`, ahead of
`updateSketchDrag`. `updateBoxSelect` already stands down via `ActiveToolConsumesClicks`.

**Field input.** `sketchHUD`'s fixed `[2]string` generalises to a variable-length field list
driven by the recipe, each field carrying `typed string` and `locked bool`:

- typing a digit engages and fills the active field
- **Tab** locks the active field and moves to the next — the padlock in the reference image
- **Enter** locks the active field and commits the shape immediately
- **Esc** clears typed state; a second Esc cancels the tool

**Locked fields feed back into the recipe builder.** Once width is locked at 10, dragging
changes only height — exactly what the reference image shows. The builder signature is
`(placedPoints, cursor, lockedValues) → Recipe`, with locked values overriding
cursor-derived ones, so one path drives preview, commit and dimension creation.

The existing X/Y panel is retained as *pointer input*: shown for a shape's first point,
replaced by dimension fields once a reference point exists — precisely the decision
`hudReference` already makes.

### Rendering

`head/ui/sketch_placement_overlay.go`:

| Element | Treatment |
|---|---|
| Preview geometry | solid, `previewColor` |
| Preview construction geometry | dashed via `linetype.Builtin(types.SketchLineDashed)` — the same pattern committed construction geometry uses, so the preview looks like the result |
| Witness lines to field boxes | fine dotted |
| Constraint glyphs | yellow at the anchor, gated on `DisplayConstraintsOnCreation` |
| Field box, active | blue fill, white text |
| Field box, inactive | white fill, dark text |
| Field box, locked | white fill, dark text, padlock |

Boxes sit at the witness-line midpoint, offset perpendicular, clamped inside the viewport.
The padlock is drawn from `DrawRectFilled` plus three short `DrawLine` calls for the
shackle: it scales with the font and needs no asset pipeline. Boxes stay custom-painted
rather than ImGui text inputs, because the typing state machine already lives in `app` and
real inputs would fight the viewport for focus.

### API

One consolidated PR in `/Oblikovati.API`:

- `types.HeadsUpDisplayOptions` — `Enabled`, `PointerInputEnabled`,
  `PointerInputInCartesianCoordinates`, `DimensionInputEnabled`,
  `DimensionInputInCartesianCoordinates`, `CreateDimensionsOnValueInput`. Defaults: all on,
  pointer input Cartesian, dimension input polar (length + angle), matching the line
  reference image.
- `wire` method constants and DTO for get/set, plus a typed `api/client` method group.
- `sketch.addEntity` composite kinds route through the constrained constructors and honour
  the document's `AutoApplyConstraints`; new kinds for the variants (three-point rectangle,
  centre rectangle, arc slots).

**One deliberate deviation from the reference API.** The reference exposes
`CreateDimensionsOnValueInput` on both its heads-up-display options and its sketch
constraint settings. This design puts it only on `HeadsUpDisplayOptions`. Two
independently-settable copies of one flag can disagree, and the document-scoped copy has no
distinct meaning here.

**Four dead settings come alive**: `DisplayConstraintsOnCreation` gates the glyphs,
`EditDimensionsWhenCreated` opens the value editor after a dimension is created,
`OverConstrainedBehavior` decides what a redundant dimension does, and
`PersistInferredConstraints` decides whether inferred constraints persist or remain hints.
`DisplayConstraintsOnCreation` currently defaults to `false` while the reference images
plainly show glyphs during creation, so its default flips to `true`; this updates the
pinned assertion in `types/sketch_settings_test.go:21`.

### Error handling

`Apply` is the single choke point, and redundancy is the real risk: a duplicated constraint
reports DOF 0 while the solver settles on a degenerate, self-intersecting configuration that
extrudes to nothing — silently, because a DOF-only check passes.

1. Apply geometry and geometric constraints.
2. For each locked field, trial-add the dimension and re-run `AnalyzeConstraints()`.
3. If `Redundant` rose, act per `OverConstrainedBehavior`: `ApplyDriven` (default) demotes
   it to a reference dimension, `ApplyDriving` accepts the redundancy, `Prompt` asks the
   user. Headless and wire callers fall back to `ApplyDriven`.
4. If the solve fails to converge, roll the whole recipe back and return an error.

`Apply` is atomic: created entity IDs are tracked and deleted on failure, so a rejected
recipe never leaves orphan geometry.

Degenerate inputs (zero-length line, coincident slot centres, collinear three-point circle
or arc) are rejected by the builder with a message naming the offending value and the
expected shape. The preview draws nothing for a degenerate cursor rather than erroring every
frame.

## Testing

| Layer | Gate |
|---|---|
| `model/sketch` | per-recipe table test pinning **exact DOF and `Redundant == 0`**, mutation-proofed by deleting a constraint and confirming the test fails |
| `model/sketch` | recipe↔commit equivalence: applied geometry matches the recipe's preview polylines, so preview cannot drift from result |
| `app` | placement machine: press-release-in-place equals click-click; press-drag-release places two points; slop boundary; three-point tools drag then click |
| `app` | fields: Tab locks and advances, Enter commits, Esc clears, locked value overrides the cursor, emitted expression carries its unit |
| `app` | locked field produces a driving dimension; untyped field produces none |
| `head/ui` | in-window render tests per the existing `*_inwindow_test.go` pattern: dashed construction present, active/inactive/locked box states distinguishable |
| live | MCP bridge drag-creates each family in the running head, screenshot-verifying the mid-drag overlay |
| regression | the six already-correct shapes keep their current DOF — no accidental over-constraining |

Coverage above 80% and duplication below 3% before any PR, with the full local suite,
lint and markdownlint run first.

## Build order

Each stage is independently verifiable:

1. `Recipe` + `Apply` + the six constraint fixes in `model/sketch` — pure headless
2. Route tools and the wire through the recipes; delete the duplicate polygon
3. Drag-create placement state machine
4. Fields and dimension creation
5. Rendering
6. API and settings
7. Live verification

## Out of scope

- The `Prompt` over-constrained behaviour needs a modal choice in `head/ui`; the headless and
  wire paths fall back to `ApplyDriven`. The modal itself is deferred.
- Selective per-family relax (`GeometricConstraintsToRemoveInRelaxMode`) stays out, as
  already recorded in `types/sketch_settings.go`.
- Issue #2015 (sketch Format panel and tools) is unrelated and separately tracked.
