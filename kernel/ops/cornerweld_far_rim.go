// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// FAR TERMINATION VARIANT B3 — the CAP-LESS rim continuation (corner-weld-layer-design.md ADR-3, Axis B3).
//
// A constant-radius arm normally ends at a far vertex where a third, transverse face caps the body; that is
// what armFarRunout's admission gate looks for, and its `0 non-host transverse faces` decline is the single
// named blocker N4 had. The forensics (DRAWEXE 8.0.0 per-face reconciliation, receipt in
// .superpowers/sdd/weld-slice1-report.md) show what that decline is really telling us: the rim does not END
// at that vertex, it CONTINUES. N4's picked cap-rim arc is only the 90° piece of a 270° rim that is G1 across
// the boss wall's two faces, and OCCT's blend walks the whole tangent chain — its result carries the fillet
// band over all 270° (oracle faces result_13 = 76.3° over the first wall face + result_10 = exactly 180° over
// the second) and recedes BOTH wall faces to z=45. So the variant is not a new termination: it is the
// recognition that the far vertex is a SEAM, and the arm must run through it to the end of the chain.
//
// This is why the design's count==0 unreachability proof holds trivially: nothing in armFarRunout changes.
// The layer resolves the arm's TRUE far vertex before calling it, and calls it with the LAST edge of the
// chain — whose far vertex does carry a transverse cap, so the existing perpendicular/oblique dispatch runs
// unmodified. Every existing green has count==1 at its own far vertex, so no existing case ever enters the
// walk (its first rimVertexIsSeam test is false).

// rimChainLimit bounds the tangent walk. A rim is split into a handful of faces at most; a longer chain means
// the recogniser is looping on a closed rim, which belongs to the closed-band path, not here.
const rimChainLimit = 8

// rimContinuationLinks walks the arm's tangent-continuous rim chain outward from its picked edge, one link
// per host-face span, and stops at the first far vertex that carries a real capping face (where the shared
// far-runout engine takes over). Declines when the chain dead-ends with no continuation and no cap.
func rimContinuationLinks(first cornerArmLink, arm geom.Surface, plan cornerWeldPlan, res Resolution) ([]cornerArmLink, string) {
	links := []cornerArmLink{first}
	for len(links) <= rimChainLimit {
		cur := links[len(links)-1]
		seam, reason := rimVertexIsSeam(cur)
		if reason != "" {
			return nil, reason
		}
		if !seam {
			return links, "" // a genuine capping face: armFarRunout terminates the arm here
		}
		next, reason := rimContinuationLink(cur, arm, plan, res)
		if reason != "" {
			return nil, reason
		}
		links = append(links, next)
	}
	return nil, fmt.Sprintf("rim continuation: chain exceeded %d links (a closed rim belongs to the band path)", rimChainLimit)
}

// rimVertexIsSeam reports whether the link's far vertex is a G1 SEAM rather than a cap — exactly
// armFarRunout's admission count being ZERO. A count of one is a cap (terminate there); two or more is the
// n-valent regime the engine declines, so the walk stops and lets it say so.
func rimVertexIsSeam(link cornerArmLink) (bool, string) {
	tan, ok := edgeTangentAt(link.edge, link.farVtx)
	if !ok {
		return false, fmt.Sprintf("rim continuation: degenerate arm-edge tangent at vertex %d", link.farVtx.ID())
	}
	_, n := uniqueNonHostTransverseFace(link.farVtx, link.hostA, link.hostB, tan)
	return n == 0, ""
}

// rimContinuationLink finds the single edge continuing the rim past a seam vertex: an UNPICKED edge, tangent
// to the arm's edge there, whose two faces still carry the SAME rolling ball at radius r (so the same fillet
// surface continues). Declines when no such edge exists or several do — never guesses.
func rimContinuationLink(cur cornerArmLink, arm geom.Surface, plan cornerWeldPlan, res Resolution) (cornerArmLink, string) {
	var found []cornerArmLink
	for _, e := range cur.farVtx.Edges() {
		if e.ID() == cur.edge.ID() || plan.filleted[e.ID()] {
			continue
		}
		link, ok := rimContinuationCandidate(cur, e, arm, plan, res)
		if ok {
			found = append(found, link)
		}
	}
	if len(found) != 1 {
		return cornerArmLink{}, fmt.Sprintf("rim continuation: vertex %d has %d tangent-continuous rim continuations (want exactly 1)", cur.farVtx.ID(), len(found))
	}
	return found[0], ""
}

