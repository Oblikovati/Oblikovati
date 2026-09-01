// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Corner relief geometry (#2072). Two walls folding away from one corner both want the material
// there, and folded flat there is not enough of it — so it is removed before either bend reaches
// it. The corner is found from the BENDS the walls already record, which is why the feature that
// builds the second wall is where the junction first exists.

// corneredSheet folds two flanges on ADJACENT edges of the seed sheet, so their bend lines meet at
// one corner, and returns the result with the given corner relief.
func corneredSheet(t *testing.T, corner CornerReliefSpec) *topo.Body {
	t.Helper()
	body, _ := corneredSheetAt(t, corner)
	return body
}

// corneredSheetAt is corneredSheet plus WHERE the two bends meet, so a test can sample around the
// junction the part actually has rather than the one it assumed.
func corneredSheetAt(t *testing.T, corner CornerReliefSpec) (*topo.Body, math.Point3) {
	t.Helper()
	pf, fs := corneredSheetFeature(t, corner)
	if !pf.Health().OK() {
		t.Fatalf("second flange sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("cornered sheet is not a valid solid: %+v", r)
	}
	return body, junctionOf(t, fs)
}

// junctionOf reports where the part's two bends meet.
func junctionOf(t *testing.T, fs *PartFeatures) math.Point3 {
	t.Helper()
	var bends []BendPlacement
	for i := 0; i < fs.Count(); i++ {
		if placed, ok := fs.Item(i).Definition().(PlacedBend); ok {
			if p, ok := placed.Placement(); ok {
				bends = append(bends, p)
			}
		}
	}
	if len(bends) != 2 {
		t.Fatalf("part has %d placed bends, want 2", len(bends))
	}
	j, ok := findBendJunction(bends[1], bends[:1])
	if !ok {
		t.Fatal("the two flanges do not share a corner")
	}
	return j.at
}

// corneredSheetFeature builds the pair without demanding success, returning the SECOND flange (the
// one that closes the corner) and the engine.
func corneredSheetFeature(t *testing.T, corner CornerReliefSpec) (*PartFeature, *PartFeatures) {
	t.Helper()
	return corneredSheetFeatureWith(t, corner, types.NoBendTransition)
}

// corneredSheetFeatureWith is corneredSheetFeature under a chosen bend transition.
func corneredSheetFeatureWith(t *testing.T, corner CornerReliefSpec,
	transition types.BendTransition) (*PartFeature, *PartFeatures) {
	t.Helper()
	fs, edgeX := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} }) // isolate the CORNER cut
	fs.SetCornerReliefSpec(func() CornerReliefSpec { return corner })
	fs.SetBendTransition(func() types.BendTransition { return transition })
	flanges := NewSheetMetalFlangeFeatures(fs)
	flanges.Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeX.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	edgeY := adjacentTopEdge(t, fs.Result()[0], edgeX)
	second := flanges.Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeY.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	return second, fs
}

// adjacentTopEdge finds a top-face edge that shares an endpoint with the first flanged edge and
// runs a different way — the neighbouring edge of the same corner.
func adjacentTopEdge(t *testing.T, body *topo.Body, first *topo.Edge) *topo.Edge {
	t.Helper()
	topZ := first.StartVertex().Point().Z
	fa, fb := first.StartVertex().Point(), first.EndVertex().Point()
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(float64(a.Z-topZ)) > 1e-6 || stdmath.Abs(float64(b.Z-topZ)) > 1e-6 {
			continue
		}
		if !sharesEndpoint(a, b, fa, fb) || sameRun(a, b, fa, fb) {
			continue
		}
		return e
	}
	t.Fatal("no adjacent top edge found")
	return nil
}

func sharesEndpoint(a, b, fa, fb math.Point3) bool {
	for _, p := range []math.Point3{a, b} {
		for _, q := range []math.Point3{fa, fb} {
			if float64(p.DistanceTo(q)) < 1e-6 {
				return true
			}
		}
	}
	return false
}

func sameRun(a, b, fa, fb math.Point3) bool {
	d1, err1 := math.UnitVector3FromVector(a.VectorTo(b))
	d2, err2 := math.UnitVector3FromVector(fa.VectorTo(fb))
	if err1 != nil || err2 != nil {
		return true
	}
	return stdmath.Abs(float64(d1.AsVector().Dot(d2.AsVector()))) > 0.999
}

// TestCornerReliefRemovesTheSharedCorner: two walls meeting at a corner leave material neither can
// fold. A square relief takes it away; without one, nothing is removed at all.
func TestCornerReliefRemovesTheSharedCorner(t *testing.T) {
	t.Parallel()
	square := CornerReliefSpec{Shape: types.CornerSquare, Size: 0.4}
	relieved := smSolidVolume(corneredSheet(t, square))
	bare := smSolidVolume(corneredSheet(t, CornerReliefSpec{Shape: types.CornerTear}))
	want := square.Size * square.Size * 0.2 // one square notch through the 2 mm gauge
	if got := bare - relieved; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("corner relief removed %.6f cm³, want %.6f (a %gx%g notch through the gauge)",
			got, want, square.Size, square.Size)
	}
}

