// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestWidthSamplesEndpointsAreExactVertices locks the root-cause fix for the G2 "radius exceeds
// geometric maximum 0" family: the two endpoint samples must be the STORED vertices, not
// A+(B−A). Interpolating reintroduces ~u_mach·|coords| of roundoff, which at large STEP
// coordinates turns a shared-corner crossing from exactly 0 into ~1e-14, slipping past the x≤0
// guard and collapsing the in-face width to ~0 (geometry-math-advisor derivation).
func TestWidthSamplesEndpointsAreExactVertices(t *testing.T) {
	t.Parallel()
	e := straightTestEdge(t, math.P3(34.2, 94, 50), math.P3(-0.612, 86, 59.7))
	samples := widthSamples(e)
	if len(samples) != 5 {
		t.Fatalf("widthSamples returned %d samples, want 5", len(samples))
	}
	if samples[0].point != e.StartVertex().Point() || samples[0].corner == nil {
		t.Errorf("sample[0] must be the exact start vertex with corner set, got %v corner=%v",
			samples[0].point, samples[0].corner)
	}
	if samples[4].point != e.EndVertex().Point() || samples[4].corner == nil {
		t.Errorf("sample[4] must be the exact end vertex with corner set, got %v corner=%v",
			samples[4].point, samples[4].corner)
	}
	for i := 1; i < 4; i++ {
		if samples[i].corner != nil {
			t.Errorf("interior sample[%d] must have nil corner (no incident-edge exclusion), got %v",
				i, samples[i].corner)
		}
	}
}

// TestSharesVertex covers the incident-edge test that excludes, at an endpoint sample, only the
// boundary edges meeting the fillet edge at that corner — by vertex identity, so coordinate
// roundoff cannot hide the shared corner.
func TestSharesVertex(t *testing.T) {
	t.Parallel()
	e := straightTestEdge(t, math.P3(0, 0, 0), math.P3(10, 0, 0))
	if !sharesVertex(e, e.StartVertex()) || !sharesVertex(e, e.EndVertex()) {
		t.Errorf("edge must share both its own endpoint vertices")
	}
	if sharesVertex(e, nil) {
		t.Errorf("nil vertex (interior sample) must never match")
	}
}

// TestMaxRadiusNoEndpointPhantom is the corpus-grounded regression: on simple/V3 (a STEP-imported
// skew prism at ~100-unit coordinates that OCCT fillets at r=5), NO planar edge may report a
// near-zero geometric maximum. Before the fix the endpoint phantom drove the binding edge's
// r_max to ~2.8e-15, so every real radius was rejected ("exceeds geometric maximum 0"); r_max is
// pure local geometry (independent of the pick radius), and V3's tightest planar edge admits
// r_max ≈ 7.2, so a floor of 1.0 cleanly separates "fixed" from "phantom".
func TestMaxRadiusNoEndpointPhantom(t *testing.T) {
	t.Parallel()
	body := importMaxWidthFixture(t, "v3_maxwidth.step")
	minRMax := stdmath.Inf(1)
	for _, e := range body.Edges() {
		a, b, nA, nB, err := edgePlanarFaces(e)
		if err != nil {
			continue // not a two-planar-face edge
		}
		pick := filletPick{edge: e, r0: 5, r1: 5}
		if rMax, _, _, ok := maxFilletRadius(pick, a, b, nA, nB, []filletPick{pick}, false); ok {
			minRMax = stdmath.Min(minRMax, rMax)
		}
	}
	if stdmath.IsInf(minRMax, 1) {
		t.Fatal("V3 has no two-planar-face edges — fixture changed")
	}
	if minRMax < 1.0 {
		t.Fatalf("smallest planar-edge max radius %.6g < 1.0 — the endpoint phantom is back", minRMax)
	}
}

// TestGrazingIncidentArcNoCollapse is the N5 regression, driven through the real validity gate
// (validateFilletRadii) with N5's two co-picked, vertex-sharing edges so the concave-fill branch
// and the neighbour interaction match production. One picked edge's planar face continues from the
// fillet edge into a near-straight ARC tangent at their shared vertex; that same-side continuation
// grazes the recession ray at ~1e-10 along the whole span — a genuine tangency, distinct from the
// endpoint roundoff phantom — and before the incident-edge graze floor it collapsed the in-face
// width and r_max to ~4.9e-11, rejecting the r=5 pick with "exceeds geometric maximum". OCCT rounds
// these edges, so the gate must not reject them.
func TestGrazingIncidentArcNoCollapse(t *testing.T) {
	t.Parallel()
	body := importMaxWidthFixture(t, "n5_graze.step")
	e14 := edgeByEndpoints(t, body, math.P3(115.845593, 81.115958, 50), math.P3(112.372630, 61.419800, 50))
	e24 := edgeByEndpoints(t, body, math.P3(96.149438, 84.588921, 50), math.P3(115.845593, 81.115958, 50))
	picks := []filletPick{{edge: e14, r0: 5, r1: 5}, {edge: e24, r0: 5, r1: 5}}
	if err := validateFilletRadii(picks, FillConcaveOutward); err != nil {
		t.Fatalf("validateFilletRadii rejected N5's r=5 picks — the tangent same-side arc grazing collapse is back: %v", err)
	}
}

// edgeByEndpoints finds the body edge whose two vertices match p and q (either orientation).
func edgeByEndpoints(t *testing.T, b *topo.Body, p, q math.Point3) *topo.Edge {
	t.Helper()
	const tol = math.Scalar(1e-3)
	for _, e := range b.Edges() {
		s, u := e.StartVertex().Point(), e.EndVertex().Point()
		if (s.DistanceTo(p) < tol && u.DistanceTo(q) < tol) || (s.DistanceTo(q) < tol && u.DistanceTo(p) < tol) {
			return e
		}
	}
	t.Fatalf("no edge with endpoints %v and %v", p, q)
	return nil
}

func importMaxWidthFixture(t *testing.T, name string) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import %s: %v (bodies=%d)", name, err, len(bodies))
	}
	return bodies[0]
}

// straightTestEdge builds a standalone line-segment edge between p and q for the sampling unit
// tests (no face/loop needed).
func straightTestEdge(t *testing.T, p, q math.Point3) *topo.Edge {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("t", "body", 0)))
	a := bld.AddVertex(p, topo.NewLineage(topo.Tok("t", "vertex", 0)))
	b := bld.AddVertex(q, topo.NewLineage(topo.Tok("t", "vertex", 1)))
	return bld.AddEdge(geom.NewLineSegment(p, q), a, b, topo.NewLineage(topo.Tok("t", "edge", 0)))
}
