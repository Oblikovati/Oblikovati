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

// Bend relief geometry (#2072). A bend that stops short of the material's edge tears without a
// notch at each end, and until width extents landed (#1958) every flange spanned its whole edge —
// so this is the first geometry the styled relief has ever cut.

// relievedFlange folds a 2 cm tab centred on the seed sheet's 4 cm edge, with the given styled
// relief, and returns the result.
func relievedFlange(t *testing.T, spec ReliefSpec, width FlangeWidth) *topo.Body {
	t.Helper()
	pf, fs := relievedFlangeFeature(t, spec, width)
	if !pf.Health().OK() {
		t.Fatalf("relieved flange sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("relieved sheet is not a valid solid: %+v", r)
	}
	return body
}

// relievedFlangeFeature builds the flange without demanding success.
func relievedFlangeFeature(t *testing.T, spec ReliefSpec, width FlangeWidth) (*PartFeature, *PartFeatures) {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return spec })
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
		Width: width,
	})
	fs.Recompute()
	return pf, fs
}

// centredTab is a 2 cm wall centred on the 4 cm edge, so its bend ends at x=1 and x=3 — both well
// inside the material, which is exactly when relief is needed.
var centredTab = FlangeWidth{Type: WidthCentered, Width: constClosure(2.0)}

// straightRelief is a 2 mm × 1 mm rectangular notch.
var straightRelief = ReliefSpec{Shape: types.ReliefStraight, Width: 0.2, Depth: 0.1}

// materialAt reports whether the body has material at the given point, sampling inside the sheet's
// thickness so the answer is about the notch and not about which face a point lies on.
func materialAt(body *topo.Body, x, y float64) bool {
	return ops.PointInsideBody(body, math.P3(math.Scalar(x), math.Scalar(y), 0.1))
}

// TestBendReliefNotchesTheBendEnds: the notch is cut BESIDE each bend end, into the parent. The
// material just outside the tab's span has to go; the material just inside it must not, or the cut
// has eaten the wall it exists to protect.
func TestBendReliefNotchesTheBendEnds(t *testing.T) {
	t.Parallel()
	body := relievedFlange(t, straightRelief, centredTab)
	// The tab spans x ∈ [1,3] of the edge at y=0, and the sheet runs to y=4.
	for _, c := range []struct {
		name string
		x, y float64
		want bool
	}{
		{"just outside the low end", 0.9, 0.05, false},
		{"just outside the high end", 3.1, 0.05, false},
		{"just inside the low end", 1.1, 0.05, true},
		{"just inside the high end", 2.9, 0.05, true},
		{"beyond the notch depth", 0.9, 0.2, true},
		{"well away from the bend", 0.5, 2.0, true},
	} {
		if got := materialAt(body, c.x, c.y); got != c.want {
			t.Errorf("%s (%.2f, %.2f): material=%v, want %v", c.name, c.x, c.y, got, c.want)
		}
	}
}

// TestBendReliefRemovesItsOwnVolume: two notches of width×depth×thickness, and no more. A cut that
// ran the wrong way or the wrong distance would still leave a valid solid.
func TestBendReliefRemovesItsOwnVolume(t *testing.T) {
	t.Parallel()
	relieved := smSolidVolume(relievedFlange(t, straightRelief, centredTab))
	bare := smSolidVolume(relievedFlange(t, ReliefSpec{}, centredTab))
	want := 2 * straightRelief.Width * straightRelief.Depth * 0.2 // two notches, 2 mm gauge
	if got := bare - relieved; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("relief removed %.6f cm³, want %.6f (two %gx%g notches through the gauge)",
			got, want, straightRelief.Width, straightRelief.Depth)
	}
}

// TestFullWidthFlangeIsNotRelieved: a flange that spans its whole edge has no bend end inside the
// material, so there is nothing to tear and nothing to cut. Relieving it anyway would notch the
// part's outside corners.
func TestFullWidthFlangeIsNotRelieved(t *testing.T) {
	t.Parallel()
	relieved := smSolidVolume(relievedFlange(t, straightRelief, FlangeWidth{}))
	bare := smSolidVolume(relievedFlange(t, ReliefSpec{}, FlangeWidth{}))
	if stdmath.Abs(relieved-bare) > 1e-9 {
		t.Errorf("a full-width flange lost %.6f cm³ to relief, want none", bare-relieved)
	}
}

// TestOneSidedTabIsRelievedOnlyWhereItStops: a tab that starts at the edge's own end has one bend
// end on the boundary and one inside. Only the inside one takes a notch.
func TestOneSidedTabIsRelievedOnlyWhereItStops(t *testing.T) {
	t.Parallel()
	width := FlangeWidth{Type: WidthOffsetAndWidth, Offset: constClosure(0), Width: constClosure(2.0)}
	relieved := smSolidVolume(relievedFlange(t, straightRelief, width))
	bare := smSolidVolume(relievedFlange(t, ReliefSpec{}, width))
	want := straightRelief.Width * straightRelief.Depth * 0.2 // ONE notch
	if got := bare - relieved; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("relief removed %.6f cm³, want %.6f (one notch — the other end is the part's edge)", got, want)
	}
}

