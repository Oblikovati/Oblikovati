// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Per-face certification of a boolean result (M48/C3, Oblikovati/Oblikovati#3445–#3448).
//
// A whole-body volume comparison is a smoke test, never a proof: a result can hold the right amount
// of material in the wrong place, and a bracket wide enough to absorb a tessellation deficit is wide
// enough to pass a mis-recognized lobe. The proof is Requicha's membership rule applied FACE BY
// FACE — every face of A op B lies on ∂A or ∂B, and on the side the operation keeps:
//
//	A ∪ B: ∂A outside B, or ∂B outside A
//	A ∖ B: ∂A outside B, or ∂B inside A
//	A ∩ B: ∂A inside B,  or ∂B inside A
//
// Both the "on" test and the "inside" test read the analytic B-rep (brep.PointOnFace,
// brep.PointInside), so the gate is exact and Quality-independent — the ground rule that an oracle
// gating a result must be more exact than the result it gates.

// certifyBooleanFaces judges a result against the OPERANDS, which are the only oracle a boolean has,
// over two of the three quantities the ground rules name: where each face lies (membership) and how
// much boundary AREA the result shows in total. It returns the evidence rather than a verdict so one
// pass serves both — the interior-point probe behind each face is a grid search, and asking for it
// twice doubled the gate.
//
// A face whose interior point cannot be found is SKIPPED rather than failed, and COUNTED. The
// certificate refuses results it can disprove and never rejects one merely because a probe was
// unavailable — but a blind spot nothing reports is a proof nobody can size. Measured on the RING
// corpus (Oblikovati/Oblikovati#3516), it skipped the bored torus face of every drilled ring —
// 296.088 of the result's 296.107 of area — at every bore radius including the ones it certifies as
// exact, because FaceInteriorPoint had no probe for a face holding the COMPLEMENT of what its loops
// enclose. It therefore examined one face of two and called the result certified. Over the
// drill/torus/ring/bore rows of this package that was 26 unexamined faces of 248; it is 0 of 272 now.
// Over the whole package, 4 of 2469 remain — two single-loop spheres and two two-loop tori, which I
// did not chase — a torus − box cut owns one of them, and the ratchet below pins it there.
//
// TestTheCertificateLeavesExactlyTheKnownFacesUnprobed holds this count, because a count nothing pins
// drifts and this one did: fix round 1 of #3516 gave 22 of those probes back in a comment-level
// change, and shipped three statements that had become false. The ratchet fails on a RISE, which is
// lost coverage, and on a FALL, which is an improvement whose pin should come down with it.
//
// Example: if ev := certifyBooleanFaces(Cut, target, tool, body, sizes.res); !ev.kept { /* guarded */ }
// res is the pair's extent resolution, decided once by the size classification and carried here: the
// certificate used to rebuild it from the two range boxes, which is the same predicate evaluated a
// second time with its own chance of a different answer (#3524).
func certifyBooleanFaces(op PartFeatureOperation, target, tool, body *topo.Body, res Resolution) faceEvidence {
	if body == nil || target == nil || tool == nil {
		return faceEvidence{}
	}
	tol := res.Sew()
	ev := faceEvidence{kept: true}
	ta, to := newBoundaryIndex(target), newBoundaryIndex(tool)
	for _, f := range body.Faces() {
		ev.addArea(f) // BEFORE the probe: the area bound must cover the faces the probe cannot find
		p, ok := query.FaceInteriorPoint(f)
		if !ok {
			ev.unprobed++
			continue
		}
		if !pointKeptBy(op, ta, to, p, tol) {
			ev.kept = false
			return ev
		}
	}
	return ev
}

// faceEvidence is what one pass over a result's faces establishes: whether every face it could probe
// is one the operation keeps, how many it could neither probe nor measure, and how much boundary
// area the ones it measured carry.
type faceEvidence struct {
	kept       bool
	unprobed   int     // faces with no interior point: the certificate could not look at them
	unmeasured int     // faces whose analytic area declined: left out of claimed, which only weakens it
	claimed    float64 // total analytic area of the faces it did measure
}

// addArea credits a face's analytic area to the result's total. It runs on EVERY face, including the
// ones the interior probe cannot find: the membership certificate is blind there and the area bound
// must not be, or a result that grew boundary on an unprobeable face would be invisible to both.
//
// A face whose analytic area declines is counted in unmeasured and left OUT of the total, which can
// only make it smaller — the bound stays sound, but it stops covering the whole body, so the count is
// REPORTED rather than only kept. Sound and vacuous are compatible.
func (ev *faceEvidence) addArea(f *topo.Face) {
	area, ok := query.AnalyticFaceArea(f)
	if !ok {
		ev.unmeasured++
		return
	}
	ev.claimed += area
}