// TestCornerReliefCutsAtTheCornerItself: the notch belongs AT the shared corner and nowhere else.
// A cut placed at the wrong end of either bend would remove the same volume and still be wrong.
func TestCornerReliefCutsAtTheCornerItself(t *testing.T) {
	t.Parallel()
	body, at := corneredSheetAt(t, CornerReliefSpec{Shape: types.CornerSquare, Size: 0.4})
	// Sample INTO the sheet from the junction: the notch reaches 0.4 along both edges, so a point
	// 0.2 in is inside it and one 0.6 in is past it. The sheet spans x,y ∈ [0,4].
	toward := func(d float64) (float64, float64) {
		x, y := float64(at.X), float64(at.Y)
		if x > 2 {
			x -= d
		} else {
			x += d
		}
		if y > 2 {
			y -= d
		} else {
			y += d
		}
		return x, y
	}
	inx, iny := toward(0.2)
	if materialAt(body, inx, iny) {
		t.Errorf("material still at (%.2f, %.2f), inside the corner notch", inx, iny)
	}
	outx, outy := toward(0.6)
	if !materialAt(body, outx, outy) {
		t.Errorf("no material at (%.2f, %.2f), which is past the notch", outx, outy)
	}
	if !materialAt(body, 2.0, 2.0) {
		t.Error("the middle of the sheet lost material to a corner relief")
	}
}

// TestRoundCornerReliefTakesLessThanTheSquare: the round relief seats a disc in the corner instead
// of a square, so it removes π/4 of the square's area — and leaves no inside corner.
func TestRoundCornerReliefTakesLessThanTheSquare(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	bare := smSolidVolume(corneredSheet(t, CornerReliefSpec{Shape: types.CornerTear}))
	squareCut := bare - smSolidVolume(corneredSheet(t, CornerReliefSpec{Shape: types.CornerSquare, Size: 0.4}))
	roundCut := bare - smSolidVolume(corneredSheet(t, CornerReliefSpec{Shape: types.CornerRound, Size: 0.4}))
	if !(roundCut < squareCut) || roundCut <= 0 {
		t.Errorf("round corner cut %.6f cm³ vs the square's %.6f — the round should take less, and some", roundCut, squareCut)
	}
	// A quarter-disc is π/4 of the square it sits in; the facets under-fill it slightly.
	if want := squareCut * stdmath.Pi / 4; roundCut < want*0.95 || roundCut > want*1.001 {
		t.Errorf("round corner cut %.6f cm³, want ≈%.6f (π/4 of the square)", roundCut, want)
	}
}

// TestTrimToBendSizesItselfFromTheBends: the style's default corner shape is sized by the BENDS it
// relieves, not by the style — trimming "to the bend" means back to where each bend's outer surface
// leaves the flat, which is its radius plus the thickness.
func TestTrimToBendSizesItselfFromTheBends(t *testing.T) {
	t.Parallel()
	bare := smSolidVolume(corneredSheet(t, CornerReliefSpec{Shape: types.CornerTear}))
	// The style size is deliberately absurd: trimToBend must ignore it and use the bends.
	trimmed := smSolidVolume(corneredSheet(t, CornerReliefSpec{Shape: types.CornerTrimToBend, Size: 99}))
	reach := 0.3 + 0.2 // the flanges' radius plus the gauge
	if want := reach * reach * 0.2; stdmath.Abs(bare-trimmed-want) > 1e-6 {
		t.Errorf("trimToBend removed %.6f cm³, want %.6f (a %gx%g notch)", bare-trimmed, want, reach, reach)
	}
}

// TestUnbuiltCornerShapesAreRefused: the shapes that need the two walls' own outlines are refused
// rather than approximated by a square, which would cut the wrong corner and still look plausible.
func TestUnbuiltCornerShapesAreRefused(t *testing.T) {
	t.Parallel()
	for _, shape := range []types.CornerReliefShape{
		types.CornerFullRound, types.CornerRoundWithRadius, types.CornerIntersection,
	} {
		pf, _ := corneredSheetFeature(t, CornerReliefSpec{Shape: shape, Size: 0.4})
		if pf.Health().OK() {
			t.Errorf("corner shape %v should be refused until it is built", shape)
		}
	}
}

// TestSizedCornerShapeNeedsASize: round and square are sized by the style, so a zero size is a
// style mistake worth naming rather than a corner quietly left unrelieved.
func TestSizedCornerShapeNeedsASize(t *testing.T) {
	t.Parallel()
	pf, _ := corneredSheetFeature(t, CornerReliefSpec{Shape: types.CornerSquare, Size: 0})
	if pf.Health().OK() {
		t.Error("a square corner relief with no size should be refused")
	}
}

