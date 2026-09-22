// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
)

// The two corpus faces #3517 left WORSE than the mesher it deleted (review 4 C1).
//
// Round 5's headline said "no face is worse than it was before #3517" while the table beside it read
// 0.90 → 1.4649 for J5 f00 and 0.86 → 3.6320 for K2 f03. The sentence was written from intent, the
// table from measurement, and the table was right. Both faces are the general chart-driven mesher's:
// their trims do NOT develop into one (u,v) branch, so the covering is correctly theirs, and on these
// two it is coarser at the rim than the path that used to take them.
//
// Both are pinned TWO-SIDED so they cannot drift further and so the pin retires itself the day someone
// fixes them, and both must NAME the miss: a coarse face that ships silently is the defect this round
// removed, and one that names its limit is a limit a reader can act on.
//
// THE RESIDUAL IS NOT REPAIRABLE INSIDE THIS MESHER, and that is why these are pins rather than a fix.
// The clearance cap that recovered the other five trades K2 f03's rim against the genus-1 complement's
// existence: at cap 0.25 K2 f03 returns to the arm's 0.8563 and the complement is DECLINED entirely
// (294.428 mm², 28 free edges). No value inside the admissible window [0.75, 1.0] recovers J5 f00 at
// all — it reads 1.4649 at every one of them. What closes the gap is the facet count, which is a
// property of the shared curve discretizer (ADR-0061 §R4.5), not of this mesher.
func TestTheFacesLeftOverToleranceNameIt(t *testing.T) {
	t.Parallel()
	for _, row := range []struct {
		grid, name string
		face       int
		want       float64
	}{
		{"", "J5", 0, 1.4649},
		{"", "K2", 3, 3.6320},
	} {
		f := pinnedBody(t, row.grid, row.name).Faces()[row.face]
		q := ops.PropertyQuality()
		m := tessellate.TessellateFace(f, q)
		if m == nil || m.TriangleCount() == 0 {
			t.Fatalf("%s f%02d meshed to nothing", row.name, row.face)
		}
		got := worstChordRatio(m, f.Geometry(), q)
		if stdmath.Abs(got-row.want) > 0.005 {
			t.Errorf("%s f%02d achieves %.4f× its chord tolerance, pinned at %.4f ± 0.005. If it "+
				"IMPROVED, delete this row — it exists only to stop a known-coarse face drifting further",
				row.name, row.face, got, row.want)
		}
		if got > 1 && !chordMissNamed(m) {
			t.Errorf("%s f%02d is %.4f× over its chord tolerance and says nothing; a coarse face that "+
				"ships must name its own limit", row.name, row.face, got)
		}
	}
}

// chordMissNamed reports whether the mesh names its own chord miss.
func chordMissNamed(m *tessellate.Mesh) bool {
	for _, d := range m.Diagnostics {
		if d.Code == tessellate.CodeFaceChordNotMet && d.Severity == diag.Defect {
			return true
		}
	}
	return false
}
