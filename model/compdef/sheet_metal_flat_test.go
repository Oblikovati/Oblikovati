// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sheetmetal"
	"oblikovati.org/model/sketch"
)

// flatVolume is the developed flat body's mesh volume at a fine chord tolerance.
func flatVolume(body *topo.Body) float64 {
	return query.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-3}).Volume
}

// developedTabLength is the expected flat tab length for the default-rule fixture: the bend
// allowance of a 90° bend plus the flange's straight run.
func developedTabLength(rule *sheetmetal.Rule, height float64) float64 {
	return rule.Unfold().BendAllowance(math.Pi/2, rule.BendRadius(), rule.Thickness()) + height
}

// TestUnfoldDevelopsWatertightFlat a flanged sheet unfolds to one watertight flat solid whose
// footprint is the base square plus the developed flange tab, and whose extents grow with the
// bend allowance — the flat-pattern acceptance criterion.
func TestUnfoldDevelopsWatertightFlat(t *testing.T) {
	t.Parallel()
	const side, height = 4.0, 1.0
	d, _ := sheetWithFlange(t) // square side=4, 90° flange height=1, default rule

	fp, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold: %v", err)
	}
	if !fp.Body.IsSolid() {
		t.Fatal("flat pattern body is not a solid")
	}
	if open := ops.BoundaryEdges(fp.Body); len(open) != 0 {
		t.Errorf("flat has %d boundary edges, want 0 (watertight)", len(open))
	}
	if r := ops.Validate(fp.Body); !r.Valid {
		t.Errorf("flat failed validation: %+v", r.Issues)
	}

	// Footprint: base side² plus the tab (side × developed length); volume = footprint × gauge.
	rule := d.SheetMetal()
	tab := developedTabLength(rule, height)
	wantVol := (side*side + side*tab) * rule.Thickness()
	if got := flatVolume(fp.Body); math.Abs(got-wantVol)/wantVol > 1e-3 {
		t.Errorf("flat volume = %.5f, want %.5f", got, wantVol)
	}

	// Extents: one side stays at the base width; the other grows by the developed tab.
	dx, dy := float64(fp.Extents.Diagonal().X), float64(fp.Extents.Diagonal().Y)
	long, short := math.Max(dx, dy), math.Min(dx, dy)
	if math.Abs(short-side) > 1e-6 {
		t.Errorf("flat short extent = %.5f, want %.5f", short, side)
	}
	if math.Abs(long-(side+tab)) > 1e-6 {
		t.Errorf("flat long extent = %.5f, want %.5f (side + developed tab)", long, side+tab)
	}
	if len(fp.Bends) != 1 || math.Abs(fp.Bends[0].Angle-math.Pi/2) > 1e-12 {
		t.Errorf("flat bends = %+v, want one 90° fold line", fp.Bends)
	}
}

// TestUnfoldDevelopsPunchRepresentation a coplanar (base-plane) punch develops into a flat punch
// representation: its outline projected into the base plane, tagged with the feature name (#378).
func TestUnfoldDevelopsPunchRepresentation(t *testing.T) {
	t.Parallel()
	d, _ := sheetWithFlange(t)
	sk := d.Sketches().Add(sketch.XYPlane())
	sk.AddRectangleByCorners(gmath.P2(1, 1), gmath.P2(2, 2))
	punch := feature.NewSheetMetalPunchFeatures(d.Features()).Add(&feature.SheetMetalPunchDefinition{Sketch: sk})
	d.Recompute()

	fp, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold: %v", err)
	}
	if len(fp.Punches) != 1 {
		t.Fatalf("flat punches = %d, want 1 (the base-plane punch)", len(fp.Punches))
	}
	if fp.Punches[0].Token != punch.Name() {
		t.Errorf("punch token = %q, want the feature name %q", fp.Punches[0].Token, punch.Name())
	}
	if len(fp.Punches[0].Outline) < 4 {
		t.Errorf("punch outline = %d points, want >= 4 (the square profile)", len(fp.Punches[0].Outline))
	}
}

// TestUnfoldEchoesPunchRepresentationAndAngle a punch's representation type reaches the flat punch
// result, and the die's rotation is added to the flat outline's own angle (#1968).
func TestUnfoldEchoesPunchRepresentationAndAngle(t *testing.T) {
	t.Parallel()
	flatPunch := func(angle float64) feature.FlatPunch {
		d, _ := sheetWithFlange(t)
		sk := d.Sketches().Add(sketch.XYPlane())
		sk.AddRectangleByCorners(gmath.P2(1, 1), gmath.P2(2, 2))
		feature.NewSheetMetalPunchFeatures(d.Features()).Add(&feature.SheetMetalPunchDefinition{
			Sketch: sk, Angle: func() float64 { return angle }, Representation: types.CentermarkPunchRepresentation,
		})
		d.Recompute()
		fp, err := d.Unfold()
		if err != nil {
			t.Fatalf("Unfold: %v", err)
		}
		if len(fp.Punches) != 1 {
			t.Fatalf("flat punches = %d, want 1", len(fp.Punches))
		}
		return fp.Punches[0]
	}
	base, turned := flatPunch(0), flatPunch(0.5)
	if turned.Representation != types.CentermarkPunchRepresentation {
		t.Errorf("flat punch representation = %v, want centermark", turned.Representation)
	}
	if math.Abs((turned.Angle-base.Angle)-0.5) > 1e-9 {
		t.Errorf("flat punch angle rose by %.5f, want the die's 0.5 rad", turned.Angle-base.Angle)
	}
}