// TestOneFlangeHasNoCorner: a lone wall corners with nothing, so nothing is cut — and a part with
// two flanges on OPPOSITE edges has two bends that never meet either.
func TestOneFlangeHasNoCorner(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} })
	fs.SetCornerReliefSpec(func() CornerReliefSpec {
		return CornerReliefSpec{Shape: types.CornerSquare, Size: 0.4}
	})
	NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	withRelief := smSolidVolume(fs.Result()[0])

	plain, edge2 := seedSheetMetalSheet(t, 4, nil)
	plain.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} })
	NewSheetMetalFlangeFeatures(plain).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge2.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	plain.Recompute()
	if stdmath.Abs(withRelief-smSolidVolume(plain.Result()[0])) > 1e-9 {
		t.Error("a single flange lost material to a corner relief; it has no corner")
	}
}

// TestParallelBendsShareNoCorner: two stretches of ONE edge are not a corner. Relieving between
// them would notch the middle of a straight run.
func TestParallelBendsShareNoCorner(t *testing.T) {
	t.Parallel()
	a := BendPlacement{AxisStart: math.P3(0, 0, 0), AxisEnd: math.P3(2, 0, 0)}
	b := BendPlacement{AxisStart: math.P3(2, 0, 0), AxisEnd: math.P3(4, 0, 0)}
	if _, ok := findBendJunction(a, []BendPlacement{b}); ok {
		t.Error("two stretches of one straight edge should not read as a corner")
	}
	perpendicular := BendPlacement{AxisStart: math.P3(0, 0, 0), AxisEnd: math.P3(0, 3, 0)}
	if _, ok := findBendJunction(a, []BendPlacement{perpendicular}); !ok {
		t.Error("two bends meeting at a point and running different ways are a corner")
	}
}

// TestBendTransitionOnlyMattersAtAJunction (#1959): a transition shapes where a bend runs into the
// face beside it, so a part whose style names one it never reaches must build fine. Refusing on
// the style alone would break every single-flange part the moment the style changed.
func TestBendTransitionOnlyMattersAtAJunction(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} })
	fs.SetBendTransition(func() types.BendTransition { return types.ArcBendTransition })
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Errorf("a lone flange went sick over a transition it never reaches: %+v", pf.Health())
	}
}

// TestUnbuiltBendTransitionsAreRefusedAtAJunction: the shaping transitions describe the FLAT
// PATTERN's outline through the transition region, which this flat does not model. They are
// refused where they would apply rather than silently ignored, which would report a part shaped one
// way and build it another.
func TestUnbuiltBendTransitionsAreRefusedAtAJunction(t *testing.T) {
	t.Parallel()
	for _, kind := range []types.BendTransition{
		types.IntersectionBendTransition, types.StraightLineBendTransition,
		types.ArcBendTransition, types.TrimToBendBendTransition,
	} {
		pf, _ := corneredSheetFeatureWith(t, CornerReliefSpec{Shape: types.CornerTear}, kind)
		if pf.Health().OK() {
			t.Errorf("bend transition %v should be refused at a junction until it is built", kind)
		}
	}
}

// TestNoTransitionBuildsTheJunction: "none" is Inventor's default and what this build makes, so a
// junction under it is healthy.
func TestNoTransitionBuildsTheJunction(t *testing.T) {
	t.Parallel()
	pf, _ := corneredSheetFeatureWith(t, CornerReliefSpec{Shape: types.CornerTear}, types.NoBendTransition)
	if !pf.Health().OK() {
		t.Errorf("a junction with no transition went sick: %+v", pf.Health())
	}
}

// TestPerBendTransitionOverridesTheStyle: one bend can name its own transition, and "default"
// defers — so a feature that sets other options does not accidentally opt out of the style's.
func TestPerBendTransitionOverridesTheStyle(t *testing.T) {
	t.Parallel()
	f := &SheetMetalFlangeFeature{def: &SheetMetalFlangeDefinition{}}
	style := Input{Transition: types.ArcBendTransition}
	if got := f.bendTransition(style); got != types.ArcBendTransition {
		t.Errorf("a flange with no options used %v, want the style's arc", got)
	}
	f.def.Options = &BendOptions{Transition: types.DefaultBendTransition}
	if got := f.bendTransition(style); got != types.ArcBendTransition {
		t.Errorf("an explicit default used %v, want the style's arc", got)
	}
	f.def.Options = &BendOptions{Transition: types.NoBendTransition}
	if got := f.bendTransition(style); got != types.NoBendTransition {
		t.Errorf("an overriding flange used %v, want its own none", got)
	}
}