// TestTearReliefCutsNothing: a tear relief is the deliberate ABSENCE of a cut — the material is
// left to tear along the bend. Treating it as a missing value and cutting a default notch would
// change the part.
func TestTearReliefCutsNothing(t *testing.T) {
	t.Parallel()
	tear := ReliefSpec{Shape: types.ReliefTear, Width: 0.2, Depth: 0.1}
	relieved := smSolidVolume(relievedFlange(t, tear, centredTab))
	bare := smSolidVolume(relievedFlange(t, ReliefSpec{}, centredTab))
	if stdmath.Abs(relieved-bare) > 1e-9 {
		t.Errorf("a tear relief removed %.6f cm³, want none", bare-relieved)
	}
}

// TestRoundReliefRemovesLessThanTheStraightOne: the round relief replaces the notch's inner end
// with a half-round, which takes away less material than the square corner it replaces — and
// leaves no inside corner for a crack to start in.
func TestRoundReliefRemovesLessThanTheStraightOne(t *testing.T) {
	t.Parallel()
	round := ReliefSpec{Shape: types.ReliefRound, Width: 0.2, Depth: 0.1}
	bare := smSolidVolume(relievedFlange(t, ReliefSpec{}, centredTab))
	straightCut := bare - smSolidVolume(relievedFlange(t, straightRelief, centredTab))
	roundCut := bare - smSolidVolume(relievedFlange(t, round, centredTab))
	if !(roundCut < straightCut) {
		t.Errorf("round relief removed %.6f cm³ and the straight one %.6f — the round should take less",
			roundCut, straightCut)
	}
	if roundCut <= 0 {
		t.Error("round relief removed nothing")
	}
}

// TestBendReliefCutsInwardWhicheverWayTheFlangeFolds: the notch is cut INTO the parent, and which
// way that is depends on the frame the wall was built in — a flipped flange folds the other way and
// mirrors the plane the notch is drawn in. Cutting on the wrong side removes nothing at all (the
// tool would sit off the part), so both orientations are measured.
func TestBendReliefCutsInwardWhicheverWayTheFlangeFolds(t *testing.T) {
	t.Parallel()
	for name, flip := range map[string]bool{"folded up": false, "folded down": true} {
		t.Run(name, func(t *testing.T) {
			cut := reliefCutVolume(t, flip)
			want := 2 * straightRelief.Width * straightRelief.Depth * 0.2
			if stdmath.Abs(cut-want) > 1e-6 {
				t.Errorf("relief removed %.6f cm³, want %.6f — the notch was cut off the part", cut, want)
			}
		})
	}
}

// reliefCutVolume is how much material the styled relief removes from a centred tab folded the
// given way.
func reliefCutVolume(t *testing.T, flip bool) float64 {
	t.Helper()
	build := func(spec ReliefSpec) float64 {
		fs, edge := seedSheetMetalSheet(t, 4, nil)
		fs.SetReliefSpec(func() ReliefSpec { return spec })
		pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
			EdgeKey: edge.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
			Width: centredTab, Flip: flip,
		})
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("flange (flip=%v) sick: %+v", flip, pf.Health())
		}
		return smSolidVolume(fs.Result()[0])
	}
	return build(ReliefSpec{}) - build(straightRelief)
}

// TestReliefEndsSkipTheBoundary: the decision of which ends need relieving is what keeps a notch
// from hanging off the part, so it is pinned directly.
func TestReliefEndsSkipTheBoundary(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		from, to     float64
		wantF, wantT bool
	}{
		"full edge":  {0, 4, false, false},
		"centred":    {1, 3, true, true},
		"from start": {0, 2, false, true},
		"to end":     {2, 4, true, false},
	} {
		t.Run(name, func(t *testing.T) {
			ends := bendReliefEnds(c.from, c.to, 4)
			if ends.relieveFrom != c.wantF || ends.relieveTo != c.wantT {
				t.Errorf("relieve [%v %v], want [%v %v]", ends.relieveFrom, ends.relieveTo, c.wantF, c.wantT)
			}
		})
	}
}

// Per-feature bend options (#1959). The style says what every bend does; one bend can say
// otherwise. These matter now only because the relief actually cuts — an override on inert data
// would have been unobservable.

