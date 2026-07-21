// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// dihedralOrthoCosTol is the largest |cos θ| between two miter edge axes still treated as an
// orthogonal (θ=90°) corner. Both axes are unit vectors, so this is a DIMENSIONLESS,
// scale-invariant cutoff (unlike a length tolerance it needs no model-scale factor). Only the
// orthogonal setback distance s = r·cot(45°) = r is derived here; a non-orthogonal miter
// (s = r·cot(θ/2), OCCT case L7) is a separate slice and must NOT fire this pass.
const dihedralOrthoCosTol = 1e-9

// applyCornerSetback re-solves every CONCAVE ORTHOGONAL PLANAR dihedral-miter corner to OCCT's
// set-back geometry and returns a fresh fils slice, ok=true when at least one such corner fired.
//
// The gap it closes (OCCT tests/blend/simple L1): a concave miter's seam is solved by miterArmOf
// with the CONVEX rolling-ball offset (v + offDir·r), so the two arms' seam lands on the REFLECTED
// cylinder — the shared-face (outer) rail is retracted to [+r,L−r] and the wall rail sits r below
// the shared plane, over-keeping the +3200 of host-plane material the derivation measured. The
// edge's own ef.cyl is already the correct concave-side cylinder (built from the flipped frame), so
// re-sampling the seam on ef.cyl with the concave-negated face normals (concaveMiterSeam) snaps both
// rails to OCCT's stations: the shared-face rail EXTENDS +r past the raw endpoint (v∈[−r,L+r]) and
// the wall rail stays flush (v∈[0,L]). Because the band rail (cylinderFace), the host-plane re-trim
// (filletMaps → transformFace) and the seam end all read the SAME corner.ta/tb/seam, rewriting them
// once moves every consumer together — the single-source coupling applyRunoutSetback already relies
// on. There is NO corner patch for a dihedral corner (OCCT L1 = 15 faces); the two bands mutually
// miter along the shared seam.
//
// It never mutates the caller's fils (it copies), so a decline (ok=false) or a downstream certify
// failure keeps the baseline body byte-identical. The gate (concaveOrthogonalDihedralMiter) excludes
// every convex miter (already correct), curved-contact miter, variable fillet, non-orthogonal miter
// (L7-skew/A8, a later slice) and trihedral blend.
//
// SCOPE OF CHANGE (not byte-identical for the whole corpus — the survivor-arc lesson): the pass fires
// for EVERY orthogonal-concave-planar-dihedral-miter config, so besides greening the RED cases L1/L7 it
// also RE-WELDS one previously-GREEN case — N5 (a rotated boss of the same family, green only by
// tolerance at base). That re-weld is a legitimate improvement (N5's area moves TOWARD OCCT, 0.3337%→
// 0.1043%, and it stays watertight); it is NEVER a regression because adoptCornerSetback keeps the
// baseline unless the set-back body certifies watertight+hole-contained+solid. The full changed-body set
// across all six grids is exactly {L1, L7, N5}, each pinned in fingerprint_pins_test.go.
func applyCornerSetback(fils []edgeFillet, miters map[uint64]*cornerMiter) ([]edgeFillet, bool) {
	ends := miterCornerEnds(fils)
	out := append([]edgeFillet(nil), fils...)
	fired := false
	for vid, cm := range miters {
		pair := ends[vid]
		if len(pair) != 2 {
			continue // a miter corner is shared by exactly two filleted edges
		}
		if resetbackCorner(out, pair, cm) {
			fired = true
		}
	}
	return out, fired
}

// miterEnd locates one filleted edge's miter corner: its index in fils and which end (c1 or c0).
type miterEnd struct {
	fi   int
	atC1 bool
}

// miterCornerEnds indexes every filleted edge's miter corners by the corner vertex id, so the two
// edges meeting at a miter vertex can be paired.
func miterCornerEnds(fils []edgeFillet) map[uint64][]miterEnd {
	m := map[uint64][]miterEnd{}
	for i := range fils {
		if fils[i].c0.miter && fils[i].c0.vertex != nil {
			m[fils[i].c0.vertex.ID()] = append(m[fils[i].c0.vertex.ID()], miterEnd{fi: i, atC1: false})
		}
		if fils[i].c1.miter && fils[i].c1.vertex != nil {
			m[fils[i].c1.vertex.ID()] = append(m[fils[i].c1.vertex.ID()], miterEnd{fi: i, atC1: true})
		}
	}
	return m
}

