# Sketch Format Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the Sketch 2D/3D Format panel — Construction and Centerline as creation modes,
Center Point, Driven Dimensions, Show Format, and per-entity line type / colour / thickness that
round-trips DWG (issue #2015).

**Architecture:** Per-entity format overrides live in a side table on the `Sketch`, keyed by
entity id, where absence means Default. The four toggles share one rule — convert the selection
when there is one, otherwise flip a creation mode — applied at the single `commitRecipe` seam that
#2014 introduced.

**Tech Stack:** Go (headless core), cgo + Vulkan + Dear ImGui (`head/`), the sibling Apache-2.0
module `oblikovati.org/api`.

Spec: `docs/superpowers/specs/2026-08-04-sketch-format-panel-design.md`

**Depends on #2014** (branch `fix/2014-sketch-entity-creation-parity`): the creation modes hook
into `commitRecipe`, which that work introduces. This branch is stacked on it.

## Global Constraints

- Functions 4–20 lines; files under 500 lines; max 2 levels of indentation; early returns.
- Types explicit — no `any`, no untyped functions.
- Every new exported `.go` file carries `SPDX-License-Identifier: GPL-2.0-only` (this module) or
  `Apache-2.0` (in `../Oblikovati.API`); run `scripts/add-spdx-headers.py`.
- Never re-declare a DTO or method-name string outside `api/wire` — import it.
- Public API additions land contract-first in `../Oblikovati.API`, then implementation here.
- Default is expressed twice and both are needed: absence from the format map means no overrides
  at all; within a stored format, `LineType == ""`, `Color.Source == types.AutomaticColorSource`
  and `LineWeight == 0` each mean that field inherits.
- `Point` gets a single `bool`, never an `entityBase` — it is arena-allocated to stay small
  because point count dominates large DWG imports.
- Show Format's documented behaviour is the inverse of its name: **on suppresses overrides**. The
  internal state is named `suppressFormatOverrides`; "Show Format" is only the button label.
- New head widgets take ≤6-method consumer interfaces (audit I5 ratchet, `arrowSession` pattern),
  each with a compile-time assertion.
- Tests run with `go test ./...`; lint with `make lint` (funlen 20); docs with `make docs-lint`.
- Coverage > 80%, duplication < 3% before any PR.
- Do not open a PR. The user opens PRs explicitly.

---

## File Structure

| File | Responsibility |
|---|---|
| `model/sketch/format.go` (create) | `EntityFormat`, the side table, accessors, prune + copy hooks |
| `app/sketch_format.go` (modify) | the panel's command definitions only |
| `app/sketch_format_modes.go` (create) | creation-mode state + the dual selection/mode rule |
| `app/sketch_format_style.go` (create) | the three lists: read/write format on selection or mode |
| `head/ui/ribbon_selection_list.go` (create) | the `SelectionListButton` control and its previews |
| `../Oblikovati.API/types/sketch_entity_format.go` (create) | `SketchEntityFormat` |

---

### Task 1: Format side table with prune and copy propagation

**Files:**

- Create: `model/sketch/format.go`
- Modify: `model/sketch/sketch.go` (the `Sketch` struct; `deleteEntity`)
- Test: `model/sketch/format_test.go`

**Interfaces:**

- Produces: `EntityFormat` struct; `(*Sketch).EntityFormat(id ID) (EntityFormat, bool)`,
  `(*Sketch).SetEntityFormat(id ID, f EntityFormat)`, `(*Sketch).ClearEntityFormat(id ID)`,
  `(*Sketch).CopyEntityFormat(from, to ID)`.

- [ ] **Step 1: Write the failing test**

```go
// model/sketch/format_test.go
package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// redFormat is a format with every field overridden, for tests that only care that a format
// travels rather than what it holds.
func redFormat() EntityFormat {
	return EntityFormat{LineType: "dashed", Color: types.NewColor(255, 0, 0), LineWeight: 0.5}
}

func TestEntityFormatSetGetClear(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))

	if _, ok := s.EntityFormat(l.EntityID()); ok {
		t.Fatal("a new entity must have no format — absence is Default")
	}
	s.SetEntityFormat(l.EntityID(), redFormat())
	got, ok := s.EntityFormat(l.EntityID())
	if !ok || got != redFormat() {
		t.Fatalf("format = %+v ok=%v, want the stored override", got, ok)
	}
	s.ClearEntityFormat(l.EntityID())
	if _, ok := s.EntityFormat(l.EntityID()); ok {
		t.Error("clearing must return the entity to Default")
	}
}

// A deleted entity's format must go with it: otherwise it leaks, and a later entity reusing the
// id would silently inherit it.
func TestEntityFormatPrunedOnDelete(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	id := l.EntityID()
	s.SetEntityFormat(id, redFormat())

	s.DeleteEntities([]Entity{l})
	if _, ok := s.EntityFormat(id); ok {
		t.Error("deleting an entity must drop its format")
	}
	if n := s.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}

func TestCopyEntityFormat(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	b := s.Lines().AddByTwoPoints(math.P2(0, 5), math.P2(10, 5))
	s.SetEntityFormat(a.EntityID(), redFormat())

	s.CopyEntityFormat(a.EntityID(), b.EntityID())
	got, ok := s.EntityFormat(b.EntityID())
	if !ok || got != redFormat() {
		t.Errorf("copied format = %+v ok=%v, want the source's", got, ok)
	}
}

// Copying from an unstyled entity must not create an entry — that would turn Default into an
// explicit empty override and defeat the absence-means-Default rule.
func TestCopyEntityFormatFromDefaultCreatesNothing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	b := s.Lines().AddByTwoPoints(math.P2(0, 5), math.P2(10, 5))

	s.CopyEntityFormat(a.EntityID(), b.EntityID())
	if n := s.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./model/sketch/ -run 'EntityFormat|CopyEntityFormat' -v`
Expected: FAIL — `undefined: EntityFormat`.

- [ ] **Step 3: Write `model/sketch/format.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/api/types"

// Per-entity format overrides (#2015): the line type, colour and line weight the Format panel's
// three lists set on selected geometry, and the values a DWG import carries in from the file's
// layer table.
//
// They live in a side table keyed by entity id rather than on the entities themselves. Absence
// means Default, which models the semantics with no sentinel values and costs an unstyled sketch
// nothing — and it keeps Point untouched, which matters because Point is arena-allocated to stay
// small and point count dominates a large DWG import.
//
//	sk.SetEntityFormat(line.EntityID(), sketch.EntityFormat{LineType: "dashed"})

// EntityFormat is one entity's format overrides. Each field independently means "inherit" when
// unset, so an entity can override its colour while taking the sketch's line type.
type EntityFormat struct {
	// LineType is the .lin pattern name; "" inherits the sketch's line type.
	LineType string
	// Color overrides the entity's colour; a Color whose Source is types.AutomaticColorSource
	// inherits instead. The zero Color is NOT the marker — its Source is 0, which is not a
	// member of the enum.
	Color types.Color
	// LineWeight is the stroke width in millimetres; 0 inherits.
	LineWeight float64
}

// IsDefault reports whether the format overrides nothing, in which case it need not be stored.
func (f EntityFormat) IsDefault() bool {
	return f.LineType == "" && f.LineWeight == 0 && !f.Color.IsOverride()
}

// EntityFormat returns the entity's overrides, or ok=false when it takes the sketch defaults.
func (s *Sketch) EntityFormat(id ID) (EntityFormat, bool) {
	f, ok := s.formats[id]
	return f, ok
}

// SetEntityFormat stores an entity's overrides. A format that overrides nothing clears the entry
// instead of storing an empty one, so "no overrides" has exactly one representation.
func (s *Sketch) SetEntityFormat(id ID, f EntityFormat) {
	if f.IsDefault() {
		s.ClearEntityFormat(id)
		return
	}
	if s.formats == nil {
		s.formats = map[ID]EntityFormat{}
	}
	s.formats[id] = f
}

// ClearEntityFormat returns an entity to the sketch defaults.
func (s *Sketch) ClearEntityFormat(id ID) { delete(s.formats, id) }

// EntityFormatCount reports how many entities carry overrides — the count persistence writes and
// the prune tests assert against.
func (s *Sketch) EntityFormatCount() int { return len(s.formats) }

// CopyEntityFormat carries one entity's overrides onto another, for the pattern, mirror and
// block-instance copies. Copying from an unstyled entity stores nothing.
func (s *Sketch) CopyEntityFormat(from, to ID) {
	f, ok := s.formats[from]
	if !ok {
		return
	}
	s.SetEntityFormat(to, f)
}
```

- [ ] **Step 4: Add `IsOverride` to the API's Color**

In `../Oblikovati.API/types/color.go`, add:

```go
// IsOverride reports whether the colour is an explicit per-object override rather than an
// inherited one (automatic, layer or sheet). Callers storing optional colour overrides use it as
// the "is this set?" test, so the meaning of automatic lives in one place.
func (c Color) IsOverride() bool { return c.Source == OverrideColorSource }
```

- [ ] **Step 5: Add the map to the Sketch and prune it on delete**

In `model/sketch/sketch.go`, add to the `Sketch` struct beside `ents`:

```go
	formats map[ID]EntityFormat // per-entity format overrides (#2015); absent ⇒ sketch defaults
```

and extend `deleteEntity` (currently `sketch.go:392`):

```go
func (s *Sketch) deleteEntity(e Entity) {
	s.removeEntity(e)
	s.dropFromCollection(e)
	s.ClearEntityFormat(e.EntityID()) // the format dies with its entity (#2015)
}
```

- [ ] **Step 6: Run the tests**

Run: `go test ./model/sketch/ -run 'EntityFormat|CopyEntityFormat' -v`
Expected: PASS — all four.

- [ ] **Step 7: Mutation-proof the prune**

Delete the `s.ClearEntityFormat(...)` line from `deleteEntity`, run
`go test ./model/sketch/ -run TestEntityFormatPrunedOnDelete`, confirm it FAILS, then restore.

- [ ] **Step 8: Commit**

```bash
gofmt -w model/sketch/format.go model/sketch/format_test.go model/sketch/sketch.go
cd ../Oblikovati.API && gofmt -w types/color.go && GOWORK=off go test ./types/ && git add -A && git commit -m "feat(types): Color.IsOverride for optional colour overrides (#2015)" && cd ../Oblikovati
git add -A && git commit -m "feat(sketch): per-entity format overrides in a side table (#2015)"
```

---

### Task 2: Format propagation across copies

**Files:**

- Modify: `model/sketch/copy_constraints.go` (the clone path)
- Test: `model/sketch/format_copy_test.go`

**Interfaces:**

- Consumes: `(*Sketch).CopyEntityFormat(from, to ID)` from Task 1.

- [ ] **Step 1: Write the failing test**

```go
// model/sketch/format_copy_test.go
package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// A pattern or mirror copy must carry the source's format, or a styled sketch loses its
// formatting the moment it is patterned.
func TestFormatCarriesAcrossClone(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	src := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.SetEntityFormat(src.EntityID(), redFormat())

	dst := s.Lines().AddByTwoPoints(math.P2(0, 5), math.P2(10, 5))
	m := &cloneMap{points: map[*Point]*Point{}, entities: map[Entity]Entity{src: dst}}
	s.carryEntityFormats(m)

	got, ok := s.EntityFormat(dst.EntityID())
	if !ok || got != redFormat() {
		t.Errorf("cloned format = %+v ok=%v, want the source's", got, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./model/sketch/ -run TestFormatCarriesAcrossClone -v`
Expected: FAIL — `s.carryEntityFormats undefined`.

- [ ] **Step 3: Implement `carryEntityFormats`**

Append to `model/sketch/format.go`:

```go
// carryEntityFormats copies each cloned entity's format onto its clone, so a pattern, mirror or
// block instance keeps the formatting of the geometry it came from (#2015).
func (s *Sketch) carryEntityFormats(m *cloneMap) {
	for src, dst := range m.entities {
		s.CopyEntityFormat(src.EntityID(), dst.EntityID())
	}
}
```

- [ ] **Step 4: Call it from the clone path**

Find where `cloneMap` is populated and constraints are carried (grep
`grep -n "carryFrom" model/sketch/copy_constraints.go`), and call `s.carryEntityFormats(m)` once
the entity map is complete, beside the constraint carry.

- [ ] **Step 5: Run the tests**

Run: `go test ./model/sketch/ -run 'Format' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
gofmt -w model/sketch/
git add -A && git commit -m "feat(sketch): format overrides carry across pattern/mirror copies (#2015)"
```

---

### Task 3: Centre-point flag

**Files:**

- Modify: `model/sketch/entities.go` (the `Point` struct)
- Modify: `app/sketch_preview.go` (`PointTool.PendingRecipe`) and `model/sketch/recipe.go`
- Test: `model/sketch/center_point_test.go`

**Interfaces:**

- Produces: `(*Point).IsCenterPoint() bool`, `(*Point).SetCenterPoint(bool)`;
  `RecipeEntity.CenterPoint bool`.

- [ ] **Step 1: Write the failing test**

```go
// model/sketch/center_point_test.go
package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// A centre point is a hole-centre marker. Nothing consumes it yet — the assembly hole takes an
// explicit 3D centre and there is no part Hole feature — but it renders distinctly and must
// survive a round trip so a future consumer finds it.
func TestCenterPointFlag(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p := s.Points().Add(math.P2(1, 2))
	if p.IsCenterPoint() {
		t.Fatal("a plain point must not start as a centre point")
	}
	p.SetCenterPoint(true)
	if !p.IsCenterPoint() {
		t.Error("SetCenterPoint(true) must take effect")
	}
}

func TestRecipePointCanBeACenterPoint(t *testing.T) {
	r := Recipe{
		Points:   []math.Point2{math.P2(3, 4)},
		Entities: []RecipeEntity{{Kind: RecipePoint, Points: []int{0}, CenterPoint: true}},
	}
	s := NewSketches().Add(XYPlane())
	ents, _, err := s.Apply(r, types.OverConstrainedApplyDriven)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	p, ok := ents[0].(*Point)
	if !ok {
		t.Fatalf("entity = %T, want *Point", ents[0])
	}
	if !p.IsCenterPoint() {
		t.Error("a RecipePoint with CenterPoint set must produce a centre point")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./model/sketch/ -run 'CenterPoint' -v`
Expected: FAIL — `p.IsCenterPoint undefined`.

- [ ] **Step 3: Add the flag to `Point`**

In `model/sketch/entities.go`, extend the struct (currently `{id, X, Y}`):

```go
type Point struct {
	id ID
	X  math.Scalar
	Y  math.Scalar
	// centerPoint marks a hole-centre marker rather than a plain sketch point (#2015). A single
	// bool, deliberately not an entityBase: Point is arena-allocated to stay small because point
	// count dominates a large DWG import.
	centerPoint bool
}

// IsCenterPoint reports whether the point is a hole-centre marker.
func (p *Point) IsCenterPoint() bool { return p.centerPoint }

// SetCenterPoint marks the point as a hole-centre marker (or back to a plain point).
func (p *Point) SetCenterPoint(c bool) { p.centerPoint = c }
```

- [ ] **Step 4: Carry the flag through a recipe**

In `model/sketch/recipe.go`, add to `RecipeEntity`:

```go
	// CenterPoint makes a RecipePoint a hole-centre marker rather than a plain point (#2015).
	CenterPoint bool
```

and in `model/sketch/recipe_apply.go`, change the `RecipePoint` case of `buildRecipeEntity`:

```go
	case RecipePoint:
		p := s.listStandalonePoint(pts[re.Points[0]])
		p.SetCenterPoint(re.CenterPoint)
		return p, nil
```

- [ ] **Step 5: Run the tests**

Run: `go test ./model/sketch/ -run 'CenterPoint' -v`
Expected: PASS.

- [ ] **Step 6: Confirm the arena is unaffected in behaviour**

Run: `go test ./model/sketch/`
Expected: `ok` — the arena block allocator and the point-heavy import tests still pass.

- [ ] **Step 7: Commit**

```bash
gofmt -w model/sketch/
git add -A && git commit -m "feat(sketch): centre-point marker on sketch points (#2015)"
```

---

### Task 4: Creation modes and the dual selection/mode rule

**Files:**

- Create: `app/sketch_format_modes.go`
- Modify: `app/sketch_format.go` (commands), `app/session.go` (state field)
- Modify: `app/sketch_placement_fields.go` (`commitRecipe` applies the modes)
- Test: `app/sketch_format_modes_test.go`

**Interfaces:**

- Consumes: `(*Session).commitRecipe(r sketch.Recipe) error` from #2014.
- Produces: `(*Session).ToggleConstruction() int`, `ToggleCenterline() int`,
  `ToggleCenterPoint() int`, `ToggleDrivenDimension() int`, each returning how many entities were
  converted (0 ⇒ it flipped the creation mode instead); `ConstructionMode() bool`,
  `CenterlineMode() bool`, `CenterPointMode() bool`, `DrivenDimensionMode() bool`.

- [ ] **Step 1: Write the failing test**

```go
// app/sketch_format_modes_test.go
package app

import (
	"testing"

	"oblikovati.org/math"
)

// The dual rule the issue describes: with geometry selected the button converts the selection;
// with nothing selected it toggles a creation mode for what is drawn next.
func TestFormatToggleConvertsSelection(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})

	if n := s.ToggleConstruction(); n != 1 {
		t.Fatalf("converted = %d, want 1", n)
	}
	if !l.IsConstruction() {
		t.Error("the selected line must become construction")
	}
	if s.ConstructionMode() {
		t.Error("converting a selection must NOT also flip the creation mode")
	}
}

func TestFormatToggleFlipsModeWithNoSelection(t *testing.T) {
	s, _ := sketchSession(t)
	if n := s.ToggleConstruction(); n != 0 {
		t.Fatalf("converted = %d, want 0 — nothing was selected", n)
	}
	if !s.ConstructionMode() {
		t.Error("with no selection the button must arm the creation mode")
	}
	s.ToggleConstruction()
	if s.ConstructionMode() {
		t.Error("a second press must disarm it")
	}
}

// Armed construction mode marks what is drawn next, through the recipe commit path.
func TestConstructionModeMarksNewGeometry(t *testing.T) {
	s, sk := sketchSession(t)
	s.ToggleConstruction() // arm the mode
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0), math.P2(10, 8)}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	for i := 0; i < sk.Lines().Count(); i++ {
		if !sk.Lines().Item(i).IsConstruction() {
			t.Fatalf("line %d is not construction — the mode did not apply", i)
		}
	}
}

// Centerline mode marks new lines; a centreline is always construction too.
func TestCenterlineModeMarksNewLines(t *testing.T) {
	s, sk := sketchSession(t)
	s.ToggleCenterline()
	tool := NewLineTool()
	tool.points = []math.Point2{math.P2(0, 0), math.P2(10, 0)}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	l := sk.Lines().Item(0)
	if !l.IsCenterline() || !l.IsConstruction() {
		t.Errorf("centerline=%v construction=%v, want both true", l.IsCenterline(), l.IsConstruction())
	}
}

// Driven mode combined with #2014's redundancy demotion must yield one driven dimension, not an
// error — both paths set the same flag.
func TestDrivenModeWithLockedField(t *testing.T) {
	s, sk := sketchSession(t)
	s.ToggleDrivenDimension()
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	for _, r := range "10" {
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab()
	tool.corners = append(tool.corners, math.P2(1, 0.8))
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	dims := sk.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("dimensions = %d, want 1", len(dims))
	}
	if !dims[0].Driven() {
		t.Error("driven mode must make the new dimension driven")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/ -run 'TestFormatToggle|Mode' -v`
Expected: FAIL — `s.ConstructionMode undefined`.

- [ ] **Step 3: Write `app/sketch_format_modes.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/sketch"

// The Format panel's four toggles share one rule (#2015): with geometry selected the button
// CONVERTS the selection; with nothing selected it flips a CREATION MODE that applies to whatever
// is drawn next. One helper implements it for all four so they cannot drift apart.
//
// Before this, Construction and Centerline converted a selection and had no creation mode at all,
// which is half of what the panel is for.

// sketchFormatModes is the armed creation state: what newly drawn geometry becomes.
type sketchFormatModes struct {
	construction bool // new geometry is construction
	centerline   bool // new lines are centerlines
	centerPoint  bool // the Point tool places centre points
	drivenDim    bool // new dimensions are driven
}

// ConstructionMode, CenterlineMode, CenterPointMode and DrivenDimensionMode report which creation
// modes are armed, so the ribbon can draw its buttons pressed.
func (s *Session) ConstructionMode() bool     { return s.formatModes.construction }
func (s *Session) CenterlineMode() bool       { return s.formatModes.centerline }
func (s *Session) CenterPointMode() bool      { return s.formatModes.centerPoint }
func (s *Session) DrivenDimensionMode() bool  { return s.formatModes.drivenDim }

// ToggleConstruction converts the selected geometry to construction, or — with nothing selected —
// arms construction mode. It returns how many entities it converted, so 0 means it flipped the
// mode instead.
func (s *Session) ToggleConstruction() int {
	return s.convertOrArm(&s.formatModes.construction, func(e sketch.Entity) bool {
		c, ok := e.(interface {
			IsConstruction() bool
			SetConstruction(bool)
		})
		if !ok {
			return false
		}
		c.SetConstruction(!c.IsConstruction())
		return true
	})
}

// ToggleCenterline converts the selected lines to centerlines, or arms centerline mode.
func (s *Session) ToggleCenterline() int {
	return s.convertOrArm(&s.formatModes.centerline, func(e sketch.Entity) bool {
		l, ok := e.(*sketch.Line)
		if !ok {
			return false
		}
		l.SetCenterline(!l.IsCenterline())
		return true
	})
}

// ToggleCenterPoint converts the selected points to centre points, or arms centre-point mode.
func (s *Session) ToggleCenterPoint() int {
	return s.convertOrArm(&s.formatModes.centerPoint, func(e sketch.Entity) bool {
		p, ok := e.(*sketch.Point)
		if !ok {
			return false
		}
		p.SetCenterPoint(!p.IsCenterPoint())
		return true
	})
}

// ToggleDrivenDimension flips the selected dimensions between driving and driven, or arms driven
// mode for new dimensions.
func (s *Session) ToggleDrivenDimension() int {
	n := 0
	for _, d := range s.selectedDimensions() {
		d.SetDriven(!d.Driven())
		n++
	}
	if n == 0 {
		s.formatModes.drivenDim = !s.formatModes.drivenDim
	}
	return n
}

// convertOrArm applies convert to every selected sketch entity it accepts; when the selection
// yields none, it flips the armed mode instead. This is the dual rule, in one place.
func (s *Session) convertOrArm(mode *bool, convert func(sketch.Entity) bool) int {
	n := 0
	for _, e := range s.selectedSketchEntities() {
		if convert(e) {
			n++
		}
	}
	if n == 0 {
		*mode = !*mode
	}
	return n
}

// applyFormatModes marks freshly created geometry per the armed creation modes. It is called from
// the single commit seam every 2D tool funnels through, so a mode cannot be honoured by some
// tools and not others.
func (s *Session) applyFormatModes(ents []sketch.Entity) {
	for _, e := range ents {
		s.applyFormatModesTo(e)
	}
}

// applyFormatModesTo marks one entity per the armed modes. A recipe's own construction geometry
// is already flagged, so re-flagging it is a harmless no-op.
func (s *Session) applyFormatModesTo(e sketch.Entity) {
	if s.formatModes.construction {
		if c, ok := e.(interface{ SetConstruction(bool) }); ok {
			c.SetConstruction(true)
		}
	}
	if l, ok := e.(*sketch.Line); ok && s.formatModes.centerline {
		l.SetCenterline(true)
	}
	if p, ok := e.(*sketch.Point); ok && s.formatModes.centerPoint {
		p.SetCenterPoint(true)
	}
}
```

- [ ] **Step 4: Add `selectedDimensions` and the session field**

In `app/session.go`, beside `placementFields`:

```go
	formatModes sketchFormatModes // Format-panel creation modes (#2015)
```

and append to `app/sketch_format_modes.go`:

```go
// selectedDimensions returns the dimension constraints in the current selection.
func (s *Session) selectedDimensions() []*sketch.DimensionConstraint {
	var out []*sketch.DimensionConstraint
	for _, it := range s.Selection().Items() {
		if h, ok := it.(SketchDimensionHandle); ok {
			out = append(out, h.Dim)
		}
	}
	return out
}
```

Check the actual dimension selection handle name first with
`grep -rn "DimensionHandle\|SketchDimension.*struct" app/*.go | grep -v _test`, and use whatever
the codebase already calls it rather than introducing a second handle type.

- [ ] **Step 5: Apply the modes at the commit seam**

In `app/sketch_placement_fields.go`, extend `commitRecipe`:

```go
func (s *Session) commitRecipe(r sketch.Recipe) error {
	if s.activeSketch == nil {
		return errors.New("sketch: no active sketch")
	}
	ents, _, err := s.activeSketch.ApplyWithFields(r, s.lockedFieldExpressions(r), s.overConstrainedBehavior())
	s.applyFormatModes(ents) // Format-panel creation modes (#2015)
	s.applyDrivenDimensionMode()
	s.placementFields = placementFieldState{}
	return err
}
```

and add, in `app/sketch_format_modes.go`:

```go
// applyDrivenDimensionMode makes dimensions created by this commit driven when the mode is armed.
// It runs after the recipe's own over-constrained handling, which may already have demoted a
// redundant dimension — both set the same flag, so the two compose.
func (s *Session) applyDrivenDimensionMode() {
	if !s.formatModes.drivenDim || s.activeSketch == nil {
		return
	}
	for _, d := range s.activeSketch.DimensionConstraints().All() {
		d.SetDriven(true)
	}
}
```

Note this marks every dimension in the sketch, not only the new ones. Narrow it by capturing the
dimension count before the apply and marking only those added after it — write that as:

```go
func (s *Session) applyDrivenDimensionMode(before int) {
	if !s.formatModes.drivenDim || s.activeSketch == nil {
		return
	}
	dims := s.activeSketch.DimensionConstraints().All()
	for i := before; i < len(dims); i++ {
		dims[i].SetDriven(true)
	}
}
```

and take `before := len(s.activeSketch.DimensionConstraints().All())` before `ApplyWithFields`.

- [ ] **Step 6: Point the commands at the new toggles**

Rewrite `sketchFormatCommands` in `app/sketch_format.go` to register all six buttons, each
`WithTab("Sketch")` and, for the four that apply in 3D, a second registration `WithTab(tab3DSketch)`
with `WithEnvironment(Sketch3DEnvironment)` and `WithEnable(inSketch3D)`. Look up the exact tab
constant and predicate names with
`grep -n "tab3DSketch\|inSketch3D\|Sketch3DEnvironment" app/commands_standard.go | head -3`.

- [ ] **Step 7: Run the tests**

Run: `go test ./app/ -run 'TestFormatToggle|Mode|Driven' -v`
Expected: PASS — all five.

- [ ] **Step 8: Run the full app suite**

Run: `go test ./app/`
Expected: `ok` — the existing Format-panel tests still pass. `ToggleConstruction`'s return value
is unchanged in meaning for the selection case.

- [ ] **Step 9: Commit**

```bash
gofmt -w app/
git add -A && git commit -m "feat(sketch): Format toggles convert a selection or arm a creation mode (#2015)"
```

---

### Task 5: Show Format

**Files:**

- Modify: `app/sketch_format_modes.go` (the toggle), `app/options/options.go`,
  `app/session_options.go`, `app/sketch_linestyle.go`
- Test: `app/sketch_show_format_test.go`

**Interfaces:**

- Produces: `(*Session).ToggleShowFormat()`, `(*Session).ShowFormat() bool`, and
  `SketchEntityStyle(sk *sketch.Sketch, e sketch.Entity, suppress bool) EntityStyle` replacing the
  pattern-only `SketchEntityPattern`.

- [ ] **Step 1: Write the failing test**

```go
// app/sketch_show_format_test.go
package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Show Format's documented behaviour is the inverse of its name: ON suppresses the overrides and
// draws with default attributes; OFF shows user formatting again.
func TestShowFormatSuppressesOverrides(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	sk.SetEntityFormat(l.EntityID(), sketch.EntityFormat{
		LineType: "dashed", Color: types.NewColor(255, 0, 0), LineWeight: 0.5,
	})

	styled := SketchEntityStyle(sk, l, false)
	if !styled.Color.IsOverride() || styled.LineWeight != 0.5 {
		t.Errorf("with Show Format off the override must apply, got %+v", styled)
	}

	suppressed := SketchEntityStyle(sk, l, true)
	if suppressed.Color.IsOverride() || suppressed.LineWeight != 0 {
		t.Errorf("with Show Format on the override must be suppressed, got %+v", suppressed)
	}
}

func TestShowFormatTogglePersists(t *testing.T) {
	s, _ := sketchSession(t)
	if s.ShowFormat() {
		t.Fatal("Show Format starts off, so user formatting is visible")
	}
	s.ToggleShowFormat()
	if !s.ShowFormat() {
		t.Error("the toggle must take effect")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/ -run ShowFormat -v`
Expected: FAIL — `undefined: SketchEntityStyle`.

- [ ] **Step 3: Widen the style resolver**

Replace `SketchEntityPattern` in `app/sketch_linestyle.go` with a resolver returning all three
attributes, keeping the existing precedence (centerline → centre pattern, construction → dashed,
else the sketch override) and putting the per-entity override in front of it:

```go
// EntityStyle is how one sketch entity draws: its dash pattern, colour and stroke width.
type EntityStyle struct {
	Pattern    []float64
	Color      types.Color
	LineWeight float64
}

// SketchEntityStyle resolves an entity's draw style. suppress is the Show Format toggle: when
// set, per-entity overrides are ignored and the entity draws with default attributes (#2015).
//
//	style := app.SketchEntityStyle(sk, entity, s.ShowFormat())
func SketchEntityStyle(sk *sketch.Sketch, e sketch.Entity, suppress bool) EntityStyle {
	style := EntityStyle{Pattern: SketchEntityPattern(sk, e)}
	if suppress {
		return style
	}
	f, ok := sk.EntityFormat(e.EntityID())
	if !ok {
		return style
	}
	if f.LineType != "" {
		style.Pattern = linetype.Builtin(types.SketchLineType(f.LineType))
	}
	style.Color, style.LineWeight = f.Color, f.LineWeight
	return style
}
```

Keep `SketchEntityPattern` — it stays the default-attributes half and is already used elsewhere.

- [ ] **Step 4: Add the toggle and persist it**

In `app/sketch_format_modes.go`:

```go
// ShowFormat reports whether formatting overrides are being suppressed. The name follows the
// button's label; the behaviour is the documented one, where ON means "show the DEFAULT format".
func (s *Session) ShowFormat() bool { return s.appOptions.Sketch.SuppressFormatOverrides }

// ToggleShowFormat flips the suppression and persists it.
func (s *Session) ToggleShowFormat() {
	s.appOptions.Sketch.SuppressFormatOverrides = !s.appOptions.Sketch.SuppressFormatOverrides
	_ = s.saveOptions()
}
```

and add to `options.Sketch` in `app/options/options.go`:

```go
	// SuppressFormatOverrides is the Format panel's Show Format toggle (#2015). On means the
	// sketch draws with DEFAULT attributes, hiding per-entity overrides — the documented
	// behaviour, which is the inverse of what the button's label suggests.
	SuppressFormatOverrides bool `yaml:"suppressFormatOverrides"`
```

- [ ] **Step 5: Run the tests**

Run: `go test ./app/ -run ShowFormat -v`
Expected: PASS.

- [ ] **Step 6: Point the head's sketch overlay at the new resolver**

In `head/ui/sketch_overlay.go`, replace the `SketchEntityPattern` call (around line 213) with
`app.SketchEntityStyle(sk, e, s.ShowFormat())` and use the returned colour and weight for the
draw item. Give the overlay a ≤6-method consumer interface rather than passing `*app.Session`, per
the audit I5 ratchet, and add its compile-time assertion.

- [ ] **Step 7: Build and test the head**

Run: `cd head && go build ./... && go test ./ui/ && cd ..`
Expected: builds and passes.

- [ ] **Step 8: Commit**

```bash
gofmt -w app/ head/ui/
git add -A && git commit -m "feat(sketch): Show Format suppresses per-entity overrides (#2015)"
```

---

### Task 6: The three lists — reading and writing format

**Files:**

- Create: `app/sketch_format_style.go`
- Test: `app/sketch_format_style_test.go`

**Interfaces:**

- Produces: `(*Session).SetSelectionLineType(name string) int`,
  `SetSelectionColor(c types.Color) int`, `SetSelectionLineWeight(w float64) int`,
  `SelectionFormat() sketch.EntityFormat` (the common format of the selection, or the armed
  creation format when nothing is selected).

- [ ] **Step 1: Write the failing test**

```go
// app/sketch_format_style_test.go
package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// The lists set the format of the selected geometry, and report what the selection currently has.
func TestSetSelectionFormat(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})

	if n := s.SetSelectionLineType("dashed"); n != 1 {
		t.Fatalf("styled = %d, want 1", n)
	}
	f, ok := sk.EntityFormat(l.EntityID())
	if !ok || f.LineType != "dashed" {
		t.Errorf("format = %+v ok=%v, want a dashed line type", f, ok)
	}

	s.SetSelectionColor(types.NewColor(255, 0, 0))
	s.SetSelectionLineWeight(0.35)
	f, _ = sk.EntityFormat(l.EntityID())
	if !f.Color.IsOverride() || f.LineWeight != 0.35 {
		t.Errorf("format = %+v, want colour and weight overridden alongside the line type", f)
	}
}

// Setting a field back to Default clears just that field, leaving the others.
func TestSetSelectionFormatToDefault(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})
	s.SetSelectionLineType("dashed")
	s.SetSelectionLineWeight(0.35)

	s.SetSelectionLineType("") // back to Default
	f, ok := sk.EntityFormat(l.EntityID())
	if !ok {
		t.Fatal("the entity still overrides its weight, so an entry must remain")
	}
	if f.LineType != "" || f.LineWeight != 0.35 {
		t.Errorf("format = %+v, want the line type cleared and the weight kept", f)
	}
}

// Clearing the last override removes the entry entirely — absence is the single representation
// of Default.
func TestClearingEveryFieldRemovesTheEntry(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})
	s.SetSelectionLineType("dashed")
	s.SetSelectionLineType("")
	if n := sk.EntityFormatCount(); n != 0 {
		t.Errorf("format entries = %d, want 0", n)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/ -run SelectionFormat -v` and `go test ./app/ -run SetSelection -v`
Expected: FAIL — `s.SetSelectionLineType undefined`.

- [ ] **Step 3: Write `app/sketch_format_style.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/sketch"
)

// The Format panel's three selection lists (#2015): line type, colour and thickness, applied to
// the selected geometry. Each list's "Default" entry clears just that field, so an entity can
// override its colour while inheriting its line type; clearing the last override removes the
// entry entirely, keeping absence as the single representation of Default.
//
// These are the values a DWG import carries in from the file's layer table, which is why they
// exist at all.

// SetSelectionLineType sets (or with "" clears) the line type of every selected entity, returning
// how many it changed.
func (s *Session) SetSelectionLineType(name string) int {
	return s.editSelectionFormat(func(f *sketch.EntityFormat) { f.LineType = name })
}

// SetSelectionColor sets the colour of every selected entity; pass a Color whose Source is
// types.AutomaticColorSource to clear the override.
func (s *Session) SetSelectionColor(c types.Color) int {
	return s.editSelectionFormat(func(f *sketch.EntityFormat) { f.Color = c })
}

// SetSelectionLineWeight sets (or with 0 clears) the stroke width of every selected entity.
func (s *Session) SetSelectionLineWeight(w float64) int {
	return s.editSelectionFormat(func(f *sketch.EntityFormat) { f.LineWeight = w })
}

// editSelectionFormat applies edit to each selected entity's format, reading the current value so
// the three lists compose rather than overwrite one another.
func (s *Session) editSelectionFormat(edit func(*sketch.EntityFormat)) int {
	sk := s.ActiveSketch()
	if sk == nil {
		return 0
	}
	n := 0
	for _, e := range s.selectedSketchEntities() {
		f, _ := sk.EntityFormat(e.EntityID())
		edit(&f)
		sk.SetEntityFormat(e.EntityID(), f)
		n++
	}
	return n
}

// SelectionFormat is what the three lists display: the format shared by the whole selection.
// Fields that differ across the selection read as Default, so a list never claims a value the
// selection does not uniformly have.
func (s *Session) SelectionFormat() sketch.EntityFormat {
	sk := s.ActiveSketch()
	if sk == nil {
		return sketch.EntityFormat{}
	}
	ents := s.selectedSketchEntities()
	if len(ents) == 0 {
		return sketch.EntityFormat{}
	}
	first, _ := sk.EntityFormat(ents[0].EntityID())
	for _, e := range ents[1:] {
		f, _ := sk.EntityFormat(e.EntityID())
		first = commonFormat(first, f)
	}
	return first
}

// commonFormat keeps only the fields two formats agree on, so a mixed selection reads as Default
// in the differing lists.
func commonFormat(a, b sketch.EntityFormat) sketch.EntityFormat {
	var out sketch.EntityFormat
	if a.LineType == b.LineType {
		out.LineType = a.LineType
	}
	if a.Color == b.Color {
		out.Color = a.Color
	}
	if a.LineWeight == b.LineWeight {
		out.LineWeight = a.LineWeight
	}
	return out
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./app/ -run 'SetSelection|ClearingEveryField' -v`
Expected: PASS — all three.

- [ ] **Step 5: Commit**

```bash
gofmt -w app/
git add -A && git commit -m "feat(sketch): Format panel line type / colour / thickness lists (#2015)"
```

---

### Task 7: Persistence

**Files:**

- Modify: `persistence/` (sketch entity records), `yamlcodec/yamlcodec.go`
- Test: `persistence/sketch_format_test.go`

**Interfaces:**

- Consumes: `(*Sketch).EntityFormat/SetEntityFormat`, `(*Point).IsCenterPoint/SetCenterPoint`.

- [ ] **Step 1: Write the failing round-trip test**

```go
// persistence/sketch_format_test.go — adapt the imports and the save/load helpers to whatever
// the package's existing sketch round-trip tests use (see the sibling *_test.go files).
//
// The test: build a sketch with one formatted line and one centre point, save, load, and assert
// both survived — the format map by value, the centre-point flag by its predicate.
```

Write it concretely by copying the structure of the nearest existing round-trip test; find one
with `ls persistence/*_test.go | head` and `grep -l "round" persistence/*_test.go`.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./persistence/ -run SketchFormat -v`
Expected: FAIL — the format and the flag are dropped.

- [ ] **Step 3: Add the fields to the persisted records**

Extend the sketch entity record with `lineType`, `color`, `lineWeight` and `centerPoint`, all
`omitempty` so existing documents are unchanged and unstyled entities add no bytes.

- [ ] **Step 4: Run the tests**

Run: `go test ./persistence/ ./yamlcodec/`
Expected: `ok`.

- [ ] **Step 5: Confirm existing documents still load**

Run: `go test ./...`
Expected: `ok` — the `.obk` fixtures in the repo load unchanged, since every new field is
optional.

- [ ] **Step 6: Commit**

```bash
gofmt -w persistence/ yamlcodec/
git add -A && git commit -m "feat(persistence): round-trip sketch formats and centre points (#2015)"
```

---

### Task 8: DWG/DXF round-trip

**Files:**

- Modify: the DXF and DWG importers and exporters under `kernel/exchange/`
- Test: `kernel/exchange/dxf/format_roundtrip_test.go`

- [ ] **Step 1: Write the failing test**

A DXF containing one entity on a layer with a non-default colour, line type and line weight must
import with those values as explicit per-entity overrides, and re-export with the same appearance.
Build the fixture inline in the test rather than adding a binary file.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./kernel/exchange/dxf/ -run FormatRoundTrip -v`
Expected: FAIL — the values are dropped.

- [ ] **Step 3: Resolve BYLAYER on import**

Where the importer creates each sketch entity, look the entity's layer up in the file's layer
table, resolve BYLAYER/BYBLOCK to the concrete values, and call `SetEntityFormat`.

- [ ] **Step 4: Write the overrides on export**

Where the exporter writes each entity, emit its stored colour (group 62), line type (group 6) and
line weight (group 370). Record in the exporter's doc comment that layer structure is not
preserved — entities carry explicit values rather than BYLAYER — so the loss is discoverable at
the code that causes it.

- [ ] **Step 5: Run the tests**

Run: `go test ./kernel/exchange/...`
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
gofmt -w kernel/exchange/
git add -A && git commit -m "feat(exchange): carry entity format through DWG/DXF round-trip (#2015)"
```

---

### Task 9: The SelectionListButton ribbon control

**Files:**

- Create: `head/ui/ribbon_selection_list.go`
- Modify: `../Oblikovati.API/types/button_style.go`, `app/command.go`,
  `head/ui/chrome_ribbon.go`
- Test: `head/ui/ribbon_selection_list_test.go`

**Interfaces:**

- Produces: `types.SelectionListButton ButtonStyle = 4`;
  `(*CommandDefinition).WithSelectionList(items func() []ListItem, current func() int, choose func(int))`;
  `ListItem{Label string; Preview ListPreview}` with `ListPreview` one of pattern / colour / weight.

- [ ] **Step 1: Add the button style and the command builder**

In `../Oblikovati.API/types/button_style.go`:

```go
	// SelectionListButton renders as a dropdown showing the current value with a preview
	// (a dash pattern, a colour swatch, a stroke sample) — the sketch Format panel's line
	// type, colour and thickness lists (Oblikovati/Oblikovati#2015).
	SelectionListButton ButtonStyle = 4
```

In `app/command.go`, add the builder and its accessors mirroring `WithVariants`.

- [ ] **Step 2: Write the failing render test**

```go
//go:build cgo

// head/ui/ribbon_selection_list_test.go
package ui

import "testing"

// The preview strip a list entry draws is chosen by its kind, so a line-type entry draws a dash
// pattern and a colour entry a swatch — the whole point of the control being a list rather than
// a menu.
func TestSelectionListPreviewKinds(t *testing.T) {
	if got := previewKindOf(ListItem{Preview: PatternPreview{Pattern: []float64{1, 1}}}); got != "pattern" {
		t.Errorf("preview kind = %q, want pattern", got)
	}
	if got := previewKindOf(ListItem{Preview: ColorPreview{}}); got != "color" {
		t.Errorf("preview kind = %q, want color", got)
	}
	if got := previewKindOf(ListItem{Preview: WeightPreview{Weight: 0.5}}); got != "weight" {
		t.Errorf("preview kind = %q, want weight", got)
	}
}
```

- [ ] **Step 3: Run it to verify it fails**

Run: `cd head && go test ./ui/ -run SelectionListPreview -v`
Expected: FAIL — `undefined: previewKindOf`.

- [ ] **Step 4: Implement the control**

Draw a combo whose closed state shows the preview plus the current label, and whose open state
lists each entry with its own preview. Reuse `native.DrawLine` with the pattern for a dash sample,
`native.DrawRectFilled` for a colour swatch, and `native.DrawLine` with the weight for a stroke
sample. Model it on the existing variant flyout in `head/ui/chrome_ribbon.go:467` for the open/close
mechanics.

- [ ] **Step 5: Wire the three lists into the Format panel**

Register three `SelectionListButton` commands whose `items` come from `linetype` (the built-in
patterns plus Default), a small colour palette plus Default, and the standard line weights plus
Default; `current` reads `SelectionFormat()`; `choose` calls the matching `SetSelection*`.

- [ ] **Step 6: Run the tests and build**

Run: `cd head && go build ./... && go test ./ui/ && cd ..`
Expected: builds and passes.

- [ ] **Step 7: Check the coupling ratchet**

Run: `go test ./archguard/ -run TestHeadSessionCouplingRatchet`
Expected: PASS. If it rose, give the new widget a ≤6-method consumer interface.

- [ ] **Step 8: Commit**

```bash
cd ../Oblikovati.API && gofmt -w types/ && GOWORK=off go test ./... && git add -A && git commit -m "feat(types): SelectionListButton ribbon style (#2015)" && cd ../Oblikovati
gofmt -w app/ && cd head && gofmt -w ui/ && cd ..
git add -A && git commit -m "feat(head): selection-list ribbon control for the Format panel (#2015)"
```

---

### Task 10: API surface

**Files:**

- Create: `../Oblikovati.API/types/sketch_entity_format.go`
- Modify: `../Oblikovati.API/wire/`, `../Oblikovati.API/client/`, `addin/router/`
- Test: `../Oblikovati.API/types/sketch_entity_format_test.go`, `addin/router/sketch_format_test.go`

- [ ] **Step 1: Add the type**

```go
// SPDX-License-Identifier: Apache-2.0

package types

// SketchEntityFormat is one sketch entity's format overrides — the line type, colour and stroke
// width the Format panel's three lists set, and the values a DWG import carries in from the
// file's layer table (Oblikovati/Oblikovati#2015).
//
// Each field independently means "inherit" when unset, so an entity can override its colour while
// taking the sketch's line type.
type SketchEntityFormat struct {
	// LineType is the pattern name; the empty value inherits the sketch's line type.
	LineType SketchLineType `json:"lineType,omitempty"`
	// OverrideColor is the entity's colour; a colour whose Source is AutomaticColorSource
	// inherits instead.
	OverrideColor Color `json:"overrideColor"`
	// LineWeight is the stroke width in millimetres; 0 inherits.
	LineWeight float64 `json:"lineWeight,omitempty"`
}
```

- [ ] **Step 2: Add wire methods and the client group**

`sketch.getEntityFormat` / `sketch.setEntityFormat`, plus get/set for the four creation modes and
Show Format, following the `sketch_inference.go` pattern exactly in `wire`, `client` and
`addin/router`.

- [ ] **Step 3: Run both modules' suites**

Run: `cd ../Oblikovati.API && GOWORK=off go test ./... && cd ../Oblikovati && go test ./...`
Expected: `ok` in both.

- [ ] **Step 4: Commit each repository separately**

```bash
cd ../Oblikovati.API && git add -A && git commit -m "feat(types): sketch entity format contract (#2015)" && cd ../Oblikovati
git add -A && git commit -m "feat(router): serve sketch entity format and Format-panel modes (#2015)"
```

---

### Task 11: Verification

- [ ] **Step 1: Full suites**

Run: `go test ./... && cd head && go test ./... && cd ..`
Expected: `ok` throughout.

- [ ] **Step 2: Lint**

Run: `make lint && make docs-lint`
Expected: clean. `funlen` is 20 — split anything longer.

- [ ] **Step 3: Coverage**

Run: `go test ./model/sketch/ ./app/ -coverprofile=/tmp/cover.out && go tool cover -func=/tmp/cover.out | tail -1`
Expected: above 80%; no new function at 0%.

- [ ] **Step 4: Live verification**

Drive the head over the MCP bridge: draw geometry with construction mode armed and confirm it
comes out construction; set a line type, colour and thickness on a selection and screenshot;
toggle Show Format and screenshot again to confirm the overrides disappear; place a centre point
and confirm its distinct glyph.

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "test(sketch): Format panel verification sweep (#2015)"
```

---

## Self-Review

**Spec coverage.** Format model → Task 1. Pruning → Task 1 Step 5, mutation-proofed at Step 7.
Copy propagation → Task 2. Centre point → Task 3. The four toggles and creation modes → Task 4.
Show Format including its inverted polarity → Task 5. The three lists → Task 6. Persistence →
Task 7. DWG/DXF → Task 8. Ribbon control → Task 9. API → Task 10. The 2D/3D panel split → Task 4
Step 6.

**Known soft spots**, each flagged at the step that resolves it rather than left as a silent
assumption: the dimension-selection handle name (Task 4 Step 4) and the 3D tab constants (Task 4
Step 6) are looked up rather than guessed, because the survey did not confirm them; Tasks 7 and 8
describe their tests structurally rather than giving literal code, because the persistence and DXF
fixture helpers must be copied from the neighbouring tests in those packages to match their
conventions.

**Type consistency.** `EntityFormat` is the model type throughout; `types.SketchEntityFormat` is
its wire twin in Task 10. `SketchEntityStyle` (Task 5) supersedes `SketchEntityPattern` for
drawing while keeping the old function as its default-attributes half. `applyDrivenDimensionMode`
takes the pre-apply dimension count, fixed in Task 4 Step 5.
