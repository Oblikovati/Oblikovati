// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"maps"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// dihedralOrthoCosTol is the largest |cos θ| between two miter edge axes still treated as an
// orthogonal (θ=90°) corner. Both axes are unit vectors, so this is a DIMENSIONLESS,
// scale-invariant cutoff (unlike a length tolerance it needs no model-scale factor). Only the
// orthogonal setback distance s = r·cot(45°) = r is derived here; a non-orthogonal miter
// (s = r·cot(θ/2), OCCT case L7) is a separate slice and must NOT fire this pass.
const dihedralOrthoCosTol = 1e-9

// The concave dihedral-miter and concave trihedral-sphere corner treatments (OCCT tests/blend/simple
// L1/L7/N5 and K6/L4) are accumulated by the unified pass (fillet_corner_setback_unified.go):
// accumulateDihedral re-samples the setback seam via concaveMiterSeam, and accumulateConcaveSphere +
// blendsWithVoidSpheres flip each concave corner's sphere to the VOID side and force ONE re-solve. The
// gate predicates (concaveOrthogonalDihedralMiter, concaveTrihedralCornerFaces) and geometry
// (concaveMiterSeam, solveVoidCornerSphere) below are the reused helpers the classifier + accumulate
// own; the old applyCornerSetback / flipConcaveTrihedralBlends entrypoints were folded into that pass.

// cloneBlendResetArcs copies a corner blend but clears its arcs, so a computeFillets re-run repopulates
// them exactly once (registerBlendArc APPENDS — reusing the original struct would double every arc).
func cloneBlendResetArcs(cb *cornerBlend) *cornerBlend {
	tan := make(map[uint64]math.Point3, len(cb.tan))
	maps.Copy(tan, cb.tan)
	// radiusTorus rides the clone: a body with BOTH a void-sphere corner and a mixed-radius torus
	// corner re-solves its bands against the same transient blend, and the classifier must still
	// route the torus vertex to accumulateRadiusTorus after the re-solve.
	return &cornerBlend{vertex: cb.vertex, center: cb.center, sphere: cb.sphere, tan: tan, radiusTorus: cb.radiusTorus}
}

// concaveTrihedralCornerFaces returns the three faces of the corner at vid and ok=true when it is the
// K6-class corner this pass owns: exactly three CONCAVE (ef.flip) fillet ends meeting three mutually
// ORTHOGONAL PLANAR faces. Any other config (mixed-sense, convex, non-orthogonal, non-planar, or a
// valence other than 3) returns ok=false so the corner keeps its material-side sphere byte-identical.
func concaveTrihedralCornerFaces(vid uint64, cb *cornerBlend, fils []edgeFillet) ([]*topo.Face, bool) {
	if cb == nil || cb.vertex == nil {
		return nil, false
	}
	faces, concave := blendCornerFaces(vid, fils)
	if len(faces) != 3 || !concave {
		return nil, false
	}
	return faces, orthogonalPlanarTriple(faces)
}

// blendCornerFaces collects the distinct faces of the fillet ends that blend at vid and reports whether
// EVERY such end is a concave (ef.flip) fillet — the same-sense concavity the void sphere requires.
func blendCornerFaces(vid uint64, fils []edgeFillet) ([]*topo.Face, bool) {
	seen := map[uint64]*topo.Face{}
	ends := 0
	concave := true
	for i := range fils {
		for _, c := range []corner{fils[i].c0, fils[i].c1} {
			if !c.blend || c.vertex == nil || c.vertex.ID() != vid {
				continue
			}
			ends++
			concave = concave && fils[i].flip
			seen[c.a.ID()], seen[c.b.ID()] = c.a, c.b
		}
	}
	faces := make([]*topo.Face, 0, len(seen))
	for _, f := range seen {
		faces = append(faces, f)
	}
	return faces, ends == 3 && concave
}

// orthogonalPlanarTriple reports whether the three faces are planar with mutually PERPENDICULAR outward
// normals (the orthogonal box corner: |cos| between every pair below the dimensionless dihedralOrthoCosTol).
func orthogonalPlanarTriple(faces []*topo.Face) bool {
	ns := make([]math.Vector3, 0, 3)
	for _, f := range faces {
		n, ok := planeNormal(f)
		if !ok {
			return false
		}
		ns = append(ns, n)
	}
	return stdmath.Abs(ns[0].Dot(ns[1])) < dihedralOrthoCosTol &&
		stdmath.Abs(ns[0].Dot(ns[2])) < dihedralOrthoCosTol &&
		stdmath.Abs(ns[1].Dot(ns[2])) < dihedralOrthoCosTol
}

// solveVoidCornerSphere builds the concave corner's rolling-ball sphere on the VOID side of the three
// planes: its centre is r from each face on the OUTWARD (void) side (n·s = n·o + r, the sign flip from
// solvePlanarBlend's material-side n·s = n·o − r), and each tangent point is the centre pushed r back
// toward the face (s − n·r). It returns a fresh blend (arcs nil) so the corner re-solve can register the
// three bounding arcs on it. ok=false on a non-planar face or a degenerate (near-parallel) triple.
func solveVoidCornerSphere(v *topo.Vertex, faces []*topo.Face, r float64) (*cornerBlend, bool) {
	s, ok := voidSphereCentre(faces, r)
	if !ok {
		return nil, false
	}
	sph, err := geom.NewSphere(s, r)
	if err != nil {
		return nil, false
	}
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		n, _ := planeNormal(f)
		tan[f.ID()] = s.TranslateBy(n.Scale(-r)) // back toward the face from the void centre
	}
	return &cornerBlend{vertex: v, center: s, sphere: sph, tan: tan}, true
}

// voidSphereCentre solves the point r from each of the three planes on their VOID (outward-normal) side —
// the sign flip (n·s = n·o + r) from solvePlanarBlend's material-side n·s = n·o − r. ok=false on a
// non-planar face or a degenerate (near-parallel) triple.
func voidSphereCentre(faces []*topo.Face, r float64) (math.Point3, bool) {
	var a [3][3]float64
	var b [3]float64
	for i, f := range faces {
		n, ok := planeNormal(f)
		if !ok {
			return math.Point3{}, false
		}
		pl := f.Geometry().(geom.Plane)
		a[i] = [3]float64{n.X, n.Y, n.Z}
		b[i] = n.Dot(pl.Origin.AsVector()) + r
	}
	x, ok := probe.Solve3(a, b)
	if !ok {
		return math.Point3{}, false
	}
	return math.P3(x[0], x[1], x[2]), true
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
	cos := probe.Unit(efA.cyl.AxisDir.AsVector()).Dot(probe.Unit(efB.cyl.AxisDir.AsVector()))
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