// TestBendOptionsOverrideTheStyleRelief: a flange with its own relief cuts ITS notch, not the
// style's, and the difference is measurable.
func TestBendOptionsOverrideTheStyleRelief(t *testing.T) {
	t.Parallel()
	deep := ReliefSpec{Shape: types.ReliefStraight, Width: 0.4, Depth: 0.3}
	styled := smSolidVolume(relievedFlange(t, straightRelief, centredTab))
	overridden := smSolidVolume(relievedFlangeWithOptions(t, straightRelief, &BendOptions{
		ReliefWidth: constClosure(deep.Width), ReliefDepth: constClosure(deep.Depth),
	}))
	bare := smSolidVolume(relievedFlange(t, ReliefSpec{}, centredTab))
	if want := 2 * deep.Width * deep.Depth * 0.2; stdmath.Abs(bare-overridden-want) > 1e-6 {
		t.Errorf("the override cut %.6f cm³, want %.6f", bare-overridden, want)
	}
	if overridden >= styled {
		t.Error("the override should cut more than the style's smaller notch")
	}
}

// TestBendOptionsLeaveUnsetFieldsToTheStyle: an override is not a restatement — a flange that sets
// only a depth keeps the style's shape and width.
func TestBendOptionsLeaveUnsetFieldsToTheStyle(t *testing.T) {
	t.Parallel()
	got := straightRelief
	got = (&BendOptions{ReliefDepth: constClosure(0.3)}).resolve(got)
	if got.Shape != straightRelief.Shape || got.Width != straightRelief.Width {
		t.Errorf("resolved relief = %+v, want the style's shape and width kept", got)
	}
	if got.Depth != 0.3 {
		t.Errorf("resolved depth = %g, want the overridden 0.3", got.Depth)
	}
	if none := (*BendOptions)(nil).resolve(straightRelief); none != straightRelief {
		t.Errorf("no options resolved to %+v, want the style unchanged", none)
	}
}

// TestBendOptionsCanTearOneBend: a per-bend tear turns the cut off for that bend while the style
// keeps relieving every other one.
func TestBendOptionsCanTearOneBend(t *testing.T) {
	t.Parallel()
	tear := types.ReliefTear
	relieved := smSolidVolume(relievedFlangeWithOptions(t, straightRelief, &BendOptions{ReliefShape: &tear}))
	bare := smSolidVolume(relievedFlange(t, ReliefSpec{}, centredTab))
	if stdmath.Abs(relieved-bare) > 1e-9 {
		t.Errorf("a torn bend lost %.6f cm³, want none", bare-relieved)
	}
}

// TestMinimumRemnantSwallowsTheSliver (#1959): a notch that would leave a strip of parent material
// thinner than the remnant takes the strip with it. Leaving it is worse than removing it — a sliver
// that thin tears off in handling, so the part that arrives is not the part that was drawn.
func TestMinimumRemnantSwallowsTheSliver(t *testing.T) {
	t.Parallel()
	// A tab from 0.3 to 3.7 leaves 0.3 of material outside each bend end; a 0.2-wide notch would
	// leave a 0.1 sliver beyond it.
	tab := FlangeWidth{Type: WidthOffsets, Offset: constClosure(0.3), Offset2: constClosure(0.3)}
	withRemnant := smSolidVolume(relievedFlangeWith(t, straightRelief, tab, &BendOptions{
		MinimumRemnant: constClosure(0.25),
	}))
	without := smSolidVolume(relievedFlangeWith(t, straightRelief, tab, nil))
	// Each notch grows from 0.2 to the full 0.3 remaining, so the cut grows by 0.1 per end.
	if want := 2 * 0.1 * straightRelief.Depth * 0.2; stdmath.Abs(without-withRemnant-want) > 1e-6 {
		t.Errorf("the remnant rule removed a further %.6f cm³, want %.6f (the two slivers)",
			without-withRemnant, want)
	}
}

// TestMinimumRemnantLeavesHealthyMaterialAlone: a strip thicker than the remnant survives, or the
// rule would eat the part's edge on every relieved bend.
func TestMinimumRemnantLeavesHealthyMaterialAlone(t *testing.T) {
	t.Parallel()
	generous := smSolidVolume(relievedFlangeWith(t, straightRelief, centredTab, &BendOptions{
		MinimumRemnant: constClosure(0.25),
	}))
	plain := smSolidVolume(relievedFlangeWith(t, straightRelief, centredTab, nil))
	if stdmath.Abs(generous-plain) > 1e-9 {
		t.Errorf("the remnant rule removed %.6f cm³ from a tab with 0.8 of material to spare", plain-generous)
	}
}

// relievedFlangeWithOptions folds the centred tab with per-feature bend options.
func relievedFlangeWithOptions(t *testing.T, spec ReliefSpec, opts *BendOptions) *topo.Body {
	t.Helper()
	return relievedFlangeWith(t, spec, centredTab, opts)
}

// relievedFlangeWith folds a tab of the given width with the given style relief and overrides.
func relievedFlangeWith(t *testing.T, spec ReliefSpec, width FlangeWidth, opts *BendOptions) *topo.Body {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return spec })
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
		Width: width, Options: opts,
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("flange sick: %+v", pf.Health())
	}
	return fs.Result()[0]
}
