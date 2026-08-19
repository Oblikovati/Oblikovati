// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
)

// Corner-seam lap / butt SOLID geometry (#2085). These drive the two-flange open corner the seam
// finishes, and MEASURE the result — a butt fill that closes the corner, a proud lap tab that grows
// with the overlap percent, and the two lap styles differing by which wall is lapped — because a
// plausible watertight solid is not enough (the tranche's measured-check rule).

// twoFlangeCorner folds two flanges on adjacent edges of a square sheet, leaving the corner between
// them OPEN (no auto-miter), and returns the engine plus the walls' vertical corner edges — the
// fixture the corner-seam lap/butt styles finish. Separate radii give the two walls different stand-
// offs so overlap and reverse-overlap lap by different amounts and can be told apart by volume.
func twoFlangeCorner(t *testing.T, radiusX, radiusY float64) (*PartFeatures, []*topo.Edge) {
	t.Helper()
	fs, edgeX := seedSheetMetalSheet(t, 4, nil)
	fs.SetReliefSpec(func() ReliefSpec { return ReliefSpec{} })
	fs.SetCornerReliefSpec(func() CornerReliefSpec { return CornerReliefSpec{Shape: types.CornerTear} })
	flanges := NewSheetMetalFlangeFeatures(fs)
	flanges.Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeX.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(radiusX),
	})
	fs.Recompute()
	edgeY := adjacentTopEdge(t, fs.Result()[0], edgeX)
	flanges.Add(&SheetMetalFlangeDefinition{
		EdgeKey: edgeY.ReferenceKey(), Height: constClosure(1.0), Radius: constClosure(radiusY),
	})
	fs.Recompute()
	return fs, verticalCornerEdges(t, fs.Result()[0])
}

// addCornerSeam attaches a corner seam of the given style to the engine and recomputes.
func addCornerSeam(t *testing.T, fs *PartFeatures, edges []*topo.Edge, seam SeamType, overlap float64) *PartFeature {
	t.Helper()
	pf := NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: edgeKeys(edges), Gap: constClosure(0.05), Type: seam, Overlap: overlap,
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("%v corner seam sick: %+v", seam, pf.Health())
	}
	return pf
}

// edgeKeys collects the reference keys of a set of edges.
func edgeKeys(edges []*topo.Edge) [][]byte {
	keys := make([][]byte, len(edges))
	for i, e := range edges {
		keys[i] = e.ReferenceKey()
	}
	return keys
}

// TestCornerSeamNoOverlapButtsTheWalls: the no-overlap style carries both walls past the corner and
// butts them into a watertight solid — material the open corner did not have — and no longer reports
// the deferral, because the corner is now modelled.
func TestCornerSeamNoOverlapButtsTheWalls(t *testing.T) {
	fs, edges := twoFlangeCorner(t, 0.3, 0.3)
	openVol := smSolidVolume(fs.Result()[0])
	pf := addCornerSeam(t, fs, edges, NoOverlapSeam, 0)
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	if hasDiagCode(pf.Diagnostics(), codeCornerSeamUnmodeled) {
		t.Errorf("no-overlap seam is modelled now and must not report %q: %v", codeCornerSeamUnmodeled, pf.Diagnostics())
	}
	if got := smSolidVolume(body); !(got > openVol) {
		t.Errorf("no-overlap butt volume %.4f did not fill the open corner %.4f", got, openVol)
	}
}

// TestCornerSeamOverlapAddsProudTab: the overlap style adds a proud tab on top of the no-overlap
// butt, so its volume is strictly greater, and the tab grows with the overlap percent. The added
// material is measured against the analytic tab thickness × lapLength × height.
func TestCornerSeamOverlapAddsProudTab(t *testing.T) {
	butt := seamVolume(t, NoOverlapSeam, 0, 0.3, 0.3)
	lap40 := seamVolume(t, OverlapSeam, 40, 0.3, 0.3)
	lap80 := seamVolume(t, OverlapSeam, 80, 0.3, 0.3)
	if !(lap40 > butt) {
		t.Errorf("overlap(40%%) %.4f is not proud of the butt %.4f — no lap was added", lap40, butt)
	}
	if !(lap80 > lap40) {
		t.Errorf("lap is not monotone: overlap(80%%) %.4f <= overlap(40%%) %.4f", lap80, lap40)
	}
	// Analytic proud tab = thickness 0.2 × lapLength (percent·standOff, standOff=radius+thickness=0.5)
	// × the flat-face height 1.0. The lap must match it, and halving the percent must halve the tab.
	const t0, standOff, height = 0.2, 0.5, 1.0
	tab := func(pct float64) float64 { return t0 * (pct / 100 * standOff) * height }
	if got := lap80 - butt; stdmath.Abs(got-tab(80)) > 0.008 {
		t.Errorf("overlap(80%%) added %.4f cm³, want the analytic tab %.4f", got, tab(80))
	}
	if got := lap40 - butt; stdmath.Abs(got-tab(40)) > 0.008 {
		t.Errorf("overlap(40%%) added %.4f cm³, want the analytic tab %.4f", got, tab(40))
	}
}

// TestCornerSeamOverlapVsReverseLapDifferentWalls: overlap laps the first wall over the second and
// reverse-overlap laps the other way. With the two walls at different radii the lapped wall's stand-
// off differs, so the two styles add different tabs — proof the choice of lapping wall is honoured,
// not just recorded.
func TestCornerSeamOverlapVsReverseLapDifferentWalls(t *testing.T) {
	// Wall a (radius 0.3, stand-off 0.5) is placed first, wall b (radius 0.6, stand-off 0.8) second.
	// Overlap laps a over b — the tab sits on b's wider face; reverse laps b over a — the smaller
	// tab sits on a. So overlap must add strictly MORE than reverse, by the stand-off difference.
	over := seamVolume(t, OverlapSeam, 80, 0.3, 0.6)
	reverse := seamVolume(t, ReverseOverlapSeam, 80, 0.3, 0.6)
	if !(over-reverse > 0.02) {
		t.Errorf("overlap %.4f is not proud of reverse-overlap %.4f by the lapped-wall stand-off — "+
			"the lapping wall was ignored", over, reverse)
	}
}

// TestCornerSeamRootReliefIsCut: a sized seam-root relief opens the corner where the two bends meet,
// so a relieved overlap seam has less material than the same seam with no relief.
func TestCornerSeamRootReliefIsCut(t *testing.T) {
	noRelief := seamVolume(t, OverlapSeam, 80, 0.3, 0.3)
	fs, edges := twoFlangeCorner(t, 0.3, 0.3)
	pf := NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: edgeKeys(edges), Gap: constClosure(0.05), Type: OverlapSeam, Overlap: 80,
		ReliefShape: types.CornerSquare, ReliefSize: constClosure(0.2),
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("relieved overlap seam sick: %+v", pf.Health())
	}
	relieved := smSolidVolume(fs.Result()[0])
	if !(relieved < noRelief) {
		t.Errorf("seam-root relief removed nothing: relieved %.4f vs unrelieved %.4f", relieved, noRelief)
	}
}

// seamVolume builds a two-flange corner, finishes it with the given seam, and returns the solid's
// volume — the shared measurement for the lap/butt tests.
func seamVolume(t *testing.T, seam SeamType, overlap, radiusX, radiusY float64) float64 {
	t.Helper()
	fs, edges := twoFlangeCorner(t, radiusX, radiusY)
	addCornerSeam(t, fs, edges, seam, overlap)
	return smSolidVolume(fs.Result()[0])
}
