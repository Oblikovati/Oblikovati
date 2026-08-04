# Sketch Entity Creation Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make interactive sketch-entity creation match reference MCAD behaviour — drag-create,
live preview with construction geometry, auto-constraints, and in-place dimension input that
creates real dimension constraints (issue #2014).

**Architecture:** One `Recipe` type in `model/sketch` declares what each shape *is* — geometry,
construction flags, constraints, and dimensionable fields. Preview renders the recipe, `Commit`
applies the same recipe, and the `AddConstrained*` constructors wrap it, so preview cannot drift
from result and the three current duplicate definitions of a polygon collapse into one.

**Tech Stack:** Go 1.x (headless core), cgo + Vulkan + Dear ImGui (`head/`), the sibling
Apache-2.0 module `oblikovati.org/api` resolved through `go.work`.

Spec: `docs/superpowers/specs/2026-08-04-sketch-entity-creation-parity-design.md`

## Global Constraints

- Functions 4–20 lines; files under 500 lines; max 2 levels of indentation; early returns.
- Types explicit — no `any`, no untyped functions.
- Every new exported `.go` file carries `SPDX-License-Identifier: GPL-2.0-only` (this module) or
  `Apache-2.0` (in `../Oblikovati.API`); run `scripts/add-spdx-headers.py`.
- Never re-declare a DTO or method-name string outside `api/wire` — import it.
- Public API additions land contract-first in `../Oblikovati.API`, then implementation here.
- The parameter engine is **unit-strict**: a locked 10 mm must be emitted as `"10 mm"`. A bare
  `"10"` silently means 10 cm (the kernel length unit).
- Constraint correctness is gated on **exact DOF *and* `Redundant == 0`**. A DOF-only check passes
  on the degenerate configurations a duplicated constraint produces.
- Raw `Add*` primitives in `model/sketch` stay behaviourally unchanged — importers, pattern copies
  and procedural add-ins depend on the unconstrained path.
- Tests run with `go test ./...`; lint with `make lint`; docs with `make docs-lint`.
- Coverage > 80%, duplication < 3% before any PR.
- Do not open a PR. The user opens PRs explicitly.

---

## File Structure

| File | Responsibility |
|---|---|
| `model/sketch/recipe.go` (create) | `Recipe`, `RecipeEntity`, `RecipeConstraint`, `RecipeField`, `RecipeDim` types |
| `model/sketch/recipe_apply.go` (create) | `(*Sketch).Apply` — materialise a recipe atomically, honour over-constrained behaviour |
| `model/sketch/recipe_rectangle.go` (create) | `RectangleRecipe`, `ThreePointRectangleRecipe`, `CenterRectangleRecipe` |
| `model/sketch/recipe_slot.go` (create) | `StraightSlotRecipe`, `ArcSlotRecipe` |
| `model/sketch/recipe_curve.go` (create) | `LineRecipe`, `CircleRecipe`, `ArcRecipe`, `EllipseRecipe`, `PolygonRecipe`, `SplineRecipe`, `PointRecipe` |
| `model/sketch/composite_constrained.go` (create) | `AddConstrainedRectangle` and siblings — thin `Apply(XRecipe(…))` wrappers |
| `app/sketch_placement.go` (create) | press/drag/release placement state machine |
| `app/sketch_placement_fields.go` (create) | variable-length field list, typing/lock, expression emission |
| `app/sketch_preview.go` (rewrite) | recipe-backed preview replacing `PreviewPolyline` |
| `head/ui/sketch_placement_overlay.go` (create) | solid/dashed/dotted painting, glyphs, field boxes, padlock |
| `../Oblikovati.API/types/hud_options.go` (create) | `HeadsUpDisplayOptions` |

---

### Task 1: Recipe types and atomic Apply

**Files:**

- Create: `model/sketch/recipe.go`
- Create: `model/sketch/recipe_apply.go`
- Test: `model/sketch/recipe_apply_test.go`

**Interfaces:**

- Consumes: existing `(*GeometricConstraints).Add*` factories, `(*DimensionConstraints).Add*`,
  `(*Sketch).DeleteEntities`, `(*Sketch).AnalyzeConstraints`.
- Produces: `Recipe`, `RecipeEntity`, `RecipeConstraint`, `RecipeField`, `RecipeDim`,
  `RecipeEntityKind` constants, and
  `func (s *Sketch) Apply(r Recipe, behavior types.OverConstrainedDimensionBehavior) ([]Entity, []*Point, error)`.

- [ ] **Step 1: Write the failing test**

```go
// model/sketch/recipe_apply_test.go
package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// applyRecipe is the shared harness: apply r to a fresh sketch and report the DOF analysis.
func applyRecipe(t *testing.T, r Recipe) (*Sketch, []Entity) {
	t.Helper()
	s := New()
	ents, _, err := s.Apply(r, types.OverConstrainedApplyDriven)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return s, ents
}

func TestApplyCreatesEntitiesWithSharedPoints(t *testing.T) {
	r := Recipe{
		Points: []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 8)},
		Entities: []RecipeEntity{
			{Kind: RecipeLine, Points: []int{0, 1}},
			{Kind: RecipeLine, Points: []int{1, 2}},
		},
		Constraints: []RecipeConstraint{{Kind: SingleLineHorizontalKind, Entities: []int{0}}},
	}
	s, ents := applyRecipe(t, r)
	if len(ents) != 2 {
		t.Fatalf("entities = %d, want 2", len(ents))
	}
	l0, l1 := ents[0].(*Line), ents[1].(*Line)
	if l0.B != l1.A {
		t.Error("consecutive lines must share the middle point (structural coincidence)")
	}
	if a := s.AnalyzeConstraints(); a.Equations != 1 || a.Redundant != 0 {
		t.Errorf("eqs=%d redundant=%d, want 1 and 0", a.Equations, a.Redundant)
	}
}

func TestApplyMarksConstructionEntities(t *testing.T) {
	r := Recipe{
		Points:   []math.Point2{math.P2(0, 0), math.P2(10, 0)},
		Entities: []RecipeEntity{{Kind: RecipeLine, Points: []int{0, 1}, Construction: true}},
	}
	_, ents := applyRecipe(t, r)
	if c, ok := ents[0].(interface{ IsConstruction() bool }); !ok || !c.IsConstruction() {
		t.Error("Construction entity must be flagged construction")
	}
}

func TestApplyRollsBackOnBadConstraint(t *testing.T) {
	s := New()
	r := Recipe{
		Points:      []math.Point2{math.P2(0, 0), math.P2(10, 0)},
		Entities:    []RecipeEntity{{Kind: RecipeLine, Points: []int{0, 1}}},
		Constraints: []RecipeConstraint{{Kind: PerpendicularKind, Entities: []int{0, 9}}}, // index out of range
	}
	if _, _, err := s.Apply(r, types.OverConstrainedApplyDriven); err == nil {
		t.Fatal("Apply must reject an out-of-range entity index")
	}
	if n := len(s.Entities()); n != 0 {
		t.Errorf("rollback left %d entities, want 0", n)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./model/sketch/ -run TestApply -v`
Expected: FAIL — `undefined: Recipe`, `undefined: RecipeLine`, `s.Apply undefined`.

- [ ] **Step 3: Write `model/sketch/recipe.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// A Recipe is the single declarative description of what one sketch tool creates: its
// geometry, which of it is construction, the constraints that make it rigid, and the
// quantities the user can dimension while placing it. Preview renders a Recipe, Commit
// applies the same Recipe, and the AddConstrained* constructors wrap it — so the shape has
// exactly one definition and a preview can never disagree with the committed result (#2014).
//
//	r := RectangleRecipe(math.P2(0, 0), math.P2(10, 8), nil)
//	ents, pts, err := sk.Apply(r, types.OverConstrainedApplyDriven)
type Recipe struct {
	// Points are the shape's defining positions; Entities and Constraints refer to them by
	// index, so two entities naming the same index share one Point and are structurally
	// coincident.
	Points []math.Point2
	// Entities are created in order; Constraints and Fields refer to them by index.
	Entities []RecipeEntity
	// Constraints are the geometric relations that make the shape rigid.
	Constraints []RecipeConstraint
	// Fields are the dimensionable quantities shown as in-place input boxes while placing.
	Fields []RecipeField
}

// RecipeEntityKind selects which entity a RecipeEntity creates.
type RecipeEntityKind uint8

const (
	// RecipeLine is a segment between Points[0] and Points[1].
	RecipeLine RecipeEntityKind = iota
	// RecipeArc is an arc about Points[0] from Points[1] to Points[2].
	RecipeArc
	// RecipeCircle is a circle about Points[0] with Radius.
	RecipeCircle
	// RecipeEllipse is an ellipse about Points[0] with MajorAxis, Radius and MinorRadius.
	RecipeEllipse
	// RecipeSpline interpolates every point in Points.
	RecipeSpline
	// RecipePoint is a standalone sketch point at Points[0].
	RecipePoint
)

// RecipeEntity is one entity to create, naming its defining points by index into
// [Recipe.Points].
type RecipeEntity struct {
	Kind         RecipeEntityKind
	Points       []int
	Construction bool
	// CounterClockwise orients a RecipeArc.
	CounterClockwise bool
	// Radius sizes a RecipeCircle or an ellipse's major radius; MinorRadius and MajorAxis
	// complete an ellipse. Unused by the other kinds.
	Radius      math.Scalar
	MinorRadius math.Scalar
	MajorAxis   math.Vector2
}

// RecipeConstraint is one geometric relation, naming its operands by index into
// [Recipe.Entities] and [Recipe.Points].
type RecipeConstraint struct {
	Kind     ConstraintKind
	Entities []int
	Points   []int
}

// FieldUnit selects how a field's value is read and formatted.
type FieldUnit uint8

const (
	// FieldLength is a model-unit length.
	FieldLength FieldUnit = iota
	// FieldAngle is an angle in radians.
	FieldAngle
)

// RecipeField is one in-place input box: a labelled quantity the user can type over while
// placing the shape. A field the user types into is locked and becomes the dimension named
// by Dim; a field left tracking the cursor creates nothing.
type RecipeField struct {
	Label string
	Unit  FieldUnit
	// Value is the live measurement at the current cursor — model units, or radians.
	Value float64
	// Witness spans the two positions the dotted extension line connects; the input box is
	// drawn at its midpoint.
	Witness [2]math.Point2
	// Dim names the dimension to create when this field is locked.
	Dim RecipeDim
}

// RecipeDim names the dimension constraint a locked field creates, by index into the
// recipe's points and entities.
type RecipeDim struct {
	Kind DimKind
	// Points are the two point indices a DistanceDim spans.
	Points [2]int
	// Entity is the entity index a RadiusDim/DiameterDim measures, or the first of two lines
	// an AngleDim spans; -1 when unused.
	Entity int
	// Entity2 is the second line of an AngleDim; -1 when unused.
	Entity2 int
	// Orientation selects what a DistanceDim measures (aligned / horizontal / vertical).
	Orientation DistanceOrientation
}
```

- [ ] **Step 4: Write `model/sketch/recipe_apply.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Apply materialises a recipe into the sketch: it mints the shared points, creates every
// entity in order, applies the geometric constraints, and returns the created entities and
// points. It is atomic — any failure deletes everything it created, so a rejected recipe
// never leaves orphan geometry behind.
//
// Dimensions are NOT created here; ApplyWithFields adds those for locked fields, because
// only the interactive layer knows which fields the user typed into.
//
//	ents, pts, err := sk.Apply(RectangleRecipe(a, c, nil), types.OverConstrainedApplyDriven)
func (s *Sketch) Apply(r Recipe, behavior types.OverConstrainedDimensionBehavior) ([]Entity, []*Point, error) {
	return s.ApplyWithFields(r, nil, behavior)
}

// ApplyWithFields is Apply plus a dimension for each locked field. locked[i] gives field i's
// expression (e.g. "10 mm"); an empty string leaves that field undimensioned. A dimension
// that would introduce redundancy is handled per behavior.
func (s *Sketch) ApplyWithFields(r Recipe, locked []string, behavior types.OverConstrainedDimensionBehavior) ([]Entity, []*Point, error) {
	pts := s.mintRecipePoints(r)
	ents, err := s.buildRecipeEntities(r, pts)
	if err != nil {
		return s.rollbackRecipe(ents, err)
	}
	if err := s.applyRecipeConstraints(r, ents, pts); err != nil {
		return s.rollbackRecipe(ents, err)
	}
	if err := s.applyRecipeFields(r, ents, pts, locked, behavior); err != nil {
		return s.rollbackRecipe(ents, err)
	}
	return ents, pts, nil
}

// rollbackRecipe deletes everything a failed Apply created (DeleteEntities also prunes the
// orphaned points and the constraints bound to them) and returns the error.
func (s *Sketch) rollbackRecipe(ents []Entity, err error) ([]Entity, []*Point, error) {
	s.DeleteEntities(ents)
	return nil, nil, err
}

// mintRecipePoints creates one sketch point per recipe position, in index order.
func (s *Sketch) mintRecipePoints(r Recipe) []*Point {
	pts := make([]*Point, len(r.Points))
	for i, p := range r.Points {
		pts[i] = s.Points().Add(p)
	}
	return pts
}

// buildRecipeEntities creates every entity in order, sharing the minted points so entities
// naming the same index are structurally coincident.
func (s *Sketch) buildRecipeEntities(r Recipe, pts []*Point) ([]Entity, error) {
	ents := make([]Entity, 0, len(r.Entities))
	for i, re := range r.Entities {
		e, err := s.buildRecipeEntity(re, pts)
		if err != nil {
			return ents, fmt.Errorf("recipe entity %d: %w", i, err)
		}
		if re.Construction {
			e.(interface{ SetConstruction(bool) }).SetConstruction(true)
		}
		ents = append(ents, e)
	}
	return ents, nil
}

// buildRecipeEntity creates one entity from its recipe description.
func (s *Sketch) buildRecipeEntity(re RecipeEntity, pts []*Point) (Entity, error) {
	if err := checkPointIndices(re, len(pts)); err != nil {
		return nil, err
	}
	switch re.Kind {
	case RecipeLine:
		return s.Lines().Add(pts[re.Points[0]], pts[re.Points[1]]), nil
	case RecipeArc:
		return s.Arcs().Add(pts[re.Points[0]], pts[re.Points[1]], pts[re.Points[2]], re.CounterClockwise), nil
	case RecipeCircle:
		return s.Circles().Add(pts[re.Points[0]], re.Radius), nil
	case RecipeEllipse:
		return s.Ellipses().AddWithCenter(pts[re.Points[0]], re.MajorAxis, re.Radius, re.MinorRadius), nil
	case RecipePoint:
		return pts[re.Points[0]], nil
	default:
		return nil, fmt.Errorf("unknown recipe entity kind %d (want RecipeLine..RecipePoint)", re.Kind)
	}
}

// recipeArity is how many point indices each entity kind needs.
var recipeArity = map[RecipeEntityKind]int{
	RecipeLine: 2, RecipeArc: 3, RecipeCircle: 1, RecipeEllipse: 1, RecipePoint: 1,
}

// checkPointIndices rejects an entity whose point indices are the wrong count or out of range.
func checkPointIndices(re RecipeEntity, n int) error {
	if want, ok := recipeArity[re.Kind]; ok && len(re.Points) != want {
		return fmt.Errorf("kind %d has %d points, want %d", re.Kind, len(re.Points), want)
	}
	for _, i := range re.Points {
		if i < 0 || i >= n {
			return fmt.Errorf("point index %d out of range [0,%d)", i, n)
		}
	}
	return nil
}
```

Note: `RecipeSpline` is deliberately absent from `buildRecipeEntity` and `recipeArity` in this
task — Task 5 adds it together with the spline recipe, so this task stays independently
testable.

- [ ] **Step 5: Write `applyRecipeConstraints` in the same file**

```go
// applyRecipeConstraints applies every geometric relation, resolving operand indices to the
// created entities and points.
func (s *Sketch) applyRecipeConstraints(r Recipe, ents []Entity, pts []*Point) error {
	g := s.GeometricConstraints()
	for i, rc := range r.Constraints {
		if err := checkOperandIndices(rc, len(ents), len(pts)); err != nil {
			return fmt.Errorf("recipe constraint %d: %w", i, err)
		}
		if err := applyOneRecipeConstraint(g, rc, ents, pts); err != nil {
			return fmt.Errorf("recipe constraint %d: %w", i, err)
		}
	}
	return nil
}

// checkOperandIndices rejects a constraint naming an entity or point outside the recipe.
func checkOperandIndices(rc RecipeConstraint, nEnts, nPts int) error {
	for _, i := range rc.Entities {
		if i < 0 || i >= nEnts {
			return fmt.Errorf("entity index %d out of range [0,%d)", i, nEnts)
		}
	}
	for _, i := range rc.Points {
		if i < 0 || i >= nPts {
			return fmt.Errorf("point index %d out of range [0,%d)", i, nPts)
		}
	}
	return nil
}

// applyOneRecipeConstraint dispatches one relation onto the geometric-constraint factories.
// Only the kinds the shape recipes need are handled; anything else is a programming error.
func applyOneRecipeConstraint(g *GeometricConstraints, rc RecipeConstraint, ents []Entity, pts []*Point) error {
	switch rc.Kind {
	case SingleLineHorizontalKind:
		g.AddLineHorizontal(ents[rc.Entities[0]].(*Line))
	case SingleLineVerticalKind:
		g.AddLineVertical(ents[rc.Entities[0]].(*Line))
	case PerpendicularKind:
		g.AddPerpendicular(ents[rc.Entities[0]].(*Line), ents[rc.Entities[1]].(*Line))
	case MidpointKind:
		g.AddMidpoint(pts[rc.Points[0]], ents[rc.Entities[0]].(*Line))
	case TangentKind:
		g.AddTangent(ents[rc.Entities[0]].(*Line), ents[rc.Entities[1]].(CircularCurve))
	case EqualRadiusKind:
		g.AddEqualRadius(ents[rc.Entities[0]].(CircularCurve), ents[rc.Entities[1]].(CircularCurve))
	case ConcentricKind:
		g.AddConcentric(ents[rc.Entities[0]].(CircularCurve), ents[rc.Entities[1]].(CircularCurve))
	case PointOnCircleKind:
		g.AddPointOnCircle(pts[rc.Points[0]], ents[rc.Entities[0]].(CircularCurve))
	case EqualLengthKind:
		g.AddEqualLength(ents[rc.Entities[0]].(*Line), ents[rc.Entities[1]].(*Line))
	default:
		return fmt.Errorf("unsupported recipe constraint kind %q", rc.Kind)
	}
	return nil
}
```

- [ ] **Step 6: Write `applyRecipeFields` in the same file**

```go
// applyRecipeFields adds a dimension for each locked field. A dimension that raises the
// sketch's redundancy count is handled per behavior: ApplyDriven (the default) demotes it to
// a reference dimension, ApplyDriving keeps it, and Prompt falls back to driven here — the
// interactive layer asks the user before it ever reaches this call.
func (s *Sketch) applyRecipeFields(r Recipe, ents []Entity, pts []*Point, locked []string, behavior types.OverConstrainedDimensionBehavior) error {
	for i, f := range r.Fields {
		if i >= len(locked) || locked[i] == "" {
			continue
		}
		before := s.AnalyzeConstraints().Redundant
		d, err := s.addRecipeDim(f.Dim, ents, pts, locked[i])
		if err != nil {
			return fmt.Errorf("field %q: %w", f.Label, err)
		}
		resolveRedundantDim(s, d, before, behavior)
	}
	return nil
}

// resolveRedundantDim demotes a dimension that introduced redundancy, unless the document
// asked to keep redundant dimensions driving.
func resolveRedundantDim(s *Sketch, d *DimensionConstraint, before int, behavior types.OverConstrainedDimensionBehavior) {
	if behavior == types.OverConstrainedApplyDriving {
		return
	}
	if s.AnalyzeConstraints().Redundant > before {
		d.SetDriven(true)
	}
}

// addRecipeDim creates the one dimension a locked field names.
func (s *Sketch) addRecipeDim(dim RecipeDim, ents []Entity, pts []*Point, expression string) (*DimensionConstraint, error) {
	dc := s.DimensionConstraints()
	switch dim.Kind {
	case DistanceDim:
		return dc.AddDistanceOriented(pts[dim.Points[0]], pts[dim.Points[1]], expression, dim.Orientation)
	case RadiusDim:
		return dc.AddRadius(ents[dim.Entity].(CircularCurve), expression)
	case DiameterDim:
		return dc.AddDiameter(ents[dim.Entity].(CircularCurve), expression)
	case AngleDim:
		return dc.AddAngle(ents[dim.Entity].(*Line), ents[dim.Entity2].(*Line), expression)
	default:
		return nil, fmt.Errorf("unsupported recipe dimension kind %d (want DistanceDim/RadiusDim/DiameterDim/AngleDim)", dim.Kind)
	}
}
```

- [ ] **Step 7: Run the tests**

Run: `go test ./model/sketch/ -run TestApply -v`
Expected: PASS — all three tests.

- [ ] **Step 8: Run the full sketch suite for regressions**

Run: `go test ./model/sketch/`
Expected: `ok oblikovati.org/model/sketch`

- [ ] **Step 9: Commit**

```bash
scripts/add-spdx-headers.py
gofmt -w model/sketch/recipe.go model/sketch/recipe_apply.go model/sketch/recipe_apply_test.go
git add model/sketch/recipe.go model/sketch/recipe_apply.go model/sketch/recipe_apply_test.go
git commit -m "feat(sketch): declarative shape Recipe with atomic Apply (#2014)"
```

---

### Task 2: Rectangle recipes with DOF gates

**Files:**

- Create: `model/sketch/recipe_rectangle.go`
- Test: `model/sketch/recipe_rectangle_test.go`

**Interfaces:**

- Consumes: `Recipe` and `Apply` from Task 1.
- Produces: `RectangleRecipe(a, c math.Point2) Recipe`,
  `ThreePointRectangleRecipe(base0, base1, height math.Point2) Recipe`,
  `CenterRectangleRecipe(center, corner math.Point2) Recipe`.

- [ ] **Step 1: Write the failing DOF tests**

```go
// model/sketch/recipe_rectangle_test.go
package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// assertDOF applies r to a fresh sketch and pins its degrees of freedom and redundancy. DOF
// alone is not enough: a duplicated constraint can report DOF 0 while the solver settles on a
// degenerate configuration, so redundancy is asserted too (#2014).
func assertDOF(t *testing.T, name string, r Recipe, wantDOF int) *Sketch {
	t.Helper()
	s := New()
	if _, _, err := s.Apply(r, types.OverConstrainedApplyDriven); err != nil {
		t.Fatalf("%s: Apply: %v", name, err)
	}
	a := s.AnalyzeConstraints()
	if a.DOF != wantDOF {
		t.Errorf("%s: DOF = %d, want %d (vars=%d eqs=%d rank=%d)", name, a.DOF, wantDOF, a.Variables, a.Equations, a.Rank)
	}
	if a.Redundant != 0 {
		t.Errorf("%s: Redundant = %d, want 0", name, a.Redundant)
	}
	return s
}

func TestRectangleRecipeDOF(t *testing.T) {
	assertDOF(t, "two-point rectangle", RectangleRecipe(math.P2(0, 0), math.P2(10, 8)), 4)
}

func TestThreePointRectangleRecipeDOF(t *testing.T) {
	r := ThreePointRectangleRecipe(math.P2(0, 0), math.P2(10, 0), math.P2(10, 8))
	assertDOF(t, "three-point rectangle", r, 5)
}

func TestCenterRectangleRecipeDOF(t *testing.T) {
	s := assertDOF(t, "centre rectangle", CenterRectangleRecipe(math.P2(0, 0), math.P2(5, 4)), 4)
	construction := 0
	for _, e := range s.Entities() {
		if c, ok := e.(interface{ IsConstruction() bool }); ok && c.IsConstruction() {
			construction++
		}
	}
	if construction != 2 {
		t.Errorf("construction entities = %d, want 2 diagonals", construction)
	}
}

func TestRectangleRecipeFields(t *testing.T) {
	r := RectangleRecipe(math.P2(0, 0), math.P2(10, 8))
	if len(r.Fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(r.Fields))
	}
	if r.Fields[0].Label != "Width" || r.Fields[1].Label != "Height" {
		t.Errorf("labels = %q/%q, want Width/Height", r.Fields[0].Label, r.Fields[1].Label)
	}
	if r.Fields[0].Value != 10 || r.Fields[1].Value != 8 {
		t.Errorf("values = %v/%v, want 10/8", r.Fields[0].Value, r.Fields[1].Value)
	}
	if r.Fields[0].Dim.Orientation != HorizontalDistance {
		t.Error("width must be a horizontal distance dimension")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./model/sketch/ -run 'Rectangle.*Recipe' -v`
Expected: FAIL — `undefined: RectangleRecipe`.

- [ ] **Step 3: Write `model/sketch/recipe_rectangle.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The three rectangle recipes. Each is rigid: a committed rectangle must stay a rectangle
// when a corner is dragged, which is what a bare four-line loop (the pre-#2014 behaviour,
// DOF 8) did not do.

// RectangleRecipe is the axis-aligned two-corner rectangle: four lines over four shared
// corners, held square by a horizontal on each horizontal edge and a vertical on each
// vertical edge. DOF 4 — corner x,y plus width and height.
//
//	r := RectangleRecipe(math.P2(0, 0), math.P2(10, 8))
func RectangleRecipe(a, c math.Point2) Recipe {
	b, d := math.P2(c.X, a.Y), math.P2(a.X, c.Y)
	return Recipe{
		Points:   []math.Point2{a, b, c, d},
		Entities: closedLoopEntities(4),
		Constraints: []RecipeConstraint{
			{Kind: SingleLineHorizontalKind, Entities: []int{0}},
			{Kind: SingleLineHorizontalKind, Entities: []int{2}},
			{Kind: SingleLineVerticalKind, Entities: []int{1}},
			{Kind: SingleLineVerticalKind, Entities: []int{3}},
		},
		Fields: rectangleFields(a, b, c, d),
	}
}

// closedLoopEntities returns n lines joining consecutive point indices in a closed ring.
func closedLoopEntities(n int) []RecipeEntity {
	ents := make([]RecipeEntity, n)
	for i := range ents {
		ents[i] = RecipeEntity{Kind: RecipeLine, Points: []int{i, (i + 1) % n}}
	}
	return ents
}

// rectangleFields is the Width/Height input pair, each witnessed along the edge it measures.
func rectangleFields(a, b, c, d math.Point2) []RecipeField {
	return []RecipeField{
		{
			Label: "Width", Unit: FieldLength, Value: stdmath.Abs(float64(c.X - a.X)),
			Witness: [2]math.Point2{a, b},
			Dim:     RecipeDim{Kind: DistanceDim, Points: [2]int{0, 1}, Entity: -1, Entity2: -1, Orientation: HorizontalDistance},
		},
		{
			Label: "Height", Unit: FieldLength, Value: stdmath.Abs(float64(c.Y - a.Y)),
			Witness: [2]math.Point2{b, c},
			Dim:     RecipeDim{Kind: DistanceDim, Points: [2]int{1, 2}, Entity: -1, Entity2: -1, Orientation: VerticalDistance},
		},
	}
}
```

- [ ] **Step 4: Add the three-point rectangle to the same file**

```go
// ThreePointRectangleRecipe is the rotated rectangle: a base edge from base0 to base1, then a
// height taken as the perpendicular distance of the height point from that base. Three
// perpendicular constraints round the loop keep it square without over-constraining (a fourth
// would be redundant). DOF 5 — corner x,y, base angle, length and width.
func ThreePointRectangleRecipe(base0, base1, height math.Point2) Recipe {
	c2, c3 := thirdRectangleCorners(base0, base1, height)
	return Recipe{
		Points:   []math.Point2{base0, base1, c2, c3},
		Entities: closedLoopEntities(4),
		Constraints: []RecipeConstraint{
			{Kind: PerpendicularKind, Entities: []int{0, 1}},
			{Kind: PerpendicularKind, Entities: []int{1, 2}},
			{Kind: PerpendicularKind, Entities: []int{2, 3}},
		},
		Fields: threePointRectangleFields(base0, base1, c2),
	}
}

// thirdRectangleCorners offsets the base edge by the perpendicular depth of the height point.
func thirdRectangleCorners(base0, base1, height math.Point2) (math.Point2, math.Point2) {
	along := base0.VectorTo(base1)
	perp := math.V2(-along.Y, along.X)
	n := perp.Scale(1 / perp.Length())
	depth := n.Scale(base1.VectorTo(height).Dot(n))
	return base1.TranslateBy(depth), base0.TranslateBy(depth)
}

// threePointRectangleFields is Length/Angle for the base edge plus Width for the offset.
func threePointRectangleFields(base0, base1, c2 math.Point2) []RecipeField {
	along := base0.VectorTo(base1)
	return []RecipeField{
		{
			Label: "Length", Unit: FieldLength, Value: base0.DistanceTo(base1),
			Witness: [2]math.Point2{base0, base1},
			Dim:     RecipeDim{Kind: DistanceDim, Points: [2]int{0, 1}, Entity: -1, Entity2: -1, Orientation: AlignedDistance},
		},
		{
			Label: "Angle", Unit: FieldAngle, Value: stdmath.Atan2(float64(along.Y), float64(along.X)),
			Witness: [2]math.Point2{base0, base1},
			Dim:     RecipeDim{Kind: AngleDim, Points: [2]int{0, 0}, Entity: -1, Entity2: -1},
		},
		{
			Label: "Width", Unit: FieldLength, Value: base1.DistanceTo(c2),
			Witness: [2]math.Point2{base1, c2},
			Dim:     RecipeDim{Kind: DistanceDim, Points: [2]int{1, 2}, Entity: -1, Entity2: -1, Orientation: AlignedDistance},
		},
	}
}
```

The Angle field carries `Entity: -1`, marking it undimensionable for now: an angle dimension
needs a second line to measure against, and a free-standing rectangle has no reference edge.
Task 8 skips locked fields whose `Dim.Entity` is `-1` and `Dim.Kind` is `AngleDim`, so typing an
angle still steers the shape without creating a dimension.

- [ ] **Step 5: Add the centre rectangle to the same file**

```go
// CenterRectangleRecipe is the centre-out rectangle: four corners around a centre point,
// squared by horizontal/vertical constraints, with the centre pinned as the midpoint of one
// construction diagonal. Both diagonals persist as construction geometry — the user drew them
// and they are what anchors the centre. DOF 4 — centre x,y plus width and height.
func CenterRectangleRecipe(center, corner math.Point2) Recipe {
	dx, dy := corner.X-center.X, corner.Y-center.Y
	a := math.P2(center.X-dx, center.Y-dy)
	b := math.P2(center.X+dx, center.Y-dy)
	c := math.P2(center.X+dx, center.Y+dy)
	d := math.P2(center.X-dx, center.Y+dy)
	ents := append(closedLoopEntities(4),
		RecipeEntity{Kind: RecipeLine, Points: []int{0, 2}, Construction: true},
		RecipeEntity{Kind: RecipeLine, Points: []int{1, 3}, Construction: true},
	)
	return Recipe{
		Points:   []math.Point2{a, b, c, d, center},
		Entities: ents,
		Constraints: []RecipeConstraint{
			{Kind: SingleLineHorizontalKind, Entities: []int{0}},
			{Kind: SingleLineHorizontalKind, Entities: []int{2}},
			{Kind: SingleLineVerticalKind, Entities: []int{1}},
			{Kind: SingleLineVerticalKind, Entities: []int{3}},
			{Kind: MidpointKind, Points: []int{4}, Entities: []int{4}},
		},
		Fields: rectangleFields(a, b, c, d),
	}
}
```

- [ ] **Step 6: Run the tests**

Run: `go test ./model/sketch/ -run 'Rectangle.*Recipe' -v`
Expected: PASS — DOF 4, 5, 4 with zero redundancy, two construction diagonals, correct fields.

If a DOF is off, print the analysis the harness already reports (`vars`, `eqs`, `rank`) and
reconcile against the spec's arithmetic before changing the target.

- [ ] **Step 7: Mutation-proof the gate**

Temporarily delete the last `SingleLineVerticalKind` constraint from `RectangleRecipe`, run
`go test ./model/sketch/ -run TestRectangleRecipeDOF`, and confirm it FAILS with `DOF = 5, want 4`.
Restore the constraint and confirm it passes again. A guard test that cannot fail is not a guard.

- [ ] **Step 8: Commit**

```bash
gofmt -w model/sketch/recipe_rectangle.go model/sketch/recipe_rectangle_test.go
git add model/sketch/recipe_rectangle.go model/sketch/recipe_rectangle_test.go
git commit -m "feat(sketch): rigid rectangle recipes, DOF 4/5/4 (#2014)"
```

---

### Task 3: Slot recipes with DOF gates

**Files:**

- Create: `model/sketch/recipe_slot.go`
- Test: `model/sketch/recipe_slot_test.go`

**Interfaces:**

- Consumes: `Recipe`, `Apply`, `assertDOF` (from `recipe_rectangle_test.go`).
- Produces: `StraightSlotRecipe(c0, c1 math.Point2, width math.Scalar) Recipe`,
  `ArcSlotRecipe(center, start, end math.Point2, width math.Scalar, ccw bool) Recipe`.

Target arithmetic from the spec — both must land exactly, with zero redundancy:

- straight slot — 12 vars, 2 circularity (implicit in every arc) + 4 tangency + 1 equal-radius
  = 7 equations, **DOF 5**;
- arc slot — 16 vars, 4 circularity + 4 tangency + 1 equal-radius + 1 concentric = 10 equations,
  **DOF 6**.

The centreline is construction geometry in both: it carries the length dimension. No parallel
constraint is added — parallel sides follow from four tangencies plus equal radii, so stating it
would be redundant and would trip the `Redundant == 0` gate.

- [ ] **Step 1: Write the failing DOF tests**

```go
// model/sketch/recipe_slot_test.go
package sketch

import (
	"testing"

	"oblikovati.org/math"
)

func TestStraightSlotRecipeDOF(t *testing.T) {
	s := assertDOF(t, "straight slot", StraightSlotRecipe(math.P2(0, 0), math.P2(10, 0), 2), 5)
	if got := countConstruction(s); got != 1 {
		t.Errorf("construction entities = %d, want 1 centreline", got)
	}
}

func TestArcSlotRecipeDOF(t *testing.T) {
	r := ArcSlotRecipe(math.P2(0, 0), math.P2(10, 0), math.P2(0, 10), 2, true)
	s := assertDOF(t, "arc slot", r, 6)
	if got := countConstruction(s); got != 1 {
		t.Errorf("construction entities = %d, want 1 centreline arc", got)
	}
}

// countConstruction reports how many of the sketch's entities are construction geometry.
func countConstruction(s *Sketch) int {
	n := 0
	for _, e := range s.Entities() {
		if c, ok := e.(interface{ IsConstruction() bool }); ok && c.IsConstruction() {
			n++
		}
	}
	return n
}

func TestStraightSlotRecipeFields(t *testing.T) {
	r := StraightSlotRecipe(math.P2(0, 0), math.P2(10, 0), 2)
	want := []string{"Length", "Angle", "Width"}
	if len(r.Fields) != len(want) {
		t.Fatalf("fields = %d, want %d", len(r.Fields), len(want))
	}
	for i, label := range want {
		if r.Fields[i].Label != label {
			t.Errorf("field %d = %q, want %q", i, r.Fields[i].Label, label)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./model/sketch/ -run 'Slot.*Recipe' -v`
Expected: FAIL — `undefined: StraightSlotRecipe`.

- [ ] **Step 3: Write `StraightSlotRecipe`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// StraightSlotRecipe is the centre-to-centre straight slot: two parallel sides capped by a
// semicircular arc at each centre, plus a construction centreline that carries the length
// dimension. Four tangencies and one equal-radius make it rigid; parallel sides follow from
// those, so stating parallel as well would be redundant. DOF 5 — centre x,y, angle, length,
// width.
//
//	r := StraightSlotRecipe(math.P2(0, 0), math.P2(10, 0), 2)
func StraightSlotRecipe(c0, c1 math.Point2, width math.Scalar) Recipe {
	d := c0.VectorTo(c1)
	du := d.Scale(1 / d.Length())
	half := math.V2(-du.Y, du.X).Scale(float64(width) / 2)
	a0, a1 := c0.TranslateBy(half), c1.TranslateBy(half)
	b1, b0 := c1.TranslateBy(half.Negate()), c0.TranslateBy(half.Negate())
	// Points: 0=a0 1=a1 2=b1 3=b0 4=c1 5=c0. Entities: 0=side a, 1=cap at c1,
	// 2=side b, 3=cap at c0, 4=construction centreline.
	return Recipe{
		Points: []math.Point2{a0, a1, b1, b0, c1, c0},
		Entities: []RecipeEntity{
			{Kind: RecipeLine, Points: []int{0, 1}},
			{Kind: RecipeArc, Points: []int{4, 1, 2}},
			{Kind: RecipeLine, Points: []int{2, 3}},
			{Kind: RecipeArc, Points: []int{5, 3, 0}},
			{Kind: RecipeLine, Points: []int{5, 4}, Construction: true},
		},
		Constraints: []RecipeConstraint{
			{Kind: TangentKind, Entities: []int{0, 1}},
			{Kind: TangentKind, Entities: []int{2, 1}},
			{Kind: TangentKind, Entities: []int{2, 3}},
			{Kind: TangentKind, Entities: []int{0, 3}},
			{Kind: EqualRadiusKind, Entities: []int{1, 3}},
		},
		Fields: straightSlotFields(c0, c1, a0, b0, width),
	}
}

// straightSlotFields is Length/Angle along the centreline plus the slot Width across it.
func straightSlotFields(c0, c1, a0, b0 math.Point2, width math.Scalar) []RecipeField {
	along := c0.VectorTo(c1)
	return []RecipeField{
		{
			Label: "Length", Unit: FieldLength, Value: c0.DistanceTo(c1),
			Witness: [2]math.Point2{c0, c1},
			Dim:     RecipeDim{Kind: DistanceDim, Points: [2]int{5, 4}, Entity: -1, Entity2: -1, Orientation: AlignedDistance},
		},
		{
			Label: "Angle", Unit: FieldAngle, Value: stdmath.Atan2(float64(along.Y), float64(along.X)),
			Witness: [2]math.Point2{c0, c1},
			Dim:     RecipeDim{Kind: AngleDim, Points: [2]int{0, 0}, Entity: -1, Entity2: -1},
		},
		{
			Label: "Width", Unit: FieldLength, Value: float64(width),
			Witness: [2]math.Point2{a0, b0},
			Dim:     RecipeDim{Kind: DiameterDim, Points: [2]int{0, 0}, Entity: 3, Entity2: -1},
		},
	}
}
```

The Width field dimensions the cap arc's **diameter** — that is the slot width by construction,
and it survives the equal-radius constraint without redundancy (dimensioning the side-to-side
point distance instead would duplicate what tangency already fixes).

- [ ] **Step 4: Add `ArcSlotRecipe` to the same file**

```go
// ArcSlotRecipe is the arc-shaped slot: a centreline arc thickened into concentric inner and
// outer arcs, capped by a semicircle at each end, plus the construction centreline arc that
// carries the radius dimension. DOF 6 — centre x,y, radius, start angle, sweep, width.
func ArcSlotRecipe(center, start, end math.Point2, width math.Scalar, ccw bool) Recipe {
	r := center.DistanceTo(start)
	half := float64(width) / 2
	outS, outE := radialPoint(center, start, r+half), radialPoint(center, end, r+half)
	inS, inE := radialPoint(center, start, r-half), radialPoint(center, end, r-half)
	// Points: 0=centre 1=outS 2=outE 3=inE 4=inS 5=start(cap centre) 6=end(cap centre).
	// Entities: 0=outer arc, 1=cap at end, 2=inner arc, 3=cap at start, 4=construction centreline.
	return Recipe{
		Points: []math.Point2{center, outS, outE, inE, inS, start, end},
		Entities: []RecipeEntity{
			{Kind: RecipeArc, Points: []int{0, 1, 2}, CounterClockwise: ccw},
			{Kind: RecipeArc, Points: []int{6, 2, 3}, CounterClockwise: ccw},
			{Kind: RecipeArc, Points: []int{0, 3, 4}, CounterClockwise: !ccw},
			{Kind: RecipeArc, Points: []int{5, 4, 1}, CounterClockwise: ccw},
			{Kind: RecipeArc, Points: []int{0, 5, 6}, CounterClockwise: ccw, Construction: true},
		},
		Constraints: []RecipeConstraint{
			{Kind: ConcentricKind, Entities: []int{0, 2}},
			{Kind: EqualRadiusKind, Entities: []int{1, 3}},
			{Kind: CircularTangentKind, Entities: []int{0, 1}},
			{Kind: CircularTangentKind, Entities: []int{2, 1}},
			{Kind: CircularTangentKind, Entities: []int{2, 3}},
			{Kind: CircularTangentKind, Entities: []int{0, 3}},
		},
		Fields: arcSlotFields(center, start, end, r, width),
	}
}

// radialPoint returns the point at distance dist from center along the center→through ray.
func radialPoint(center, through math.Point2, dist float64) math.Point2 {
	v := center.VectorTo(through)
	return center.TranslateBy(v.Scale(dist / v.Length()))
}

// arcSlotFields is Radius/Sweep along the centreline arc plus the slot Width across it.
func arcSlotFields(center, start, end math.Point2, r float64, width math.Scalar) []RecipeField {
	vs, ve := center.VectorTo(start), center.VectorTo(end)
	sweep := stdmath.Atan2(float64(ve.Y), float64(ve.X)) - stdmath.Atan2(float64(vs.Y), float64(vs.X))
	return []RecipeField{
		{
			Label: "Radius", Unit: FieldLength, Value: r,
			Witness: [2]math.Point2{center, start},
			Dim:     RecipeDim{Kind: RadiusDim, Points: [2]int{0, 0}, Entity: 4, Entity2: -1},
		},
		{
			Label: "Sweep", Unit: FieldAngle, Value: sweep,
			Witness: [2]math.Point2{start, end},
			Dim:     RecipeDim{Kind: AngleDim, Points: [2]int{0, 0}, Entity: -1, Entity2: -1},
		},
		{
			Label: "Width", Unit: FieldLength, Value: float64(width),
			Witness: [2]math.Point2{start, radialPoint(center, start, r+float64(width)/2)},
			Dim:     RecipeDim{Kind: DiameterDim, Points: [2]int{0, 0}, Entity: 3, Entity2: -1},
		},
	}
}
```

- [ ] **Step 5: Add `CircularTangentKind` to the constraint dispatcher**

In `model/sketch/recipe_apply.go`, extend `applyOneRecipeConstraint`:

```go
	case CircularTangentKind:
		g.AddCircularTangent(ents[rc.Entities[0]].(CircularCurve), ents[rc.Entities[1]].(CircularCurve))
```

- [ ] **Step 6: Run the tests**

Run: `go test ./model/sketch/ -run 'Slot.*Recipe' -v`
Expected: PASS — DOF 5 and 6, zero redundancy, one construction entity each.

If the arc slot lands off 6, the likely cause is the concentric constraint duplicating what the
shared centre point already fixes structurally (both arcs name point index 0). In that case drop
the `ConcentricKind` row and re-run: the target becomes 16 vars − 9 equations = 7, so re-derive
against the spec arithmetic and record the corrected number in both the spec and the test rather
than adjusting the test to whatever the code produces.

- [ ] **Step 7: Mutation-proof the gate**

Delete one `TangentKind` row from `StraightSlotRecipe`, run
`go test ./model/sketch/ -run TestStraightSlotRecipeDOF`, confirm FAIL, then restore.

- [ ] **Step 8: Commit**

```bash
gofmt -w model/sketch/recipe_slot.go model/sketch/recipe_slot_test.go model/sketch/recipe_apply.go
git add model/sketch/recipe_slot.go model/sketch/recipe_slot_test.go model/sketch/recipe_apply.go
git commit -m "feat(sketch): rigid slot recipes with construction centreline (#2014)"
```

---

### Task 4: Curve recipes and the polygon de-duplication

**Files:**

- Create: `model/sketch/recipe_curve.go`
- Modify: `model/sketch/recipe_apply.go` (add `RecipeSpline` to `buildRecipeEntity`/`recipeArity`)
- Test: `model/sketch/recipe_curve_test.go`

**Interfaces:**

- Produces: `LineRecipe(a, b math.Point2) Recipe`,
  `CircleRecipe(center math.Point2, radius math.Scalar) Recipe`,
  `ArcRecipe(center, start, end math.Point2, ccw bool) Recipe`,
  `EllipseRecipe(center math.Point2, majorAxis math.Vector2, majorR, minorR math.Scalar) Recipe`,
  `PolygonRecipe(center, through math.Point2, sides int, inscribed bool) Recipe`,
  `SplineRecipe(pts []math.Point2) Recipe`, `PointRecipe(p math.Point2) Recipe`.

The six already-correct shapes keep their current DOF — this task must not over-constrain them.
`PolygonRecipe` reproduces what `sketch.AddPolygon` already does correctly (construction
circumcircle, `PointOnCircle` per vertex, `EqualLength` per consecutive edge pair), which is the
in-repo precedent for the construction-geometry rule.

- [ ] **Step 1: Write the failing tests**

```go
// model/sketch/recipe_curve_test.go
package sketch

import (
	"testing"

	"oblikovati.org/math"
)

func TestCurveRecipeDOFUnchanged(t *testing.T) {
	cases := []struct {
		name string
		r    Recipe
		dof  int
	}{
		{"line", LineRecipe(math.P2(0, 0), math.P2(10, 0)), 4},
		{"circle", CircleRecipe(math.P2(0, 0), 5), 3},
		{"arc", ArcRecipe(math.P2(0, 0), math.P2(5, 0), math.P2(0, 5), true), 5},
		{"ellipse", EllipseRecipe(math.P2(0, 0), math.V2(1, 0), 5, 3), 5},
		{"point", PointRecipe(math.P2(1, 2)), 2},
	}
	for _, c := range cases {
		assertDOF(t, c.name, c.r, c.dof)
	}
}

func TestPolygonRecipeIsRigid(t *testing.T) {
	s := assertDOF(t, "hexagon", PolygonRecipe(math.P2(0, 0), math.P2(5, 0), 6, true), 4)
	if got := countConstruction(s); got != 1 {
		t.Errorf("construction entities = %d, want 1 circumcircle", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./model/sketch/ -run 'CurveRecipe|PolygonRecipe' -v`
Expected: FAIL — `undefined: LineRecipe`.

- [ ] **Step 3: Write `model/sketch/recipe_curve.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Recipes for the shapes that were already well-formed before #2014 (line, circle, arc,
// ellipse, spline, point) plus the polygon, whose correct constrained construction already
// existed in AddPolygon but was bypassed by a duplicate implementation in the interactive
// tool. These recipes add no constraints to the already-correct shapes: their DOF is their
// intrinsic parameter count and over-constraining them would be a regression.

// LineRecipe is a plain two-point segment. Inference (horizontal/vertical/parallel) is applied
// separately by the interactive layer, which knows the neighbouring geometry.
func LineRecipe(a, b math.Point2) Recipe {
	return Recipe{
		Points:   []math.Point2{a, b},
		Entities: []RecipeEntity{{Kind: RecipeLine, Points: []int{0, 1}}},
		Fields:   lineFields(a, b),
	}
}

// lineFields is the Length/Angle pair measured from the line's start.
func lineFields(a, b math.Point2) []RecipeField {
	v := a.VectorTo(b)
	return []RecipeField{
		{
			Label: "Length", Unit: FieldLength, Value: a.DistanceTo(b),
			Witness: [2]math.Point2{a, b},
			Dim:     RecipeDim{Kind: DistanceDim, Points: [2]int{0, 1}, Entity: -1, Entity2: -1, Orientation: AlignedDistance},
		},
		{
			Label: "Angle", Unit: FieldAngle, Value: stdmath.Atan2(float64(v.Y), float64(v.X)),
			Witness: [2]math.Point2{a, b},
			Dim:     RecipeDim{Kind: AngleDim, Points: [2]int{0, 0}, Entity: -1, Entity2: -1},
		},
	}
}

// CircleRecipe is a centre-and-radius circle; its one field dimensions the diameter.
func CircleRecipe(center math.Point2, radius math.Scalar) Recipe {
	rim := math.P2(center.X+radius, center.Y)
	return Recipe{
		Points:   []math.Point2{center},
		Entities: []RecipeEntity{{Kind: RecipeCircle, Points: []int{0}, Radius: radius}},
		Fields: []RecipeField{{
			Label: "Diameter", Unit: FieldLength, Value: 2 * float64(radius),
			Witness: [2]math.Point2{center, rim},
			Dim:     RecipeDim{Kind: DiameterDim, Points: [2]int{0, 0}, Entity: 0, Entity2: -1},
		}},
	}
}

// ArcRecipe is a centre/start/end arc; its fields dimension the radius and the sweep.
func ArcRecipe(center, start, end math.Point2, ccw bool) Recipe {
	return Recipe{
		Points:   []math.Point2{center, start, end},
		Entities: []RecipeEntity{{Kind: RecipeArc, Points: []int{0, 1, 2}, CounterClockwise: ccw}},
		Fields: []RecipeField{
			{
				Label: "Radius", Unit: FieldLength, Value: center.DistanceTo(start),
				Witness: [2]math.Point2{center, start},
				Dim:     RecipeDim{Kind: RadiusDim, Points: [2]int{0, 0}, Entity: 0, Entity2: -1},
			},
			{
				Label: "Sweep", Unit: FieldAngle, Value: arcSweep(center, start, end),
				Witness: [2]math.Point2{start, end},
				Dim:     RecipeDim{Kind: AngleDim, Points: [2]int{0, 0}, Entity: -1, Entity2: -1},
			},
		},
	}
}

// arcSweep is the signed angle from start to end about center.
func arcSweep(center, start, end math.Point2) float64 {
	vs, ve := center.VectorTo(start), center.VectorTo(end)
	return stdmath.Atan2(float64(ve.Y), float64(ve.X)) - stdmath.Atan2(float64(vs.Y), float64(vs.X))
}

// EllipseRecipe is a centre/axis/two-radii ellipse.
func EllipseRecipe(center math.Point2, majorAxis math.Vector2, majorR, minorR math.Scalar) Recipe {
	return Recipe{
		Points: []math.Point2{center},
		Entities: []RecipeEntity{{
			Kind: RecipeEllipse, Points: []int{0},
			MajorAxis: majorAxis, Radius: majorR, MinorRadius: minorR,
		}},
		Fields: []RecipeField{
			{
				Label: "Major", Unit: FieldLength, Value: float64(majorR),
				Witness: [2]math.Point2{center, center.TranslateBy(majorAxis.Scale(float64(majorR)))},
				Dim:     RecipeDim{Kind: EllipseRadiusDim, Points: [2]int{0, 0}, Entity: 0, Entity2: -1},
			},
			{
				Label: "Minor", Unit: FieldLength, Value: float64(minorR),
				Witness: [2]math.Point2{center, center.TranslateBy(math.V2(-majorAxis.Y, majorAxis.X).Scale(float64(minorR)))},
				Dim:     RecipeDim{Kind: EllipseRadiusDim, Points: [2]int{0, 0}, Entity: -1, Entity2: -1},
			},
		},
	}
}

// SplineRecipe interpolates the given fit points; each fit point stays free (DOF 2n), which is
// the shape's intrinsic parameterisation.
func SplineRecipe(pts []math.Point2) Recipe {
	idx := make([]int, len(pts))
	for i := range pts {
		idx[i] = i
	}
	return Recipe{
		Points:   append([]math.Point2(nil), pts...),
		Entities: []RecipeEntity{{Kind: RecipeSpline, Points: idx}},
	}
}

// PointRecipe is a standalone sketch point.
func PointRecipe(p math.Point2) Recipe {
	return Recipe{
		Points:   []math.Point2{p},
		Entities: []RecipeEntity{{Kind: RecipePoint, Points: []int{0}}},
	}
}
```

- [ ] **Step 4: Add `PolygonRecipe` to the same file**

```go
// PolygonRecipe is the regular n-gon: vertices pinned to a shared construction circumcircle
// with equal consecutive edges, which makes it rigid — equal chords on one circle are equally
// spaced. This mirrors AddPolygon, the implementation the interactive tool used to bypass.
// DOF 4 — centre x,y, circumradius, rotation.
func PolygonRecipe(center, through math.Point2, sides int, inscribed bool) Recipe {
	v := center.VectorTo(through)
	r := v.Length()
	angle := stdmath.Atan2(float64(v.Y), float64(v.X))
	verts := polygonVertices(center, angle, r, sides, inscribed)
	pts := append(append([]math.Point2(nil), verts...), center)
	centerIdx := len(verts)
	ents := append(closedLoopEntities(sides), RecipeEntity{
		Kind: RecipeCircle, Points: []int{centerIdx},
		Radius: math.Scalar(center.DistanceTo(verts[0])), Construction: true,
	})
	return Recipe{
		Points:      pts,
		Entities:    ents,
		Constraints: polygonConstraints(sides, centerIdx),
		Fields:      polygonFields(center, verts[0], r),
	}
}

// polygonConstraints pins every vertex to the circumcircle (entity index sides) and equalises
// consecutive edges.
func polygonConstraints(sides, centerIdx int) []RecipeConstraint {
	cons := make([]RecipeConstraint, 0, 2*sides)
	for i := 0; i < sides; i++ {
		cons = append(cons, RecipeConstraint{Kind: PointOnCircleKind, Points: []int{i}, Entities: []int{sides}})
	}
	for i := 0; i+1 < sides; i++ {
		cons = append(cons, RecipeConstraint{Kind: EqualLengthKind, Entities: []int{i, i + 1}})
	}
	return cons
}

// polygonFields dimensions the circumscribed diameter and the rotation.
func polygonFields(center, vertex math.Point2, r float64) []RecipeField {
	v := center.VectorTo(vertex)
	return []RecipeField{
		{
			Label: "Diameter", Unit: FieldLength, Value: 2 * r,
			Witness: [2]math.Point2{center, vertex},
			Dim:     RecipeDim{Kind: DiameterDim, Points: [2]int{0, 0}, Entity: -1, Entity2: -1},
		},
		{
			Label: "Angle", Unit: FieldAngle, Value: stdmath.Atan2(float64(v.Y), float64(v.X)),
			Witness: [2]math.Point2{center, vertex},
			Dim:     RecipeDim{Kind: AngleDim, Points: [2]int{0, 0}, Entity: -1, Entity2: -1},
		},
	}
}
```

The polygon's Diameter field carries `Entity: -1` because the circumcircle's entity index depends
on `sides`; Task 8 resolves it at lock time from the last entity in the recipe. Set it explicitly
here instead if the index is known — prefer explicit over resolved.

- [ ] **Step 5: Extend `buildRecipeEntity` for splines**

In `model/sketch/recipe_apply.go`, add to `recipeArity` nothing (splines are variadic) and add to
the `switch` in `buildRecipeEntity`:

```go
	case RecipeSpline:
		positions := make([]math.Point2, len(re.Points))
		for i, idx := range re.Points {
			positions[i] = pts[idx].Position()
		}
		return s.Splines().Add(positions), nil
```

Verify the exact `Splines().Add` signature first with
`grep -n "func (c \*Splines) Add" model/sketch/entity_collections.go` and match it; adjust the
call if it differs.

- [ ] **Step 6: Run the tests**

Run: `go test ./model/sketch/ -run 'CurveRecipe|PolygonRecipe' -v`
Expected: PASS — line 4, circle 3, arc 5, ellipse 5, point 2, hexagon 4 with one construction
circle and zero redundancy.

- [ ] **Step 7: Run the whole sketch suite**

Run: `go test ./model/sketch/`
Expected: `ok` — no regression in the existing composite/constraint tests.

- [ ] **Step 8: Commit**

```bash
gofmt -w model/sketch/recipe_curve.go model/sketch/recipe_curve_test.go model/sketch/recipe_apply.go
git add model/sketch/recipe_curve.go model/sketch/recipe_curve_test.go model/sketch/recipe_apply.go
git commit -m "feat(sketch): curve and polygon recipes (#2014)"
```

---

### Task 5: Constrained constructors and tool routing

**Files:**

- Create: `model/sketch/composite_constrained.go`
- Modify: `app/sketch_tools.go` (`RectangleTool.Commit`, `LineTool.Commit`)
- Modify: `app/sketch_geometry_tools.go` (`PolygonTool.Commit` — delete the duplicate)
- Modify: `app/sketch_variant_tools.go` (the three variant `Commit`s)
- Modify: `app/sketch_create_tools.go` (`SketchSlotTool.Commit`, both arc-slot `Commit`s)
- Test: `app/sketch_recipe_commit_test.go`

**Interfaces:**

- Consumes: every `*Recipe` builder from Tasks 2–4.
- Produces: `AddConstrainedRectangle`, `AddConstrainedThreePointRectangle`,
  `AddConstrainedCenterRectangle`, `AddConstrainedStraightSlot`, `AddConstrainedArcSlot`,
  `AddConstrainedPolygon` on `*Sketch`, each returning `([]Entity, error)`.

- [ ] **Step 1: Write the failing test**

```go
// app/sketch_recipe_commit_test.go
package app

import (
	"testing"

	"oblikovati.org/math"
)

// TestToolCommitsAreRigid is the end-to-end gate for #2014: geometry created through the
// interactive tools must come out constrained, not as a floppy loop of free points.
func TestToolCommitsAreRigid(t *testing.T) {
	cases := []struct {
		name string
		dof  int
		commit func(s *Session) error
	}{
		{"rectangle", 4, func(s *Session) error {
			tool := NewRectangleTool()
			tool.corners = []math.Point2{math.P2(0, 0), math.P2(10, 8)}
			return tool.Commit(s)
		}},
		{"centre rectangle", 4, func(s *Session) error {
			tool := NewCenterRectangleTool()
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(5, 4)}
			return tool.Commit(s)
		}},
		{"three-point rectangle", 5, func(s *Session) error {
			tool := NewThreePointRectangleTool()
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 8)}
			return tool.Commit(s)
		}},
		{"polygon", 4, func(s *Session) error {
			tool := NewPolygonTool(6)
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(5, 0)}
			return tool.Commit(s)
		}},
		{"slot", 5, func(s *Session) error {
			tool := NewSketchSlotTool(2)
			tool.points = []math.Point2{math.P2(0, 0), math.P2(10, 0)}
			return tool.Commit(s)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, sk := sketchSession(t)
			if err := c.commit(s); err != nil {
				t.Fatalf("Commit: %v", err)
			}
			a := sk.AnalyzeConstraints()
			if a.DOF != c.dof {
				t.Errorf("DOF = %d, want %d (vars=%d eqs=%d)", a.DOF, c.dof, a.Variables, a.Equations)
			}
			if a.Redundant != 0 {
				t.Errorf("Redundant = %d, want 0", a.Redundant)
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./app/ -run TestToolCommitsAreRigid -v`
Expected: FAIL — rectangle DOF 8 want 4, polygon DOF 12 want 4, slot DOF 10 want 5.

- [ ] **Step 3: Write `model/sketch/composite_constrained.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// The constrained composite constructors: each applies the shape's Recipe, so interactive
// tools, the api/wire composite path and the preview all share one definition of what the
// shape is (#2014). The raw Add* primitives alongside them stay unconstrained on purpose —
// importers, pattern copies and procedural add-ins that state their own constraints depend on
// that path, and auto-applied duplicates there produce a redundant, degenerate solve.
//
//	ents, err := sk.AddConstrainedRectangle(math.P2(0, 0), math.P2(10, 8))

// AddConstrainedRectangle creates the rigid axis-aligned two-corner rectangle (DOF 4).
func (s *Sketch) AddConstrainedRectangle(a, c math.Point2) ([]Entity, error) {
	ents, _, err := s.Apply(RectangleRecipe(a, c), types.OverConstrainedApplyDriven)
	return ents, err
}

// AddConstrainedThreePointRectangle creates the rigid rotated rectangle (DOF 5).
func (s *Sketch) AddConstrainedThreePointRectangle(base0, base1, height math.Point2) ([]Entity, error) {
	ents, _, err := s.Apply(ThreePointRectangleRecipe(base0, base1, height), types.OverConstrainedApplyDriven)
	return ents, err
}

// AddConstrainedCenterRectangle creates the rigid centre-out rectangle with its two
// construction diagonals (DOF 4).
func (s *Sketch) AddConstrainedCenterRectangle(center, corner math.Point2) ([]Entity, error) {
	ents, _, err := s.Apply(CenterRectangleRecipe(center, corner), types.OverConstrainedApplyDriven)
	return ents, err
}

// AddConstrainedStraightSlot creates the rigid centre-to-centre slot with its construction
// centreline (DOF 5).
func (s *Sketch) AddConstrainedStraightSlot(c0, c1 math.Point2, width math.Scalar) ([]Entity, error) {
	ents, _, err := s.Apply(StraightSlotRecipe(c0, c1, width), types.OverConstrainedApplyDriven)
	return ents, err
}

// AddConstrainedArcSlot creates the rigid arc slot with its construction centreline arc (DOF 6).
func (s *Sketch) AddConstrainedArcSlot(center, start, end math.Point2, width math.Scalar, ccw bool) ([]Entity, error) {
	ents, _, err := s.Apply(ArcSlotRecipe(center, start, end, width, ccw), types.OverConstrainedApplyDriven)
	return ents, err
}

// AddConstrainedPolygon creates the rigid regular n-gon with its construction circumcircle
// (DOF 4).
func (s *Sketch) AddConstrainedPolygon(center, through math.Point2, sides int, inscribed bool) ([]Entity, error) {
	ents, _, err := s.Apply(PolygonRecipe(center, through, sides, inscribed), types.OverConstrainedApplyDriven)
	return ents, err
}
```

- [ ] **Step 4: Route `RectangleTool.Commit`**

Replace the body of `RectangleTool.Commit` in `app/sketch_tools.go:70-87` with:

```go
// Commit adds the rigid rectangle: four lines over shared corners, squared by horizontal and
// vertical constraints so dragging a corner cannot shear it (#2014).
func (t *RectangleTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("rectangle")
	}
	_, err := s.activeSketch.AddConstrainedRectangle(t.corners[0], t.corners[1])
	return err
}
```

- [ ] **Step 5: Route the remaining tools**

Apply the same one-line substitution to each `Commit`:

| File | Tool | Call |
|---|---|---|
| `app/sketch_variant_tools.go` | `ThreePointRectangleTool` | `AddConstrainedThreePointRectangle(t.pts[0], t.pts[1], t.pts[2])` |
| `app/sketch_variant_tools.go` | `CenterRectangleTool` | `AddConstrainedCenterRectangle(t.pts[0], t.pts[1])` |
| `app/sketch_geometry_tools.go` | `PolygonTool` | `AddConstrainedPolygon(t.pts[0], t.pts[1], t.Sides, true)` — **delete** the duplicate vertex loop |
| `app/sketch_create_tools.go` | `SketchSlotTool` | `AddConstrainedStraightSlot(t.points[0], t.points[1], t.width)` |
| `app/sketch_create_tools.go` | `CenterPointArcSlotTool` | `AddConstrainedArcSlot(t.pts[0], t.pts[1], t.pts[2], t.width, leftTurn(t.pts[0], t.pts[1], t.pts[2]))` |
| `app/sketch_create_tools.go` | `ThreePointArcSlotTool` | `AddConstrainedArcSlot(center, start, end, t.width, leftTurn(start, through, end))` after the existing `circumcenter` call |

- [ ] **Step 6: Run the tests**

Run: `go test ./app/ -run TestToolCommitsAreRigid -v`
Expected: PASS — all five subtests at their target DOF with zero redundancy.

- [ ] **Step 7: Run the full suite**

Run: `go test ./...`
Expected: `ok` everywhere. Existing tests that asserted the old floppy entity counts may need
their expectations updated — update the assertion only when the new value is provably correct
(a construction centreline legitimately adds one entity to a slot), never to whatever the code
now emits.

- [ ] **Step 8: Commit**

```bash
gofmt -w model/sketch/composite_constrained.go app/sketch_tools.go app/sketch_geometry_tools.go app/sketch_variant_tools.go app/sketch_create_tools.go app/sketch_recipe_commit_test.go
git add -A
git commit -m "feat(sketch): tools commit rigid geometry via constrained constructors (#2014)"
```

---

### Task 6: Recipe-backed preview

**Files:**

- Modify: `app/sketch_preview.go` (rewrite)
- Modify: `head/ui/chrome_viewport.go:699-717` (`toolPreview`)
- Test: `app/sketch_preview_test.go` (extend)

**Interfaces:**

- Consumes: recipe builders from Tasks 2–4.
- Produces: `type RecipeTool interface { PendingRecipe(cursor math.Point2, locked []string) (sketch.Recipe, bool) }`
  and `func (s *Session) ActiveToolRecipe(cursor math.Point2) (sketch.Recipe, bool)`.

Every tool that has a recipe builder implements `PendingRecipe` by calling it with its placed
points plus the cursor. `PreviewPolyline` is kept as a thin adapter derived from the recipe so
the existing `inference_glyphs.go` consumer keeps working unchanged.

- [ ] **Step 1: Write the failing test**

```go
// append to app/sketch_preview_test.go
func TestActiveToolRecipePreviewsRectangle(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	r, ok := s.ActiveToolRecipe(math.P2(10, 8))
	if !ok {
		t.Fatal("a rectangle with one placed corner must preview")
	}
	if len(r.Entities) != 4 {
		t.Errorf("preview entities = %d, want 4", len(r.Entities))
	}
	if len(r.Fields) != 2 || r.Fields[0].Value != 10 {
		t.Errorf("fields = %+v, want Width 10 and Height 8", r.Fields)
	}
}

func TestActiveToolRecipePreviewsConstructionGeometry(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewCenterRectangleTool()
	tool.pts = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	r, ok := s.ActiveToolRecipe(math.P2(5, 4))
	if !ok {
		t.Fatal("a centre rectangle with a placed centre must preview")
	}
	construction := 0
	for _, e := range r.Entities {
		if e.Construction {
			construction++
		}
	}
	if construction != 2 {
		t.Errorf("construction entities in preview = %d, want 2 diagonals", construction)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./app/ -run TestActiveToolRecipe -v`
Expected: FAIL — `s.ActiveToolRecipe undefined`.

- [ ] **Step 3: Rewrite `app/sketch_preview.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Live preview: while a geometry tool has some clicks but not enough to commit, it shows the
// shape it would create at the current cursor — the same sketch.Recipe the commit applies, so
// what is drawn and what is created cannot disagree (#2014). The head maps the recipe's
// sketch-space geometry to model space and paints it: solid for real geometry, dashed for
// construction, dotted witness lines to the in-place dimension boxes.

// RecipeTool is a geometry tool that can describe the shape it would create at the cursor.
// locked carries each field's typed override ("" ⇒ the field tracks the cursor), so a locked
// width freezes that dimension while the drag changes only the rest.
type RecipeTool interface {
	PendingRecipe(cursor math.Point2, locked []string) (sketch.Recipe, bool)
}

// ActiveToolRecipe returns the active tool's provisional shape at the cursor, honouring any
// locked input fields. ok is false when no tool is active or it has too few points yet.
func (s *Session) ActiveToolRecipe(cursor math.Point2) (sketch.Recipe, bool) {
	if s.tool == nil {
		return sketch.Recipe{}, false
	}
	rt, ok := s.tool.tool.(RecipeTool)
	if !ok {
		return sketch.Recipe{}, false
	}
	return rt.PendingRecipe(cursor, s.placementFieldValues())
}

// CursorSketchPoint maps a viewport pixel to a snapped point in the active sketch's plane —
// the point a click would place, used to drive the preview.
func (s *Session) CursorSketchPoint(px, py float64) (math.Point2, bool) {
	return s.sketchClickPoint(px, py)
}

// ActiveToolPreview returns the active tool's provisional outline as a flat polyline. It is
// derived from the recipe so the two can never disagree; the inference-glyph overlay consumes
// it to find the segment being rubber-banded.
func (s *Session) ActiveToolPreview(cursor math.Point2) (pts []math.Point2, closed bool) {
	r, ok := s.ActiveToolRecipe(cursor)
	if !ok {
		return nil, false
	}
	return recipeOutline(r)
}

// recipeOutline flattens a recipe's non-construction line work into one polyline, reporting
// whether it closes back on its first point.
func recipeOutline(r sketch.Recipe) ([]math.Point2, bool) {
	var idx []int
	for _, e := range r.Entities {
		if e.Construction || e.Kind != sketch.RecipeLine {
			continue
		}
		idx = appendOutlineIndices(idx, e.Points)
	}
	pts := make([]math.Point2, len(idx))
	for i, j := range idx {
		pts[i] = r.Points[j]
	}
	return pts, len(idx) > 2 && idx[0] == idx[len(idx)-1]
}

// appendOutlineIndices chains a segment's endpoints onto the running outline, skipping the
// start when it repeats the previous segment's end.
func appendOutlineIndices(idx []int, seg []int) []int {
	if len(idx) > 0 && idx[len(idx)-1] == seg[0] {
		return append(idx, seg[1])
	}
	return append(idx, seg[0], seg[1])
}
```

- [ ] **Step 4: Implement `PendingRecipe` on each tool**

Add one method per tool, in the file where the tool lives. Rectangle, as the pattern:

```go
// PendingRecipe previews the rectangle from its placed corner through the cursor.
func (t *RectangleTool) PendingRecipe(cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.corners) != 1 {
		return sketch.Recipe{}, false
	}
	return sketch.RectangleRecipe(t.corners[0], cursor), true
}
```

The full set, each guarding on its own placed-point count:

| Tool | Guard | Recipe |
|---|---|---|
| `LineTool` | `len(t.points) == 1` | `LineRecipe(t.points[0], cursor)` |
| `RectangleTool` | `len(t.corners) == 1` | `RectangleRecipe(t.corners[0], cursor)` |
| `ThreePointRectangleTool` | `len(t.pts) == 2` | `ThreePointRectangleRecipe(t.pts[0], t.pts[1], cursor)` |
| `CenterRectangleTool` | `len(t.pts) == 1` | `CenterRectangleRecipe(t.pts[0], cursor)` |
| `CircleTool` | `len(t.pts) == 1` | `CircleRecipe(t.pts[0], math.Scalar(t.pts[0].DistanceTo(cursor)))` |
| `ArcTool` | `len(t.pts) == 2` | `ArcRecipe` via the existing `circumcenter(t.pts[0], t.pts[1], cursor)` |
| `CenterPointArcTool` | `len(t.pts) == 2` | `ArcRecipe(t.pts[0], t.pts[1], cursor, leftTurn(t.pts[0], t.pts[1], cursor))` |
| `EllipseTool` | `len(t.pts) == 2` | `EllipseRecipe` from centre, major point, cursor |
| `PolygonTool` | `len(t.pts) == 1` | `PolygonRecipe(t.pts[0], cursor, t.Sides, true)` |
| `SketchSlotTool` | `len(t.points) == 1` | `StraightSlotRecipe(t.points[0], cursor, t.width)` |
| `CenterPointArcSlotTool` | `len(t.pts) == 2` | `ArcSlotRecipe(t.pts[0], t.pts[1], cursor, t.width, leftTurn(...))` |
| `ThreePointArcSlotTool` | `len(t.pts) == 2` | `ArcSlotRecipe` after `circumcenter(t.pts[0], t.pts[1], cursor)` |
| `SplineTool` | `len(t.pts) >= 1` | `SplineRecipe(append(t.pts, cursor))` |
| `ControlVertexSplineTool` | `len(t.pts) >= 1` | `SplineRecipe(append(t.pts, cursor))` |
| `PointTool` | always | `PointRecipe(cursor)` |

Delete the five old `PreviewPolyline` methods — `ActiveToolPreview` now derives the outline.

- [ ] **Step 5: Add a `placementFieldValues` stub**

Task 7 fills this in. For now, in `app/sketch_placement_fields.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only

package app

// placementFieldValues returns each in-place input field's typed override ("" ⇒ the field
// tracks the cursor). Task 8 gives it real state; until then every field tracks the cursor.
func (s *Session) placementFieldValues() []string { return nil }
```

- [ ] **Step 6: Run the tests**

Run: `go test ./app/ -run 'TestActiveToolRecipe|TestToolCommitsAreRigid' -v`
Expected: PASS.

- [ ] **Step 7: Build the head**

Run: `cd head && go build ./... && cd ..`
Expected: builds clean. `toolPreview` and `inference_glyphs.go` still compile because
`ActiveToolPreview` kept its signature.

- [ ] **Step 8: Commit**

```bash
gofmt -w app/sketch_preview.go app/sketch_placement_fields.go app/sketch_tools.go app/sketch_geometry_tools.go app/sketch_variant_tools.go app/sketch_create_tools.go
git add -A
git commit -m "feat(sketch): preview driven by the same recipe the commit applies (#2014)"
```

---

### Task 7: Drag-create placement state machine

**Files:**

- Create: `app/sketch_placement.go`
- Modify: `head/ui/box_select_view.go` (`handleViewportSelection`)
- Test: `app/sketch_placement_test.go`

**Interfaces:**

- Produces: `(*Session).BeginPlacement(px, py float64) bool`,
  `(*Session).UpdatePlacement(px, py float64)`, `(*Session).EndPlacement(px, py float64)`,
  `(*Session).PlacementActive() bool`.

- [ ] **Step 1: Write the failing test**

```go
// app/sketch_placement_test.go
package app

import "testing"

// A press and release without movement is a click: it places one point and waits, exactly as
// the pre-#2014 click-click flow did.
func TestPlacementClickPlacesOnePoint(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewRectangleTool()
	s.StartTool(tool)
	s.BeginPlacement(100, 100)
	s.EndPlacement(100, 100)
	if len(tool.corners) != 1 {
		t.Fatalf("corners = %d, want 1", len(tool.corners))
	}
	if n := len(sk.Entities()); n != 0 {
		t.Errorf("entities = %d, want 0 — a click must not commit", n)
	}
}

// A press, drag past the slop, and release places both points and commits the shape.
func TestPlacementDragPlacesTwoPointsAndCommits(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewRectangleTool())
	s.BeginPlacement(100, 100)
	s.UpdatePlacement(160, 150)
	s.EndPlacement(160, 150)
	if n := len(sk.Entities()); n == 0 {
		t.Fatal("a drag-release must commit the rectangle")
	}
	a := sk.AnalyzeConstraints()
	if a.DOF != 4 || a.Redundant != 0 {
		t.Errorf("DOF = %d redundant = %d, want 4 and 0", a.DOF, a.Redundant)
	}
}

// A drag shorter than the slop is a click, not a drag.
func TestPlacementSlopBoundary(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	s.StartTool(tool)
	s.BeginPlacement(100, 100)
	s.UpdatePlacement(102, 101) // 2.24 px < 4 px slop
	s.EndPlacement(102, 101)
	if len(tool.corners) != 1 {
		t.Errorf("corners = %d, want 1 — a sub-slop drag is a click", len(tool.corners))
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./app/ -run TestPlacement -v`
Expected: FAIL — `s.BeginPlacement undefined`.

- [ ] **Step 3: Write `app/sketch_placement.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package app

// Drag-to-create (#2014): a sketch geometry tool accepts both click-click and
// click-drag-release, and they are the same path — a drag simply means the second point
// arrived on release instead of on the next press. Tools needing three or more points get the
// drag for points 1→2 and click-click for the rest, with no special-casing.
//
// Before this, creation ran off ImGui's IsItemClicked (a mouse *press*) with no release
// handler, so press-drag-release placed one point and the shape appeared on a later,
// unrelated press.

// placementDragSlop is the movement (in viewport pixels) below which a press-release counts as
// a click rather than a drag. It mirrors orbitPivotClickSlop, the same judgement for the
// Free-Orbit set-pivot click.
const placementDragSlop = 4

// sketchPlacement is one in-progress press: where it started, and whether it has moved far
// enough to count as a drag.
type sketchPlacement struct {
	active  bool
	pressX  float64
	pressY  float64
	dragged bool
}

// PlacementActive reports whether a creation press is in progress.
func (s *Session) PlacementActive() bool { return s.placement.active }

// BeginPlacement handles a left press over the sketch plane while a geometry tool is active:
// it places a point and arms the drag. It returns false — so the caller falls through to the
// normal pick path — when no click-consuming tool is active.
func (s *Session) BeginPlacement(px, py float64) bool {
	if s.tool == nil {
		return false
	}
	if _, ok := s.tool.tool.(PlaneClickTool); !ok {
		return false
	}
	s.placement = sketchPlacement{active: true, pressX: px, pressY: py}
	s.sketchClick(px, py)
	return true
}

// UpdatePlacement tracks the cursor while the button is held, promoting the press to a drag
// once it passes the slop.
func (s *Session) UpdatePlacement(px, py float64) {
	if !s.placement.active || s.placement.dragged {
		return
	}
	dx, dy := px-s.placement.pressX, py-s.placement.pressY
	if dx*dx+dy*dy > placementDragSlop*placementDragSlop {
		s.placement.dragged = true
	}
}

// EndPlacement handles the release: a drag places the shape's second point (committing the
// tool if that completes it), while a click leaves the tool waiting for the next press.
func (s *Session) EndPlacement(px, py float64) {
	if !s.placement.active {
		return
	}
	dragged := s.placement.dragged
	s.placement = sketchPlacement{}
	if !dragged {
		return
	}
	s.UpdatePlacement(px, py)
	s.sketchClick(px, py)
}
```

Add the field to `Session` in `app/session.go`, beside `entityDrag`:

```go
	placement sketchPlacement // in-progress drag-to-create press (#2014)
```

- [ ] **Step 4: Run the tests**

Run: `go test ./app/ -run TestPlacement -v`
Expected: PASS — all three.

- [ ] **Step 5: Wire the head**

In `head/ui/box_select_view.go`, add `updateSketchPlacement` and call it from
`handleViewportSelection` immediately before `updateSketchDrag(s)`:

```go
// updateSketchPlacement drives drag-to-create for sketch geometry tools and reports whether it
// consumed this frame's left input: a press places the first point, the cursor rubber-bands,
// and release places the second point and commits (#2014). A press-release without movement
// falls back to the click-click flow.
func updateSketchPlacement(s *app.Session) bool {
	if !s.InSketch() {
		return false
	}
	lx, ly := viewportCursor()
	if s.PlacementActive() {
		if native.MouseDown(native.MouseLeft) {
			s.UpdatePlacement(lx, ly)
		} else {
			s.EndPlacement(lx, ly)
		}
		return true
	}
	if !native.IsItemClicked(native.MouseLeft) {
		return false
	}
	return s.BeginPlacement(lx, ly)
}
```

```go
	if updateSketchPlacement(s) {
		return
	}
	if updateSketchDrag(s) {
		return
	}
```

- [ ] **Step 6: Build and run the head suite**

Run: `cd head && go build ./... && go test ./ui/ && cd ..`
Expected: builds and passes.

- [ ] **Step 7: Verify the click path is unchanged**

Run: `go test ./app/`
Expected: `ok` — the existing click-driven tool tests still pass, since `BeginPlacement` places
the point on press exactly as `sketchClick` did.

- [ ] **Step 8: Commit**

```bash
gofmt -w app/sketch_placement.go app/session.go app/sketch_placement_test.go
git add -A
git commit -m "feat(sketch): drag-to-create placement state machine (#2014)"
```

---

### Task 8: In-place dimension fields

**Files:**

- Modify: `app/sketch_placement_fields.go` (replace the Task 6 stub)
- Modify: `app/sketch_tools.go` (`autoCommitSketchTool` — pass locked values through)
- Test: `app/sketch_placement_fields_test.go`

**Interfaces:**

- Produces: `(*Session).PlacementFields() []PlacementFieldView`,
  `(*Session).PlacementFieldInput(r rune)`, `(*Session).PlacementFieldTab()`,
  `(*Session).PlacementFieldBackspace()`, `(*Session).PlacementFieldCancel()`,
  `(*Session).PlacementFieldCommit(px, py float64) error`, and the view struct
  `PlacementFieldView{Label, Value, Unit string; Active, Locked bool; Witness [2]math.Point2}`.

The contract, from the reference image: **a field the user types into is locked and becomes a
driving dimension; a field left tracking the cursor creates nothing.** A locked field also
freezes that quantity during the drag.

- [ ] **Step 1: Write the failing test**

```go
// app/sketch_placement_fields_test.go
package app

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// Typing a width and Tabbing locks it: the drag then changes only the height.
func TestLockedFieldFreezesTheDrag(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	for _, r := range "10" {
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab()
	r, ok := s.ActiveToolRecipe(math.P2(99, 8))
	if !ok {
		t.Fatal("recipe expected")
	}
	if r.Fields[0].Value != 1 { // 10 mm == 1 cm in model units
		t.Errorf("locked width = %v, want 1 cm (10 mm)", r.Fields[0].Value)
	}
	if r.Fields[1].Value != 8 {
		t.Errorf("height = %v, want 8 — an unlocked field still tracks the cursor", r.Fields[1].Value)
	}
}

// A locked field becomes a driving dimension; an untyped one creates nothing.
func TestLockedFieldCreatesOneDrivingDimension(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	for _, r := range "10" {
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab()
	tool.corners = append(tool.corners, math.P2(1, 0.8))
	if err := s.OK(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	dims := sk.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("dimensions = %d, want 1 (only the locked field)", len(dims))
	}
	if dims[0].Driven() {
		t.Error("a locked field must create a DRIVING dimension")
	}
}

// The parameter engine is unit-strict: the expression must carry its unit or 10 silently means
// 10 cm.
func TestLockedFieldExpressionCarriesItsUnit(t *testing.T) {
	s, _ := sketchSession(t)
	if got := s.placementFieldExpression("10", FieldLengthUnit); !strings.HasSuffix(got, " mm") {
		t.Errorf("expression = %q, want a millimetre unit suffix", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./app/ -run 'TestLockedField' -v`
Expected: FAIL — `s.PlacementFieldInput undefined`.

- [ ] **Step 3: Write `app/sketch_placement_fields.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	stdmath "math"
	"strconv"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// In-place dimension input (#2014): while a shape is being placed, each of its dimensionable
// quantities gets an input box on a dotted witness line. Typing a value and Tabbing locks it —
// the drag then changes only the remaining quantities, and on commit the locked value becomes
// a real driving dimension. A field left tracking the cursor creates nothing.
//
// This is the dimension-input half of the heads-up display; the X/Y pointer-input panel in
// sketch_hud.go is the other half and stays as it is, shown for a shape's first point.

// placementField is one input box's typing state.
type placementField struct {
	typed  string // "" ⇒ the field tracks the cursor
	locked bool   // typed and committed with Tab/Enter ⇒ becomes a dimension
}

// placementFieldState is the whole in-place input strip for the shape being placed.
type placementFieldState struct {
	fields  []placementField
	active  int
	engaged bool
}

// PlacementFieldView is one input box as the head draws it.
type PlacementFieldView struct {
	Label   string
	Value   string
	Unit    string
	Active  bool
	Locked  bool
	Witness [2]math.Point2
}

// FieldLengthUnit and FieldAngleUnit mirror sketch.FieldLength/FieldAngle for callers outside
// the model package.
const (
	FieldLengthUnit = sketch.FieldLength
	FieldAngleUnit  = sketch.FieldAngle
)

// placementFieldValues returns each field's typed override ("" ⇒ tracks the cursor), which the
// recipe builders use to freeze locked quantities during the drag.
func (s *Session) placementFieldValues() []string {
	out := make([]string, len(s.placementFields.fields))
	for i, f := range s.placementFields.fields {
		if f.locked {
			out[i] = f.typed
		}
	}
	return out
}

// PlacementFields returns the input boxes for the shape being placed, sized and labelled from
// the active tool's recipe. It is empty when no shape is in progress.
func (s *Session) PlacementFields() []PlacementFieldView {
	r, ok := s.ActiveToolRecipe(s.lastCursorSketchPoint)
	if !ok {
		return nil
	}
	s.syncPlacementFieldCount(len(r.Fields))
	views := make([]PlacementFieldView, len(r.Fields))
	for i, f := range r.Fields {
		views[i] = s.placementFieldView(i, f)
	}
	return views
}

// syncPlacementFieldCount resizes the typing state when the tool's field count changes (a
// three-point shape shows a different set after its second click).
func (s *Session) syncPlacementFieldCount(n int) {
	if len(s.placementFields.fields) == n {
		return
	}
	s.placementFields.fields = make([]placementField, n)
	s.placementFields.active = 0
}

// placementFieldView renders field i: its typed text when the user has entered one, otherwise
// the live measurement in the document's preferred unit.
func (s *Session) placementFieldView(i int, f sketch.RecipeField) PlacementFieldView {
	st := s.placementFields.fields[i]
	v := PlacementFieldView{
		Label: f.Label, Value: st.typed, Witness: f.Witness,
		Active: i == s.placementFields.active, Locked: st.locked,
		Unit: s.placementFieldUnitName(f.Unit),
	}
	if st.typed == "" {
		v.Value = formatHUDNumber(s.placementFieldLive(f))
	}
	return v
}

// placementFieldLive converts a field's model-unit value into the document's display unit.
func (s *Session) placementFieldLive(f sketch.RecipeField) float64 {
	if f.Unit == sketch.FieldAngle {
		return f.Value * 180 / stdmath.Pi
	}
	return s.DocumentUnits().ToPreferred(param.Q(f.Value, param.Length))
}

// placementFieldUnitName is the suffix shown after a field's value.
func (s *Session) placementFieldUnitName(u sketch.FieldUnit) string {
	if u == sketch.FieldAngle {
		return "deg"
	}
	return s.DocumentUnits().PreferredName(param.Length)
}

// PlacementFieldInput appends a typed character to the active field when it is part of a
// number. Other runes are ignored.
func (s *Session) PlacementFieldInput(r rune) {
	if !isHUDNumberRune(r) || len(s.placementFields.fields) == 0 {
		return
	}
	s.placementFields.engaged = true
	f := &s.placementFields.fields[s.placementFields.active]
	f.typed += string(r)
}

// PlacementFieldTab locks the active field and moves focus to the next — the padlock in the
// reference behaviour. Locking a field freezes that quantity for the rest of the drag.
func (s *Session) PlacementFieldTab() {
	if len(s.placementFields.fields) == 0 {
		return
	}
	s.lockActivePlacementField()
	s.placementFields.active = (s.placementFields.active + 1) % len(s.placementFields.fields)
}

// lockActivePlacementField marks the active field locked when it has a typed value.
func (s *Session) lockActivePlacementField() {
	f := &s.placementFields.fields[s.placementFields.active]
	if f.typed != "" {
		f.locked = true
	}
}

// PlacementFieldBackspace deletes the active field's last character and unlocks it.
func (s *Session) PlacementFieldBackspace() {
	if len(s.placementFields.fields) == 0 {
		return
	}
	f := &s.placementFields.fields[s.placementFields.active]
	if f.typed == "" {
		return
	}
	f.typed, f.locked = f.typed[:len(f.typed)-1], false
}

// PlacementFieldCancel clears all typed state, returning every box to cursor tracking.
func (s *Session) PlacementFieldCancel() { s.placementFields = placementFieldState{} }

// PlacementFieldEngaged reports whether the user has begun typing, so the head can claim plain
// keystrokes before the viewport does.
func (s *Session) PlacementFieldEngaged() bool { return s.placementFields.engaged }

// PlacementFieldCommit locks the active field and commits the shape at the cursor.
func (s *Session) PlacementFieldCommit(px, py float64) error {
	if len(s.placementFields.fields) == 0 {
		return fmt.Errorf("placement fields: no shape is being placed")
	}
	s.lockActivePlacementField()
	if _, ok := s.CursorSketchPoint(px, py); !ok {
		return fmt.Errorf("placement fields: the cursor at (%v,%v) is not over the sketch plane", px, py)
	}
	s.sketchClick(px, py)
	return nil
}

// placementFieldExpression renders a typed value as a parameter expression carrying its unit.
// The parameter engine is unit-strict: a bare "10" means 10 cm (the kernel length unit), so the
// unit is always explicit.
func (s *Session) placementFieldExpression(typed string, u sketch.FieldUnit) string {
	if u == sketch.FieldAngle {
		return typed + " deg"
	}
	return typed + " " + s.DocumentUnits().PreferredName(param.Length)
}

// placementFieldModelValue parses a typed field into model units (or radians), for the recipe
// builders that freeze locked quantities.
func (s *Session) placementFieldModelValue(typed string, u sketch.FieldUnit) (float64, bool) {
	v, err := strconv.ParseFloat(typed, 64)
	if err != nil {
		return 0, false
	}
	if u == sketch.FieldAngle {
		return v * stdmath.Pi / 180, true
	}
	return s.DocumentUnits().FromPreferred(v, param.Length).Value, true
}
```

Add to `Session` in `app/session.go`:

```go
	placementFields       placementFieldState // in-place dimension input (#2014)
	lastCursorSketchPoint math.Point2         // last cursor position mapped to the sketch plane
```

Set `lastCursorSketchPoint` in `CursorSketchPoint` so `PlacementFields` has a cursor to build
from.

- [ ] **Step 4: Honour locked values in the recipe builders**

`PendingRecipe` receives `locked []string`. For the rectangle, the locked width replaces the
cursor-derived corner:

```go
// PendingRecipe previews the rectangle from its placed corner through the cursor, with any
// locked Width/Height overriding what the cursor would give.
func (t *RectangleTool) PendingRecipe(cursor math.Point2, locked []string) (sketch.Recipe, bool) {
	if len(t.corners) != 1 {
		return sketch.Recipe{}, false
	}
	return sketch.RectangleRecipe(t.corners[0], lockedCorner(t.corners[0], cursor, locked)), true
}
```

Add the shared helper to `app/sketch_placement_fields.go`:

```go
// lockedCorner replaces the cursor's X and Y offsets from the anchor with any locked Width and
// Height, keeping the cursor's sign so the rectangle still flips through the anchor.
func (s *Session) lockedCorner(anchor, cursor math.Point2, locked []string) math.Point2 {
	x, y := cursor.X, cursor.Y
	if w, ok := s.lockedLength(locked, 0); ok {
		x = anchor.X + math.Scalar(stdmath.Copysign(w, float64(cursor.X-anchor.X)))
	}
	if h, ok := s.lockedLength(locked, 1); ok {
		y = anchor.Y + math.Scalar(stdmath.Copysign(h, float64(cursor.Y-anchor.Y)))
	}
	return math.P2(x, y)
}

// lockedLength parses locked field i as a model-unit length.
func (s *Session) lockedLength(locked []string, i int) (float64, bool) {
	if i >= len(locked) || locked[i] == "" {
		return 0, false
	}
	return s.placementFieldModelValue(locked[i], sketch.FieldLength)
}
```

`lockedCorner` needs the session, so `PendingRecipe` must take it. Change the `RecipeTool`
interface to `PendingRecipe(s *Session, cursor math.Point2, locked []string) (sketch.Recipe, bool)`
and update `ActiveToolRecipe` to pass `s`. Update every tool's `PendingRecipe` signature from
Task 6 accordingly.

- [ ] **Step 5: Create dimensions on commit**

In each tool's `Commit`, route through `ApplyWithFields` instead of the `AddConstrained*`
wrapper when locked values exist. Add to `app/sketch_placement_fields.go`:

```go
// commitRecipe applies a tool's recipe with a dimension for every locked field, honouring the
// document's over-constrained behaviour.
func (s *Session) commitRecipe(r sketch.Recipe) error {
	exprs := make([]string, len(r.Fields))
	for i, f := range r.Fields {
		if st := s.placementField(i); st.locked {
			exprs[i] = s.placementFieldExpression(st.typed, f.Unit)
		}
	}
	_, _, err := s.activeSketch.ApplyWithFields(r, exprs, s.DocumentSketchSettings().OverConstrainedBehavior)
	s.placementFields = placementFieldState{}
	return err
}

// placementField returns field i's typing state, or a zero field when out of range.
func (s *Session) placementField(i int) placementField {
	if i >= len(s.placementFields.fields) {
		return placementField{}
	}
	return s.placementFields.fields[i]
}
```

Then each `Commit` becomes, for the rectangle:

```go
func (t *RectangleTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("rectangle")
	}
	return s.commitRecipe(sketch.RectangleRecipe(t.corners[0], t.corners[1]))
}
```

Skip fields whose `Dim.Entity` is `-1` and `Dim.Kind` is `AngleDim` — they steer the shape but
have no reference edge to dimension against. Add that guard inside `applyRecipeFields`:

```go
		if f.Dim.Kind == AngleDim && f.Dim.Entity < 0 {
			continue
		}
```

- [ ] **Step 6: Route the keystrokes in the head**

In `head/ui/sketch_hud.go`, extend `routeSketchHUDKeys` to prefer the placement fields when a
shape is being placed, falling back to the X/Y pointer panel otherwise. Tab, Enter, Esc and
Backspace map to `PlacementFieldTab`, `PlacementFieldCommit`, `PlacementFieldCancel`,
`PlacementFieldBackspace`.

- [ ] **Step 7: Run the tests**

Run: `go test ./app/ -run 'TestLockedField|TestPlacement|TestToolCommits' -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
gofmt -w app/ && git add -A
git commit -m "feat(sketch): in-place dimension input creates driving dimensions (#2014)"
```

---

### Task 9: Overlay rendering

**Files:**

- Create: `head/ui/sketch_placement_overlay.go`
- Modify: `head/ui/chrome_viewport.go` (`toolPreview` → recipe-aware)
- Test: `head/ui/sketch_placement_overlay_test.go`

**Interfaces:**

- Consumes: `app.Session.ActiveToolRecipe`, `app.Session.PlacementFields`.
- Produces: `placementOverlayItems(s *app.Session, plane sketch.Plane) []renderer.DrawItem`,
  `drawPlacementFieldBoxes(s *app.Session, bx, by float32)`, `drawPadlock(x, y, h float32)`.

- [ ] **Step 1: Write the failing test**

```go
// head/ui/sketch_placement_overlay_test.go
//go:build cgo

package ui

import "testing"

// The overlay must separate solid geometry from dashed construction, so a centre rectangle's
// diagonals read as construction while its four edges read as real geometry.
func TestPlacementOverlaySplitsConstruction(t *testing.T) {
	solid, construction := splitRecipeGeometry(centerRectangleTestRecipe())
	if len(solid) != 4 {
		t.Errorf("solid entities = %d, want 4 edges", len(solid))
	}
	if len(construction) != 2 {
		t.Errorf("construction entities = %d, want 2 diagonals", len(construction))
	}
}
```

Add `centerRectangleTestRecipe()` to the test file as
`return sketch.CenterRectangleRecipe(math.P2(0, 0), math.P2(5, 4))`.

- [ ] **Step 2: Run to verify it fails**

Run: `cd head && go test ./ui/ -run TestPlacementOverlay -v`
Expected: FAIL — `undefined: splitRecipeGeometry`.

- [ ] **Step 3: Write `head/ui/sketch_placement_overlay.go`**

Paint in three passes so the styles match the committed result: solid `previewColor` for real
geometry, the `types.SketchLineDashed` pattern for construction, and a fine dotted pattern for
witness lines. Reuse `segAccum` and the existing `dashPolyline` helper the sketch overlay
already uses for line-typed geometry (confirm its name with
`grep -n "func dash\|pattern" head/ui/sketch_overlay.go` and match it).

```go
// splitRecipeGeometry separates a preview recipe's real geometry from its construction
// geometry, so each can be painted in its own style.
func splitRecipeGeometry(r sketch.Recipe) (solid, construction []sketch.RecipeEntity) {
	for _, e := range r.Entities {
		if e.Construction {
			construction = append(construction, e)
			continue
		}
		solid = append(solid, e)
	}
	return solid, construction
}
```

- [ ] **Step 4: Draw the field boxes**

Boxes sit at the witness-line midpoint, offset perpendicular, clamped inside the viewport:
active = `hudActiveBgColor` fill with white text, inactive = white fill with dark text, locked =
white fill with dark text plus a padlock.

```go
// drawPadlock paints a small padlock at (x,y) sized to a text line: a filled body with a
// three-segment shackle. It marks an input field whose value the user has locked, so the drag
// no longer changes it.
func drawPadlock(x, y, h float32) {
	body := h * 0.55
	native.DrawRectFilled(x, y+h-body, x+body, y+h, padlockColor)
	top := y + h - body
	native.DrawLine(x+body*0.2, top, x+body*0.2, top-body*0.35, padlockColor, 1.2)
	native.DrawLine(x+body*0.2, top-body*0.35, x+body*0.8, top-body*0.35, padlockColor, 1.2)
	native.DrawLine(x+body*0.8, top-body*0.35, x+body*0.8, top, padlockColor, 1.2)
}
```

- [ ] **Step 5: Replace `toolPreview`**

`toolPreview` in `head/ui/chrome_viewport.go:701` currently builds one polyline. Replace its body
with a call to `placementOverlayItems`, returning the styled item list.

- [ ] **Step 6: Run the tests and build**

Run: `cd head && go build ./... && go test ./ui/ && cd ..`
Expected: builds and passes.

- [ ] **Step 7: Commit**

```bash
cd head && gofmt -w ui/ && cd ..
git add -A
git commit -m "feat(head): placement overlay with construction, witness lines and locked fields (#2014)"
```

---

### Task 10: API — heads-up display options

**Files:**

- Create: `../Oblikovati.API/types/hud_options.go`
- Modify: `../Oblikovati.API/wire/` (method constants + DTO)
- Modify: `../Oblikovati.API/client/` (typed method group)
- Modify: `../Oblikovati.API/types/sketch_settings.go` (`DisplayConstraintsOnCreation` default)
- Modify: `addin/router/` (serve the new methods)
- Test: `../Oblikovati.API/types/hud_options_test.go`, `addin/router/hud_options_test.go`

This is one consolidated API PR — batch every contract change for #2014 into it rather than
trickling them out.

- [ ] **Step 1: Write `types.HeadsUpDisplayOptions`**

```go
// SPDX-License-Identifier: Apache-2.0

package types

// HeadsUpDisplayOptions is the application's in-canvas input configuration while sketching:
// the pointer-input boxes that place a shape's first point, the dimension-input boxes that
// size it, and whether a typed value becomes a persistent dimension (#2014).
type HeadsUpDisplayOptions struct {
	// Enabled switches the whole heads-up display on or off.
	Enabled bool `json:"enabled"`
	// PointerInputEnabled shows the coordinate boxes for a shape's first point.
	PointerInputEnabled bool `json:"pointerInputEnabled"`
	// PointerInputInCartesianCoordinates shows X/Y rather than length/angle for that point.
	PointerInputInCartesianCoordinates bool `json:"pointerInputInCartesianCoordinates"`
	// DimensionInputEnabled shows the dimension boxes that size the shape being placed.
	DimensionInputEnabled bool `json:"dimensionInputEnabled"`
	// DimensionInputInCartesianCoordinates shows width/height rather than length/angle.
	DimensionInputInCartesianCoordinates bool `json:"dimensionInputInCartesianCoordinates"`
	// CreateDimensionsOnValueInput makes a typed value a persistent driving dimension.
	CreateDimensionsOnValueInput bool `json:"createDimensionsOnValueInput"`
}

// DefaultHeadsUpDisplayOptions is the out-of-the-box configuration: everything on, the first
// point entered in Cartesian coordinates and the shape sized in polar length/angle, with typed
// values persisted as dimensions.
func DefaultHeadsUpDisplayOptions() HeadsUpDisplayOptions {
	return HeadsUpDisplayOptions{
		Enabled:                              true,
		PointerInputEnabled:                  true,
		PointerInputInCartesianCoordinates:   true,
		DimensionInputEnabled:                true,
		DimensionInputInCartesianCoordinates: false,
		CreateDimensionsOnValueInput:         true,
	}
}
```

- [ ] **Step 2: Flip the glyph default**

In `../Oblikovati.API/types/sketch_settings.go`, change `DisplayConstraintsOnCreation` in
`DefaultSketchSettings` from `false` to `true`, and update the pinned assertion in
`types/sketch_settings_test.go:21` (it currently asserts the field is false).

- [ ] **Step 3: Add wire constants and DTO, then the client method group**

Follow the existing `sketch_inference.go` pattern exactly: a `MethodApplicationGetHUDOptions`
and `MethodApplicationSetHUDOptions` constant, a `HeadsUpDisplayOptionsView` DTO, and a typed
client group. Never re-declare the DTO or the method string in this module — import from
`api/wire`.

- [ ] **Step 4: Serve them in `addin/router`**

Mirror `getInferenceOptions`/`setInferenceOptions` in `addin/router/sketch_inference.go`.

- [ ] **Step 5: Honour `DisplayConstraintsOnCreation` in the overlay**

Gate the glyph pass in `head/ui/inference_glyphs.go` on the active document's setting.

- [ ] **Step 6: Run both modules' suites**

Run: `cd ../Oblikovati.API && go test ./... && cd ../Oblikovati && go test ./...`
Expected: `ok` in both.

- [ ] **Step 7: Commit**

Commit the API module and this module separately — they are separate repositories.

```bash
cd ../Oblikovati.API && git add -A && git commit -m "feat(types): heads-up display options for sketch value input (#2014)" && cd ../Oblikovati
git add -A && git commit -m "feat(router): serve heads-up display options (#2014)"
```

---

### Task 11: Verification

**Files:**

- Test: `app/sketch_creation_parity_test.go` (regression sweep)

- [ ] **Step 1: Write the regression sweep**

One table asserting the final DOF and redundancy for all 17 tools, so a later change that
over-constrains a previously-correct shape fails loudly.

- [ ] **Step 2: Run the full suite**

Run: `go test ./... && cd head && go test ./... && cd ..`
Expected: `ok` throughout.

- [ ] **Step 3: Lint**

Run: `make lint && make docs-lint`
Expected: clean. `funlen` is set to 20 — split anything longer.

- [ ] **Step 4: Coverage and duplication**

Run: `go test ./... -coverprofile=/tmp/cover.out && go tool cover -func=/tmp/cover.out | tail -1`
Expected: total above 80%.

- [ ] **Step 5: Live verification**

Launch the head via the MCP bridge, drag-create one shape from each family (line, rectangle,
slot, circle, arc, polygon), and screenshot mid-drag to confirm the overlay: solid geometry,
dashed construction, dotted witness lines, and the input boxes with the active one highlighted.
Then type a value, Tab to lock it, complete the shape, and screenshot the committed result to
confirm the locked value became a visible driving dimension.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "test(sketch): creation parity regression sweep across all 17 tools (#2014)"
```

---

## Self-Review

**Spec coverage.** Drag-create → Task 7. Preview for all tools → Tasks 4 and 6. Construction
geometry → Tasks 2–4 (centre-rectangle diagonals, slot centrelines, polygon circumcircle). Auto
constraints → Tasks 2–5. In-place dimension input → Task 8. Rendering and the padlock → Task 9.
API and the four dead settings → Task 10. Unit-strict expressions → Task 8, Step 3. Atomic
rollback and over-constrained behaviour → Task 1, Steps 4 and 6. Redundancy gating →
`assertDOF` from Task 2 onward, mutation-proofed in Tasks 2 and 3.

**Known adjustment points.** Two numbers in this plan are derived rather than measured, and the
step that could disprove them says so explicitly: the arc slot's DOF 6 (Task 3, Step 6) and the
polygon's construction-circle entity index (Task 4, Step 4). Both instruct the implementer to
re-derive against the spec arithmetic and correct the spec, not to bend the test to whatever the
code emits.

**Type consistency.** `RecipeField.Dim` is `RecipeDim` throughout. `PendingRecipe` gains its
`*Session` parameter in Task 8, Step 4, which explicitly restates the Task 6 signature change.
`ApplyWithFields` is introduced in Task 1 and consumed in Task 8. `assertDOF` and
`countConstruction` are defined once (Tasks 2 and 3) and reused thereafter.
