// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Audit A9 (#1605): the terminal ear-clip patch fallback must never ship a defective covering
// silently — acceptance-check every result, recover once through the CDT (#1604), and flag any
// residual defect loudly.

// planarBoundaryPatch runs boundaryPatchMesh on a z=0 polygon over an actual plane surface.
func planarBoundaryPatch(t *testing.T, poly []math.Point2) *Mesh {
	t.Helper()
	s, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	outer3D := make([]math.Point3, len(poly))
	for i, p := range poly {
		outer3D[i] = math.P3(p.X, p.Y, 0)
	}
	return boundaryPatchMesh(s, outer3D, nil)
}

// patchCoverageDefects counts CodePatchCoverage Defect diagnostics on the mesh.
func patchCoverageDefects(m *Mesh) int {
	n := 0
	for _, d := range m.Diagnostics {
		if d.Code == CodePatchCoverage && d.Severity == diag.Defect {
			n++
		}
	}
	return n
}

// TestBoundaryPatchSelfTouchRecoveredByCDT pins the recovery tier (#1605): a self-touching trim —
// a vertex lying exactly ON another boundary edge — defeats ear clipping into count-complete but
// OVERLAPPING triangles (measured 24 for an 8-area region, a silent 3× cover). The coverage gate
// must detect the area mismatch and retriangulate through the CDT, whose split-at-vertex recovery
// (#1604) handles exactly this: correct area, watertight free-edge budget, no defect flag.
func TestBoundaryPatchSelfTouchRecoveredByCDT(t *testing.T) {
	t.Parallel()
	poly := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 4), math.P2(2, 0), math.P2(0, 4)}
	m := planarBoundaryPatch(t, poly)
	const want = 8.0 // |shoelace| of the self-touching pentagon
	if got := m.Area(); stdmath.Abs(got-want) > 1e-9*want {
		t.Errorf("self-touching patch area = %.6f, want %.6f (ear-clip overlap not recovered)", got, want)
	}
	if free := WeldedFreeEdgeCount(m); free > len(poly)+2 {
		// +2: the split at the on-edge vertex doubles that boundary station into two edges.
		t.Errorf("self-touching patch has %d free edges, want ≤ %d", free, len(poly)+2)
	}
	if n := patchCoverageDefects(m); n != 0 {
		t.Errorf("recovered patch must carry no %s defect, got %d", CodePatchCoverage, n)
	}
}

// TestBoundaryPatchSelfCrossingShipsLoudly pins the loud-failure guarantee (#1605): a genuinely
// self-crossing trim (bow-tie — winding area zero, ear-clip emits overlapping garbage) cannot be
// covered consistently by ANY triangulation of its loops, so whatever ships must carry the
// CodePatchCoverage Defect — visible in feature diagnostics and over the wire — never a silent
// return.
func TestBoundaryPatchSelfCrossingShipsLoudly(t *testing.T) {
	t.Parallel()
	poly := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(0, 4), math.P2(4, 4)}
	m := planarBoundaryPatch(t, poly)
	if n := patchCoverageDefects(m); n == 0 {
		t.Errorf("self-crossing trim shipped without a %s defect (silent degradation)", CodePatchCoverage)
	}
}

// TestBoundaryPatchCleanTrimUnflagged guards against false positives: an ordinary concave simple
// polygon ear-clips exactly and must ship with the exact area and no coverage defect.
func TestBoundaryPatchCleanTrimUnflagged(t *testing.T) {
	t.Parallel()
	poly := []math.Point2{math.P2(0, 0), math.P2(6, 0), math.P2(6, 2), math.P2(2, 2), math.P2(2, 4), math.P2(0, 4)}
	m := planarBoundaryPatch(t, poly)
	const want = 16.0 // L-shape: 6×2 + 2×2
	if got := m.Area(); stdmath.Abs(got-want) > 1e-9*want {
		t.Errorf("clean trim area = %.6f, want %.6f", got, want)
	}
	if n := patchCoverageDefects(m); n != 0 {
		t.Errorf("clean trim must carry no %s defect, got %d", CodePatchCoverage, n)
	}
}