// resetbackCorner re-solves the concave dihedral miter shared by the two edges in pair and rewrites
// both corners' ta/tb/seam to the set-back seam. Returns false (leaving both corners untouched) when
// the gate rejects the corner or the seam cannot be sampled.
func resetbackCorner(fils []edgeFillet, pair []miterEnd, cm *cornerMiter) bool {
	efA, efB := &fils[pair[0].fi], &fils[pair[1].fi]
	if !concaveOrthogonalDihedralMiter(efA, efB, cm) {
		return false
	}
	seam, ok := concaveMiterSeam(*efA, *efB, cm)
	if !ok {
		return false
	}
	writeMiterEnd(&fils[pair[0].fi], pair[0].atC1, cm.shared, seam)
	writeMiterEnd(&fils[pair[1].fi], pair[1].atC1, cm.shared, seam)
	return true
}

// concaveOrthogonalDihedralMiter reports whether the two edges form the L1-class corner this pass
// owns: a planar (non-curved) miter whose two arms are CONCAVE fills meeting at an orthogonal
// (θ=90°) edge pair. Every other config (convex miter, curved-contact miter, variable fillet,
// non-orthogonal miter) is excluded so it stays byte-identical.
func concaveOrthogonalDihedralMiter(efA, efB *edgeFillet, cm *cornerMiter) bool {
	if cm == nil || cm.curved != nil || cm.shared == nil {
		return false
	}
	if !efA.flip || !efB.flip || efA.varying || efB.varying {
		return false
	}
	if _, ok := cm.shared.Geometry().(geom.Plane); !ok {
		return false
	}
	cos := unit(efA.cyl.AxisDir.AsVector()).Dot(unit(efB.cyl.AxisDir.AsVector()))
	return stdmath.Abs(cos) < dihedralOrthoCosTol
}

// concaveMiterSeam re-samples the miter seam on the two edges' CORRECT concave-side cylinders
// (ef.cyl, already built from the flipped frame) using the concave-negated shared/outer face
// normals, so seam[0] (sTop) lands on the shared face EXTENDED +r past the raw endpoint and the last
// sample (sBot) lands flush on the wall. It is the concave analogue of solveMiter's sampleMiterSeam
// call, differing only in the negated normals and the pre-solved arm centres.
func concaveMiterSeam(efA, efB edgeFillet, cm *cornerMiter) ([]math.Point3, bool) {
	nS, okS := planeNormal(cm.shared)
	outerA, outerB := otherFace(efA.edge, cm.shared), otherFace(efB.edge, cm.shared)
	if outerA == nil || outerB == nil {
		return nil, false
	}
	nFA, okA := planeNormal(outerA)
	nFB, okB := planeNormal(outerB)
	if !okS || !okA || !okB {
		return nil, false
	}
	a1 := miterArm{cen: efA.cyl.Origin, axis: efA.cyl.AxisDir.AsVector(), nF: nFA.Negate()}
	a2 := miterArm{cen: efB.cyl.Origin, axis: efB.cyl.AxisDir.AsVector(), nF: nFB.Negate()}
	seam, err := sampleMiterSeam(cm.vertex.Point(), efA.cyl.Radius, nS.Negate(), a1, a2)
	if err != nil {
		return nil, false
	}
	return seam, true
}

// writeMiterEnd rewrites one edge's miter corner to the re-sampled seam, matching miterTangents'
// orientation: the shared-face arm carries seam[0]→sBot forward, the outer-face arm carries it
// reversed so its ta→tb still runs the seam the same way the untouched convex path does.
func writeMiterEnd(ef *edgeFillet, atC1 bool, shared *topo.Face, seam []math.Point3) {
	c := &ef.c0
	if atC1 {
		c = &ef.c1
	}
	sTop, sBot := seam[0], seam[len(seam)-1]
	if ef.a == shared {
		c.ta, c.tb, c.seam = sTop, sBot, seam
		return
	}
	c.ta, c.tb, c.seam = sBot, sTop, reversePoints(seam)
}
