// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/math"
)

// CodeCDTConstraintLeak marks a constrained triangulation where a boundary constraint edge could not be
// recovered (the flip cap hit, or a non-convex crossing stalled). Left unhandled the domain boundary
// leaks across the gap and floodInside mislabels inside/outside — a silent watertightness defect, the
// highest-priority bug class. The tessellator instead falls back to a deterministic boundary-only ear
// clip and records this Defect so the degradation is counted, never silent (#1410).
const CodeCDTConstraintLeak diag.Code = "tessellate.cdt-constraint-leak"

// earcutFromLoops triangulates the index-based loops (loops[0] is the outer boundary, the rest are
// holes) by deterministic ear clipping, returning triangles indexed back into the original pts space.
// It is the watertight fallback taken when constraint recovery failed and extractDomain would leak the
// domain (#1410): ear clipping respects every boundary edge by construction, so it cannot leak. It uses
// only the loop (boundary) points — any interior Steiner refinement is dropped on this rare failure
// path, trading curved-area accuracy for a guaranteed watertight result.
func earcutFromLoops(pts [][2]float64, loops [][]int) [][3]int {
	if len(loops) == 0 || len(loops[0]) < 3 {
		return nil
	}
	outer := loopPoints(pts, loops[0])
	holes := make([][]math.Point2, 0, len(loops)-1)
	for _, h := range loops[1:] {
		holes = append(holes, loopPoints(pts, h))
	}
	// earcut indexes its output into outer++holes in input order (it normalizes winding without
	// reordering vertices), so the loops flattened in the same order remap straight back into pts.
	remap := make([]int, 0, len(pts))
	for _, lp := range loops {
		remap = append(remap, lp...)
	}
	local := earcut(outer, holes)
	out := make([][3]int, len(local))
	for i, t := range local {
		out[i] = [3]int{remap[t[0]], remap[t[1]], remap[t[2]]}
	}
	return out
}

// loopPoints gathers a loop's points from pts by its index list, as the math.Point2 type earcut takes.
func loopPoints(pts [][2]float64, idx []int) []math.Point2 {
	out := make([]math.Point2, len(idx))
	for i, j := range idx {
		out[i] = math.P2(math.Scalar(pts[j][0]), math.Scalar(pts[j][1]))
	}
	return out
}

// domainLeaked reports whether the extracted domain bled across a constraint LOOP — a hole or concave
// notch that did not get cut because its boundary was not recovered, so the flood filled it. A loop edge
// kept on exactly one side is a clean domain boundary (use==1); shared by TWO kept triangles it is
// interior (use≥2), meaning that segment dissolved into the domain. A properly cut loop is (almost) all
// boundary; a FILLED loop is (almost) all interior. We flag a leak only when a loop is PREDOMINANTLY
// dissolved (interior edges outnumber boundary edges) — an isolated use≥2 edge is a local fold repairFolds
// later removes, not a missing cut, so a single fold must not discard the higher-quality refined mesh
// (#1410). use==0 (bordering the excluded super region, e.g. an unrecovered outer seam) is benign.
func (m *cdt) domainLeaked(tris [][3]int, loops [][]int) bool {
	use := make(map[[2]int]int, len(tris)*3)
	for _, t := range tris {
		for i := range 3 {
			use[conKey(t[i], t[(i+1)%3])]++
		}
	}
	rep := m.representatives()
	for _, lp := range loops {
		boundary, interior := 0, 0
		for k := range lp {
			switch e := conKey(rep[lp[k]], rep[lp[(k+1)%len(lp)]]); {
			case use[e] == 1:
				boundary++
			case use[e] >= 2:
				interior++
			}
		}
		if interior > boundary {
			return true
		}
	}
	return false
}

// recordConstraintLeak surfaces the recovery status on the mesh when a boundary constraint did not
// recover (#1410). A genuine leak — the domain bled across the gap and the deterministic earcut fallback
// replaced it — is a counted Defect; a non-recovery that stayed watertight (an outer seam into the
// excluded super region) is a benign Info, so the degradation is never silent yet the common harmless
// case does not raise a false alarm.
func recordConstraintLeak(m *Mesh, unrecovered [][2]int, leaked bool) {
	if m == nil || len(unrecovered) == 0 {
		return
	}
	if leaked {
		m.Diagnose(diag.Diagnostic{
			Code:     CodeCDTConstraintLeak,
			Severity: diag.Defect,
			Detail:   fmt.Sprintf("%d boundary constraint(s) unrecovered and the domain leaked; used deterministic ear-clip fallback to stay watertight", len(unrecovered)),
		})
		return
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeCDTConstraintLeak,
		Severity: diag.Info,
		Detail:   fmt.Sprintf("%d boundary constraint(s) unrecovered but the cut stayed watertight; corridor-walk recovery (#1409) would realize them", len(unrecovered)),
	})
}
