// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// These tests exercise the full edit→recompute→solve path for parameter-driven sketch
// dimensions — 2D and 3D — and the cross-sketch case where geometry constrained in several
// 2D sketches is included into one 3D sketch. They assert through PartComponentDefinition
// (not sketch.Solve directly), so they cover the same path the UI/router edit verbs take:
// SetExpression → RecomputeAfterChange → solveSketches / solveSketches3D → UpdateIncluded.

// --- helpers ---------------------------------------------------------------------------

// dimLine2D adds a 2D sketch on plane with a single line whose two endpoints are dimensioned
// distance=expr (referencing a user parameter), and returns the sketch and the two endpoints.
// The endpoints stay otherwise free, so the solver fixes only their separation — the distance
// is the deterministic, parameter-driven quantity to assert.
func dimLine2D(t *testing.T, def *compdef.PartComponentDefinition, plane sketch.Plane, expr string) (*sketch.Sketch, *sketch.Point, *sketch.Point) {
	t.Helper()
	sk := def.Sketches().Add(plane)
	p0 := sk.Points().Add(math.P2(0, 0))
	p1 := sk.Points().Add(math.P2(1, 0))
	sk.Lines().Add(p0, p1)
	if _, err := sk.DimensionConstraints().AddDistance(p0, p1, expr); err != nil {
		t.Fatalf("AddDistance(%q): %v", expr, err)
	}
	return sk, p0, p1
}

func mustUserParam(t *testing.T, def *compdef.PartComponentDefinition, name, expr string) {
	t.Helper()
	if _, err := def.Parameters().AddUserParameter(name, expr); err != nil {
		t.Fatalf("AddUserParameter(%q=%q): %v", name, expr, err)
	}
}

func setParam(t *testing.T, def *compdef.PartComponentDefinition, name, expr string) {
	t.Helper()
	p, ok := def.Parameters().ByName(name)
	if !ok {
		t.Fatalf("parameter %q not found", name)
	}
	if err := def.Parameters().SetExpression(p.ID(), expr); err != nil {
		t.Fatalf("SetExpression(%q=%q): %v", name, expr, err)
	}
}

// dist2D measures the current distance between two 2D sketch points (database units).
func dist2D(p0, p1 *sketch.Point) float64 { return p0.Position().DistanceTo(p1.Position()) }

const dimTol = 1e-6

// --- 2D: dimension constrained by parameter --------------------------------------------

// A 2D distance dimension bound to a user parameter must take the parameter's value on the
// first recompute and follow every later edit.
func TestSketch2DDistanceDimensionFollowsParameterOnRecompute(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	mustUserParam(t, def, "w", "3 cm")
	_, p0, p1 := dimLine2D(t, def, sketch.XYPlane(), "w")

	def.Recompute()
	if d := dist2D(p0, p1); d < 3-dimTol || d > 3+dimTol {
		t.Fatalf("after recompute, distance = %v cm, want 3 (= w)", d)
	}

	setParam(t, def, "w", "7 cm")
	def.RecomputeAfterChange()
	if d := dist2D(p0, p1); d < 7-dimTol || d > 7+dimTol {
		t.Errorf("after w→7, distance = %v cm, want 7", d)
	}
}

// A dimension bound to a parameter whose expression references ANOTHER parameter must follow
// the transitive edit (w changes → h = w/2 changes → the dimension moves).
func TestSketch2DDimensionFollowsDependentParameterExpression(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	mustUserParam(t, def, "w", "10 cm")
	mustUserParam(t, def, "h", "w / 2")
	_, p0, p1 := dimLine2D(t, def, sketch.XYPlane(), "h")

	def.Recompute()
	if d := dist2D(p0, p1); d < 5-dimTol || d > 5+dimTol {
		t.Fatalf("after recompute, distance = %v cm, want 5 (= w/2)", d)
	}

	setParam(t, def, "w", "16 cm") // h = 8
	def.RecomputeAfterChange()
	if d := dist2D(p0, p1); d < 8-dimTol || d > 8+dimTol {
		t.Errorf("after w→16, distance = %v cm, want 8 (= w/2, transitive)", d)
	}
}

// Independent dimensions on independent parameters in the same 2D sketch must each follow
// their own parameter, and editing one must not disturb the other.
func TestSketch2DIndependentDimensionsTrackOwnParameters(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	mustUserParam(t, def, "a", "4 cm")
	mustUserParam(t, def, "b", "9 cm")
	skA, a0, a1 := dimLine2D(t, def, sketch.XYPlane(), "a")
	_ = skA
	skB, b0, b1 := dimLine2D(t, def, sketch.XZPlane(), "b")
	_ = skB

	def.Recompute()
	setParam(t, def, "a", "6 cm")
	def.RecomputeAfterChange()

	if d := dist2D(a0, a1); d < 6-dimTol || d > 6+dimTol {
		t.Errorf("dim a after a→6 = %v, want 6", d)
	}
	if d := dist2D(b0, b1); d < 9-dimTol || d > 9+dimTol {
		t.Errorf("dim b disturbed by a's edit = %v, want 9 (unchanged)", d)
	}
}

