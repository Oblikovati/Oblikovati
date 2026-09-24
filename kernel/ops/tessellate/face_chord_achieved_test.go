// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// The faces #3517 left WORSE than the mesher it deleted, pinned per face and two-sided — and the report
// that stops them being silent (review 4 C1).
//
// Round 5's headline said "no face is worse than it was before #3517" while the table beside it showed
// J5 f00 at 0.90 → 1.4649 and K2 f03 at 0.86 → 3.6320. The sentence was written from intent and the
// table from measurement, and the table was right. A third, RODB∩ f01, was disclosed only at
// DefaultQuality (0.1614×, inside tolerance) and not at PropertyQuality (0.1225× → 1.3747×), which is
// the faceting mass properties read.
//
// Each row pins the ratio TWO-SIDED, so the face cannot drift further and the pin retires itself the
// day someone fixes it, and asserts the face NAMES its own limit. J5 f00 and K2 f03 are occtparity
// corpus bodies, which this package cannot reach (model/feature depends on kernel/ops, not the
// reverse), so they are pinned beside their own corpus — see occtparity's
// TestTheFacesLeftOverToleranceNameIt.

// TestRodBallIntersectionWallIsOverToleranceAndSaysSo pins RODB∩ f01: the rod WALL of rod ∩ ball, a
// cylinder whose trim develops into one (u,v) branch and which the one-path dispatch therefore moved
// from the covering onto the structured grid. That move is right — it is what takes W8 f02 from 36.87×
// to 0.75× — and on this face it costs, because the grid chords flat across a lens 0.1 mm deep. The
// body's PropertyQuality volume deficit doubles with it, 0.654 % → 1.447 %.
func TestRodBallIntersectionWallIsOverToleranceAndSaysSo(t *testing.T) {
	t.Parallel()
	f := rodBall(t, ops.Intersect).Faces()[1]
	if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
		t.Fatalf("RODB∩ face 1 is a %T, not the rod wall this row reads", f.Geometry())
	}
	assertChordRatioPinned(t, "RODB∩ f01", f, tessellate.PropertyQuality(), 1.3747)
	// Inside tolerance at the display faceting, which is why only the PropertyQuality half regressed.
	assertChordRatioPinned(t, "RODB∩ f01 display", f, ops.DefaultQuality(), 0.1614)
}

// assertChordRatioPinned holds one face's achieved chord to a measured multiple of the tolerance it was
// asked for, both ways, and requires the face to NAME the miss whenever it is over.
func assertChordRatioPinned(t *testing.T, name string, f *topo.Face, q tessellate.Quality, want float64) {
	t.Helper()
	m := tessellate.TessellateFace(f, q)
	if m == nil || m.TriangleCount() == 0 {
		t.Fatalf("%s meshed to nothing", name)
	}
	got := tessellate.WorstEdgeChord(m, f.Geometry()) / q.Tol()
	if stdmath.Abs(got-want) > 0.005 {
		t.Errorf("%s achieves %.4f× its chord tolerance, pinned at %.4f ± 0.005. If it IMPROVED, delete "+
			"this row — it exists only to stop a known-coarse face drifting further", name, got, want)
	}
	if got <= 1 {
		return // fixed, and the row above has already said so
	}
	if !meshNamesCode(m, tessellate.CodeFaceChordNotMet) {
		t.Errorf("%s is %.4f× over its chord tolerance and says nothing; a coarse face that ships must "+
			"name its own limit", name, got)
	}
}

// meshNamesCode reports whether a mesh recorded the given diagnostic code.
func meshNamesCode(m *tessellate.Mesh, code diag.Code) bool {
	for _, d := range m.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}