// rimContinuationCandidate tests one edge at the seam vertex as the rim's continuation and, when it is,
// returns the link it defines (its two hosts mapped to the arm's own hostA/hostB roles).
func rimContinuationCandidate(cur cornerArmLink, e *topo.Edge, arm geom.Surface, plan cornerWeldPlan, res Resolution) (cornerArmLink, bool) {
	if !rimEdgesAreTangent(cur.edge, e, cur.farVtx) {
		return cornerArmLink{}, false
	}
	faces := e.Faces()
	if len(faces) != 2 {
		return cornerArmLink{}, false
	}
	hostA, hostB, ok := rimRoleHosts(faces[0], faces[1], cur)
	if !ok {
		return cornerArmLink{}, false
	}
	if !rimBallStillTangent(cur, hostA, hostB, arm, plan, res) {
		return cornerArmLink{}, false
	}
	return cornerArmLink{edge: e, hostA: hostA, hostB: hostB, farVtx: otherEdgeVertex(e, cur.farVtx)}, true
}

// rimEdgesAreTangent reports G1 continuity of two edges meeting at v: their unit tangents there are parallel
// (the seam is smooth) rather than meeting at an angle.
func rimEdgesAreTangent(a, b *topo.Edge, v *topo.Vertex) bool {
	ta, okA := edgeTangentAt(a, v)
	tb, okB := edgeTangentAt(b, v)
	if !okA || !okB {
		return false
	}
	return stdmath.Abs(float64(ta.AsVector().Dot(tb.AsVector()))) > 1-sinFloor
}

// rimRoleHosts maps the continuation edge's two faces onto the arm's hostA/hostB roles: by FACE IDENTITY when
// a host simply continues (N4's shared top plane), otherwise by surface type. ok=false when the pairing is
// ambiguous or does not match the arm's host kinds.
func rimRoleHosts(f0, f1 *topo.Face, cur cornerArmLink) (*topo.Face, *topo.Face, bool) {
	for _, pair := range [2][2]*topo.Face{{f0, f1}, {f1, f0}} {
		if pair[0] == cur.hostA && pair[1] != cur.hostB {
			return pair[0], pair[1], true
		}
		if pair[1] == cur.hostB && pair[0] != cur.hostA {
			return pair[0], pair[1], true
		}
	}
	if sameSurfaceKind(f0, cur.hostA) && sameSurfaceKind(f1, cur.hostB) {
		return f0, f1, true
	}
	if sameSurfaceKind(f1, cur.hostA) && sameSurfaceKind(f0, cur.hostB) {
		return f1, f0, true
	}
	return nil, nil, false
}

// sameSurfaceKind reports whether two faces carry the same surface TYPE (Plane/Cylinder/Torus/…).
func sameSurfaceKind(x, y *topo.Face) bool {
	return fmt.Sprintf("%T", x.Geometry()) == fmt.Sprintf("%T", y.Geometry())
}

// rimBallStillTangent is the geometric admission test for a continuation: the arm's own rolling ball, placed
// at the seam vertex, must be internally tangent at radius r to BOTH candidate hosts. That is what makes the
// continuation the same fillet — checked on geometry rather than on surface-object identity, which a STEP
// round trip does not preserve across faces.
func rimBallStillTangent(cur cornerArmLink, hostA, hostB *topo.Face, arm geom.Surface, plan cornerWeldPlan, res Resolution) bool {
	ball, ok := armBallCenter(arm, cur.farVtx.Point())
	if !ok {
		return false
	}
	tol := res.Weld() * plan.radius
	_, okA := armRunoutFoot(hostA, ball, plan.radius, tol)
	_, okB := armRunoutFoot(hostB, ball, plan.radius, tol)
	return okA && okB
}

// otherEdgeVertex returns e's endpoint that is not v.
func otherEdgeVertex(e *topo.Edge, v *topo.Vertex) *topo.Vertex {
	if e.StartVertex().ID() == v.ID() {
		return e.EndVertex()
	}
	return e.StartVertex()
}