// overclaimsItsOperands reports whether the result shows more boundary than its two operands have
// between them. Every face of A op B lies on ∂A or ∂B — the membership rule below is exactly that
// statement — and a boolean TRIMS those boundaries, so the total the result shows cannot exceed
// area(A) + area(B). It is an identity, not an accuracy statement, which is why the only slack is a
// rounding one.
//
// It is bounded by the PAIR and not by each operand, and that is not a weakening for convenience: the
// stronger per-operand form is FALSE. A cocylindrical join merges the boss's wall and its host's into
// ONE face lying on both (ADR-0061 stage 5), and a single interior point names only one of them, so
// the merged face is credited whole to that operand. Measured on
// TestCocylindricalCapOnWallIsOneAnalyticFace, the one row in kernel/ops where the two forms differ:
// 203.575 credited to a target holding 169.646 — a correct result the per-operand bound refuses.
//
// What this catches is a result that DUPLICATED or grew boundary. It does not catch a face measured
// badly over the RIGHT region: the bore wall of a 1e-3 drill through the RING measures 1.8e-4
// relative high against an independent quadrature and still sits far under the pair's own area.
// Separating those needs an oracle more exact than the kernel's own analytic face integral, which
// does not exist in process; the root is fixed at source instead (Oblikovati/Oblikovati#3538).
func (ev faceEvidence) overclaimsItsOperands(target, tool *topo.Body) (available float64, over bool) {
	ta, aok := query.AnalyticGeometryProperties(target)
	to, bok := query.AnalyticGeometryProperties(tool)
	if !aok || !bok {
		return 0, false // an operand the analytic path cannot measure is no oracle
	}
	available = ta.Area + to.Area
	return available, ev.claimed > available*(1+operandAreaSlack)
}

// operandAreaSlack is how far past its operands' combined area a result's own may sit before it is a
// contradiction rather than a rounding, relative. Measured across every boolean in kernel/ops: the
// largest ratio any accepted result reaches is 1 + 1e-15, the quadrature's own convergence, and the
// only thing above it is the deliberate probe in TestAResultShowingMoreBoundaryThanItsOperandsIsRefused
// at 1 + 1.07e-2. Every value between those two gives the same verdicts; 1e-6 sits in the middle.
const operandAreaSlack = 1e-6 // tol:calibrated — plateau 1e-14..1e-2, measured across kernel/ops

// pointKeptBy applies the membership rule to one boundary point. A point on BOTH operands'
// boundaries is kept by every operation — that is a coincident-face contact, where the shared
// boundary survives once — so it is accepted before the per-op split.
func pointKeptBy(op PartFeatureOperation, target, tool *boundaryIndex, p math.Point3, tol float64) bool {
	if op != Join && op != Cut && op != Intersect {
		return true // an operation with no membership rule to check
	}
	onTarget, onTool := target.on(p, tol), tool.on(p, tol)
	if !onTarget && !onTool {
		return false // the face lies on neither operand: the result fabricated it
	}
	if onTarget && onTool {
		return true
	}
	if onTarget {
		return targetBoundaryKept(op, tool, p)
	}
	return toolBoundaryKept(op, target, p)
}

// boundaryIndex is one operand prepared for repeated "does this point lie on the boundary?"
// questions: its faces, with a box tree over their range boxes, so a probe only reaches the faces
// whose box covers it. It is built ONCE per boolean — scanning every face per probe made the gate
// cost O(result faces × operand faces) exact projections, which on a body of tens of thousands of
// faces costs far more than the boolean it is checking.
type boundaryIndex struct {
	body    *topo.Body
	faces   []*topo.Face
	tree    *geom.BoxTree
	unboxed []*topo.Face // faces with no range box: a BOUNDARY-LESS face has no vertices to build one
}

// newBoundaryIndex prepares b's faces for boundary probes. A face with an empty range box cannot be
// found by a tree query, so those are kept aside and always tested — that is the whole sphere a ball
// is made of, exactly the face a coaxial sphere/rod boolean has to certify.
func newBoundaryIndex(b *topo.Body) *boundaryIndex {
	bi := &boundaryIndex{body: b}
	var boxes []math.Box
	for _, f := range b.Faces() {
		if box := f.RangeBox(); !box.IsEmpty() {
			bi.faces = append(bi.faces, f)
			boxes = append(boxes, box)
			continue
		}
		bi.unboxed = append(bi.unboxed, f)
	}
	bi.tree = geom.NewBoxTree(boxes)
	return bi
}

// on reports whether p lies on any of the body's trimmed faces, within tol.
func (bi *boundaryIndex) on(p math.Point3, tol float64) bool {
	for _, f := range bi.unboxed {
		if brep.PointOnFace(f, p, tol) {
			return true
		}
	}
	found := false
	bi.tree.Query(pointReach(p, tol), func(i int) bool {
		found = brep.PointOnFace(bi.faces[i], p, tol)
		return found
	})
	return found
}

// inside reports whether p is strictly inside the body.
func (bi *boundaryIndex) inside(p math.Point3) bool { return brep.PointInside(bi.body, p) }

// pointReach is the query box for a probe: the point grown by the on-boundary tolerance.
func pointReach(p math.Point3, tol float64) math.Box {
	t := math.Scalar(tol)
	return math.NewBox(math.P3(p.X-t, p.Y-t, p.Z-t), math.P3(p.X+t, p.Y+t, p.Z+t))
}

// targetBoundaryKept applies the rule to a point on the TARGET's boundary alone: a join or a cut
// keeps the part outside the tool, an intersection keeps only what the tool contains.
func targetBoundaryKept(op PartFeatureOperation, tool *boundaryIndex, p math.Point3) bool {
	if op == Intersect {
		return tool.inside(p)
	}
	return !tool.inside(p)
}

// toolBoundaryKept applies the rule to a point on the TOOL's boundary alone: a cut or an
// intersection keeps what the target contains (the carved wall, the shared lens), a join keeps the
// part outside it.
func toolBoundaryKept(op PartFeatureOperation, target *boundaryIndex, p math.Point3) bool {
	if op == Join {
		return !target.inside(p)
	}
	return target.inside(p)
}