// TestUnfoldDevelopsTabForOutOfPlaneFold a flange whose 3D fold direction leaves the base
// plane (a flange folded off a bottom edge folds in −Z) must still develop a tab in the base
// plane: the tab outward is derived from the base geometry, not the 3D fold vector. Regression
// for the bridge-found bug where such a tab collapsed (projected −Z → zero) and the flat was
// just the base.
func TestUnfoldDevelopsTabForOutOfPlaneFold(t *testing.T) {
	t.Parallel()
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	addRectFace(t, d, 4, 3)
	body := d.Features().Result()[0]
	edge := body.Edges()[0] // a bottom edge: its flange folds out of the base plane (−Z)
	pf := feature.NewSheetMetalFlangeFeatures(d.Features()).Add(&feature.SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Height: func() float64 { return 1 },
	})
	d.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("flange unhealthy: %s", pf.Health().Reason)
	}

	fp, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold: %v", err)
	}
	baseVol := 4 * 3 * d.SheetMetal().Thickness()
	if got := flatVolume(fp.Body); got <= baseVol*1.05 {
		t.Errorf("flat volume = %.4f, want clearly above the base %.4f (the tab must develop)", got, baseVol)
	}
}

// addRectFace adds a w×h rectangle base Face on XY and recomputes.
func addRectFace(t *testing.T, d *PartComponentDefinition, w, h float64) {
	t.Helper()
	sk := d.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(gmath.P2(0, 0))
	c1 := sk.Points().Add(gmath.P2(w, 0))
	c2 := sk.Points().Add(gmath.P2(w, h))
	c3 := sk.Points().Add(gmath.P2(0, h))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewSheetMetalFaceFeatures(d.Features()).Add(&feature.SheetMetalFaceDefinition{Sketch: sk, ProfileIndex: 0, Operation: ops.NewBody})
	d.Recompute()
}

// TestFlatBendDownForFlippedFlange a flipped flange (folds toward the back) marks its fold
// line bend-down in the flat; a plain flange marks it bend-up.
func TestFlatBendDownForFlippedFlange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		flip     bool
		wantDown bool
	}{{"plain", false, false}, {"flipped", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewPartComponentDefinition()
			if _, err := d.EnableSheetMetal(); err != nil {
				t.Fatalf("EnableSheetMetal: %v", err)
			}
			addSquareFace(d, 4)
			edge := topXEdge(t, d.Features().Result()[0])
			pf := feature.NewSheetMetalFlangeFeatures(d.Features()).Add(&feature.SheetMetalFlangeDefinition{
				EdgeKey: edge.ReferenceKey(), Height: func() float64 { return 1 }, Flip: tc.flip,
			})
			d.Recompute()
			if !pf.Health().OK() {
				t.Fatalf("flange unhealthy: %s", pf.Health().Reason)
			}
			fp, err := d.Unfold()
			if err != nil {
				t.Fatalf("Unfold: %v", err)
			}
			if len(fp.Bends) != 1 || fp.Bends[0].FoldDown != tc.wantDown {
				t.Errorf("bends = %+v, want one with FoldDown=%v", fp.Bends, tc.wantDown)
			}
		})
	}
}

// TestUnfoldTracksKFactor the flat is associative on the rule: raising the K-factor lengthens
// the bend allowance and so the developed extent, without recomputing the folded model.
func TestUnfoldTracksKFactor(t *testing.T) {
	t.Parallel()
	d, _ := sheetWithFlange(t)
	tight, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold (tight): %v", err)
	}
	d.SheetMetal().SetUnfold(sheetmetal.KFactorMethod(0.9)) // neutral axis nearer the outside
	loose, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold (loose): %v", err)
	}
	if !(flatVolume(loose.Body) > flatVolume(tight.Body)) {
		t.Errorf("looser K-factor flat volume %.5f should exceed %.5f", flatVolume(loose.Body), flatVolume(tight.Body))
	}
}

// TestUnfoldRejectsNonSheetMetal and a sheet-metal part with no base Face both error clearly.
func TestUnfoldRejectsNonSheetMetal(t *testing.T) {
	t.Parallel()
	plain := NewPartComponentDefinition()
	if _, err := plain.Unfold(); err == nil {
		t.Error("Unfold on a plain part must error")
	}
	empty := NewPartComponentDefinition()
	if _, err := empty.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	if _, err := empty.Unfold(); err == nil {
		t.Error("Unfold with no base Face must error")
	}
}