// --- 3D: dimension constrained by parameter --------------------------------------------

// dimLine3D adds a 3D sketch with one line dimensioned distance=expr and returns the sketch
// and the line. The line length is the deterministic, parameter-driven quantity to assert.
func dimLine3D(t *testing.T, def *compdef.PartComponentDefinition, expr string) (*sketch.Sketch3D, *sketch.Line3D) {
	t.Helper()
	sk := def.Sketches3D().Add()
	line := sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	if _, err := sk.DimensionConstraints3D().AddDistance(line.StartPoint(), line.EndPoint(), expr); err != nil {
		t.Fatalf("AddDistance3D(%q): %v", expr, err)
	}
	return sk, line
}

// A 3D dimension bound to a parameter whose expression references another parameter must
// follow the transitive edit on recompute (the 3D analogue of the 2D dependent-expression
// test, and a guard that 3D re-solving honours the parameter graph — Oblikovati#1566).
func TestSketch3DDimensionFollowsDependentParameterExpression(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	mustUserParam(t, def, "w", "10 cm")
	mustUserParam(t, def, "h", "w / 2")
	_, line := dimLine3D(t, def, "h")

	def.Recompute()
	if l := float64(line.Length()); l < 5-dimTol || l > 5+dimTol {
		t.Fatalf("after recompute, 3D line length = %v cm, want 5 (= w/2)", l)
	}

	setParam(t, def, "w", "20 cm") // h = 10
	def.RecomputeAfterChange()
	if l := float64(line.Length()); l < 10-dimTol || l > 10+dimTol {
		t.Errorf("after w→20, 3D line length = %v cm, want 10 (= w/2, transitive)", l)
	}
}

// Independent dimensions on independent parameters across two 3D sketches must each follow
// their own parameter, and editing one must not disturb the other.
func TestSketch3DIndependentDimensionsTrackOwnParameters(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	mustUserParam(t, def, "p", "4 cm")
	mustUserParam(t, def, "q", "9 cm")
	_, lineP := dimLine3D(t, def, "p")
	_, lineQ := dimLine3D(t, def, "q")

	def.Recompute()
	setParam(t, def, "p", "6 cm")
	def.RecomputeAfterChange()

	if l := float64(lineP.Length()); l < 6-dimTol || l > 6+dimTol {
		t.Errorf("3D dim p after p→6 = %v, want 6", l)
	}
	if l := float64(lineQ.Length()); l < 9-dimTol || l > 9+dimTol {
		t.Errorf("3D dim q disturbed by p's edit = %v, want 9 (unchanged)", l)
	}
}

// --- projection: constrained 2D entities included into a 3D sketch ----------------------

// included2D adds a parameter-dimensioned line to a 2D sketch on plane, includes BOTH its
// endpoints into the 3D sketch, and returns the two included 3D points. The separation of the
// included points equals the source dimension's parameter (a rigid plane lift preserves
// distance), so it is the deterministic quantity to assert across recomputes.
func included2D(t *testing.T, def *compdef.PartComponentDefinition, sk3 *sketch.Sketch3D, plane sketch.Plane, expr string) (*sketch.IncludedPoint3D, *sketch.IncludedPoint3D) {
	t.Helper()
	sk, p0, p1 := dimLine2D(t, def, plane, expr)
	inc0 := sk3.IncludePoint3D(sketch.NewSketch2DPointSource(sk, p0.EntityID()))
	inc1 := sk3.IncludePoint3D(sketch.NewSketch2DPointSource(sk, p1.EntityID()))
	return inc0, inc1
}

func dist3D(a, b *sketch.IncludedPoint3D) float64 { return a.Position().DistanceTo(b.Position()) }

