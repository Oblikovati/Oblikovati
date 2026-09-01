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

// Auto-miter (#1961). Two walls folding away from one corner each stop at their own bend line, so
// the corner between them is OPEN — measuring the flat pattern of an L-bracket shows the two tabs
// meeting at a point with the square beyond them empty. Mitering fills that corner and cuts a gap
// on the bisector.

// miteredCorner folds two flanges on adjacent edges, mitering the second onto the first.
func miteredCorner(t *testing.T, miter bool, gap float64) (*topo.Body, *PartFeatures) {
	t.Helper()
	fs, edgeX := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} })
	fs.SetCornerReliefSpec(func() CornerReliefSpec { return CornerReliefSpec{Shape: types.CornerTear} })
	flanges := NewSheetMetalFlangeFeatures(fs)
	flanges.Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeX.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	edgeY := adjacentTopEdge(t, fs.Result()[0], edgeX)
	def := &SheetMetalFlangeDefinition{
		EdgeKey: edgeY.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
		AutoMiter: miter,
	}
	if gap > 0 {
		def.MiterGap = constClosure(gap)
	}
	second := flanges.Add(def)
	fs.Recompute()
	if !second.Health().OK() {
		t.Fatalf("mitered flange sick: %+v", second.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("mitered part is not a valid solid: %+v", r)
	}
	return body, fs
}

// TestAutoMiterFillsTheOpenCorner: without mitering the corner between two walls is empty. The
// miter carries each wall past the corner until it meets the other, which is material the part did
// not have — so the volume rises, and it rises by the corner the two walls now share.
func TestAutoMiterFillsTheOpenCorner(t *testing.T) {
	t.Parallel()
	open, _ := miteredCorner(t, false, 0)
	mitered, _ := miteredCorner(t, true, 0)
	openVol := smSolidVolume(open)
	miteredVol := smSolidVolume(mitered)
	if !(miteredVol > openVol) {
		t.Fatalf("mitered volume %.4f is not more than the open corner's %.4f — nothing was filled",
			miteredVol, openVol)
	}
	// Each wall carries its section (0.2 thick, 1.3 tall over the bend and run) past the corner by
	// the other's stand-off; the two extensions overlap in the corner column, which the union
	// resolves. The fill is therefore well under twice one extension and well over zero.
	added := miteredVol - openVol
	if added < 0.05 || added > 0.6 {
		t.Errorf("miter added %.4f cm³, want a corner-sized fill (0.05–0.6)", added)
	}
}

// TestAutoMiterGapSeparatesTheWalls: the two extensions meet on the bisector, and without a cut
// there they occupy the same corner. The gap is what lets the part fold.
func TestAutoMiterGapSeparatesTheWalls(t *testing.T) {
	t.Parallel()
	solid, _ := miteredCorner(t, true, 0)
	slotted, _ := miteredCorner(t, true, 0.1)
	if !(smSolidVolume(slotted) < smSolidVolume(solid)) {
		t.Errorf("the miter gap removed nothing: %.4f vs %.4f",
			smSolidVolume(slotted), smSolidVolume(solid))
	}
}

// TestNoMiterLeavesTheCornerAlone: mitering is opt-in, so an existing part's corners are built
// exactly as they were before the option existed.
func TestNoMiterLeavesTheCornerAlone(t *testing.T) {
	t.Parallel()
	plain, _ := miteredCorner(t, false, 0)
	// The same part built with the miter switched off must match a part that never knew about it.
	fs, edgeX := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} })
	fs.SetCornerReliefSpec(func() CornerReliefSpec { return CornerReliefSpec{Shape: types.CornerTear} })
	flanges := NewSheetMetalFlangeFeatures(fs)
	flanges.Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeX.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	edgeY := adjacentTopEdge(t, fs.Result()[0], edgeX)
	flanges.Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeY.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
	})
	fs.Recompute()
	if got, want := smSolidVolume(fs.Result()[0]), smSolidVolume(plain); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("an unmitered corner = %.6f, want %.6f — the option changed a part that did not ask", got, want)
	}
}

// TestMiterNeedsACorner: a lone flange has nothing to miter to, so asking for one is a no-op
// rather than an error or a stray extension hanging off the part.
func TestMiterNeedsACorner(t *testing.T) {
	t.Parallel()
	build := func(miter bool) float64 {
		fs, edge := seedSheetMetalSheet(t, 4, nil)
		fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} })
		pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
			EdgeKey: edge.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(0.3),
			AutoMiter: miter,
		})
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("flange sick: %+v", pf.Health())
		}
		return smSolidVolume(fs.Result()[0])
	}
	if got, want := build(true), build(false); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("a lone mitered flange = %.6f, want the unmitered %.6f", got, want)
	}
}

// TestWallStandOffReadsTheSection: the miter's reach is each wall's stand-off from its own bend
// line, read off the band it was built from. A 90° bend at radius r in gauge t stands r+t away, so
// a change to the section has to move the miter with it.
func TestWallStandOffReadsTheSection(t *testing.T) {
	t.Parallel()
	x, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	z, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	bend := BendPlacement{
		Outward: x, Up: z,
		Angle: stdmath.Pi / 2, Radius: 0.3, Thickness: 0.2, Length: 1.0,
	}
	if got, want := wallStandOff(bend), 0.5; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("stand-off = %.4f, want %.4f (radius + thickness)", got, want)
	}
	bend.Radius = 0.8
	if got, want := wallStandOff(bend), 1.0; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("stand-off at a bigger radius = %.4f, want %.4f", got, want)
	}
}