// Geometry constrained by parameters in three different 2D sketches (on XY, XZ and YZ) and
// included into one 3D sketch must, after recompute, present each source's parameter-driven
// separation in 3D — and follow when the parameters change. This covers the chain
// SetExpression → 2D solve → UpdateIncluded (the 2D→3D include refresh) end to end.
func TestIncludedGeometryFromConstrained2DSketchesFollowsParametersInto3D(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	mustUserParam(t, def, "ax", "3 cm")
	mustUserParam(t, def, "by", "4 cm")
	mustUserParam(t, def, "cz", "5 cm")

	sk3 := def.Sketches3D().Add()
	ax0, ax1 := included2D(t, def, sk3, sketch.XYPlane(), "ax")
	by0, by1 := included2D(t, def, sk3, sketch.XZPlane(), "by")
	cz0, cz1 := included2D(t, def, sk3, sketch.YZPlane(), "cz")

	def.Recompute()
	for _, c := range []struct {
		name string
		a, b *sketch.IncludedPoint3D
		want float64
	}{
		{"ax", ax0, ax1, 3}, {"by", by0, by1, 4}, {"cz", cz0, cz1, 5},
	} {
		if d := dist3D(c.a, c.b); d < c.want-dimTol || d > c.want+dimTol {
			t.Errorf("included %s separation in 3D = %v, want %v (from its 2D dimension)", c.name, d, c.want)
		}
	}

	// Drive all three parameters; the included 3D geometry must follow each source sketch.
	setParam(t, def, "ax", "8 cm")
	setParam(t, def, "by", "1 cm")
	setParam(t, def, "cz", "6 cm")
	def.RecomputeAfterChange()

	for _, c := range []struct {
		name string
		a, b *sketch.IncludedPoint3D
		want float64
	}{
		{"ax", ax0, ax1, 8}, {"by", by0, by1, 1}, {"cz", cz0, cz1, 6},
	} {
		if d := dist3D(c.a, c.b); d < c.want-dimTol || d > c.want+dimTol {
			t.Errorf("after edit, included %s separation in 3D = %v, want %v", c.name, d, c.want)
		}
	}
}

// A curve (not just points) included from a parameter-constrained 2D sketch into a 3D sketch
// must present the source's parameter-driven length in 3D and follow edits — exercising the
// IncludeCurve3D path alongside IncludePoint3D.
func TestIncludedCurveFromConstrained2DSketchFollowsParameterInto3D(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	mustUserParam(t, def, "len", "5 cm")
	sk, p0, p1 := dimLine2D(t, def, sketch.XYPlane(), "len")
	_, _ = p0, p1

	line := sk.Lines().Item(0)
	sk3 := def.Sketches3D().Add()
	inc := sk3.IncludeCurve3D(sketch.NewSketch2DCurveSource(sk, line.EntityID()))

	def.Recompute()
	if l := includedCurveLength(inc); l < 5-dimTol || l > 5+dimTol {
		t.Fatalf("included curve length in 3D = %v, want 5 (= len)", l)
	}

	setParam(t, def, "len", "11 cm")
	def.RecomputeAfterChange()
	if l := includedCurveLength(inc); l < 11-dimTol || l > 11+dimTol {
		t.Errorf("after len→11, included curve length = %v, want 11", l)
	}
}

// A 3D sketch that DIMENSIONS one of its points against geometry included from a 2D sketch
// must be correct after a SINGLE recompute — not converge over several. This requires the 2D
// source to be solved and its include refreshed into the 3D sketch BEFORE the 3D sketch
// solves (sketch dependencies computed first, in creation order). The 2D source point moves
// on its solve, so a stale (pre-solve) include anchor would place the dependent 3D point at
// the wrong distance; this pins the single-recompute ordering.
func TestSketch3DDimensionAgainstIncluded2DPointIsCorrectInOneRecompute(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	mustUserParam(t, def, "seg", "5 cm") // drives the 2D source (moves the included point)
	mustUserParam(t, def, "gap", "7 cm") // drives the 3D dimension against the included anchor

	// 2D source on XY: a line whose endpoint A is pulled to distance "seg" from O, so A moves
	// well away from its authored position when the sketch solves.
	src := def.Sketches().Add(sketch.XYPlane())
	o := src.Points().Add(math.P2(0, 0))
	a := src.Points().Add(math.P2(1, 0))
	src.Lines().Add(o, a)
	if _, err := src.DimensionConstraints().AddDistance(o, a, "seg"); err != nil {
		t.Fatalf("AddDistance(2D): %v", err)
	}

	// 3D sketch: include A as a fixed anchor, add a free point B, and dimension B to the
	// anchor at "gap". B depends on the included anchor, which depends on the 2D solve.
	sk3 := def.Sketches3D().Add()
	incA := sk3.IncludePoint3D(sketch.NewSketch2DPointSource(src, a.EntityID()))
	b := sk3.AddPoint3D(math.P3(0, 0, 0))
	if _, err := sk3.DimensionConstraints3D().AddDistance(incA.Anchor(), b, "gap"); err != nil {
		t.Fatalf("AddDistance(3D): %v", err)
	}

	def.Recompute()

	if d := incA.Anchor().Position().DistanceTo(b.Position()); d < 7-dimTol || d > 7+dimTol {
		t.Errorf("3D point dimensioned to an included 2D point = %v from the anchor, want 7 (= gap) in one recompute; a stale include anchor during the 3D solve causes this", float64(d))
	}
}

// includedCurveLength sums the segment lengths of an included 3D curve's polyline.
func includedCurveLength(c *sketch.IncludedCurve3D) float64 {
	pts := c.Points()
	total := 0.0
	for i := 1; i < len(pts); i++ {
		total += pts[i-1].DistanceTo(pts[i])
	}
	return total
}
