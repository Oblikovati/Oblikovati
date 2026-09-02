// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The corner FRAME: the geometry a corner is solved in (split out of fillet.go for #2217).
//
// A corner sits at a vertex where blended edges meet. Its frame comes from the host faces' planes
// and outward normals; from that frame come the tangent points, the arc centre and the seam the
// blend face runs to. The blend and miter records that carry a solved corner live here too, beside
// the frame that defines them.

// edgePlanarFaces returns the edge's two faces and their outward normals, erroring unless
// the edge bounds exactly two planar faces.
func edgePlanarFaces(e *topo.Edge) (a, b *topo.Face, nA, nB math.Vector3, err error) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, nA, nB, fmt.Errorf("fillet: edge bounds %d faces, need 2", len(faces))
	}
	pa, oka := faces[0].Geometry().(geom.Plane)
	pb, okb := faces[1].Geometry().(geom.Plane)
	if !oka || !okb {
		return nil, nil, nA, nB, fmt.Errorf("fillet: both faces of the edge must be planar")
	}
	// Material-OUTWARD normals: a plane's geometric normal negated when its face is reversed.
	// Native construction leaves faces unreversed with outward plane normals, but STEP-imported
	// (and any oriented) faces carry a Reversed flag with an inward plane normal. Ignoring it
	// flips offDir outward, so the rolling-ball centre lands outside and a plainly convex edge
	// reads as non-convex — filleting every imported solid failed until this was applied.
	return faces[0], faces[1], outwardPlaneNormal(faces[0], pa), outwardPlaneNormal(faces[1], pb), nil
}

// outwardPlaneNormal is a planar face's material-outward normal (its plane normal, negated
// when the face is reversed) — matching outwardFaceNormal's orientation handling.
func outwardPlaneNormal(f *topo.Face, p geom.Plane) math.Vector3 {
	if f.Reversed() {
		return p.Normal().Negate()
	}
	return p.Normal()
}

// cornerInputs bundles the per-edge data a corner needs. offDir is the centre offset from
// the edge into the solid PER UNIT RADIUS (a variable fillet's centre line follows offDir
// scaled by the local radius).
type cornerInputs struct {
	a, b   *topo.Face
	nA, nB math.Vector3
	offDir math.Vector3
	axis   math.Vector3
	flip   bool    // invert the cylinder face's outward sense (a concave fillet's surface faces the centre)
	weld   float64 // model-relative coincidence tolerance for the F3a spine-concurrence gate (armCornerCentre)
}

// cornerAt solves a fillet corner at vertex v with the local radius r. Without a blend it is
// a simple end: centre v+offDir·r, tangent points r along each face normal, an arc on the end
// face. With a blend (v is a shared corner) the centre is the blend sphere's centre and the
// tangent points are the sphere's tangents on the two faces; the corner-end arc joins the
// sphere patch (no end face), and the arc is registered on the blend.
func cornerAt(v *topo.Vertex, in cornerInputs, r float64, blend *cornerBlend, miter *cornerMiter, variable bool) (corner, error) {
	if r <= runoutTol { // a variable fillet tapered to 0: the blend collapses to an apex on the edge here
		p := v.Point()
		return corner{a: in.a, b: in.b, vertex: v, cen: p, ta: p, tb: p, mid: p, runout: true}, nil
	}
	cen, ta, tb, arcCen, end, seam, err := cornerTangents(v, in, r, blend, miter)
	if err != nil {
		return corner{}, err
	}
	// mid uses arcCen (the blend ball for a shared corner) so the sphere-patch arc registers ON the sphere
	// even when the arm SURFACE centre was kept frame-derived by the F3a spine-concurrence gate (armCornerCentre).
	mid := arcCen.TranslateBy(perpToward(arcCen, v.Point(), in.axis).Scale(r))
	c := corner{a: in.a, b: in.b, endFace: end, vertex: v, cen: cen, ta: ta, tb: tb, mid: mid, blend: blend != nil, miter: miter != nil, seam: seam}
	registerBlendArc(blend, c, in, variable)
	return c, nil
}

// cornerTangents resolves a corner's arm-surface centre (cen), the two face tangent points, the sphere-patch
// arc centre (arcCen), and the end face / miter seam, by corner kind: a miter seam, a blend ball (whose
// override is gated on spine concurrence via armCornerCentre, F3a), or a plain end-face round. Split out of
// cornerAt to keep it within funlen; it errors only on a plain corner with no end face to round.
func cornerTangents(v *topo.Vertex, in cornerInputs, r float64, blend *cornerBlend, miter *cornerMiter) (cen, ta, tb, arcCen math.Point3, end *topo.Face, seam []math.Point3, err error) {
	cen = v.Point().TranslateBy(in.offDir.Scale(r)) // the frame-derived rolling-ball centre, on the arm axis
	ta = cen.TranslateBy(in.nA.Scale(r))
	tb = cen.TranslateBy(in.nB.Scale(r))
	arcCen = cen // the sphere-patch arc's centre: the blend ball for a shared (concurrent OR canal) corner
	switch {
	case miter != nil:
		ta, tb, seam = miterTangents(in, miter) // the end is the seam, not an end-face arc
	case blend != nil:
		cen, ta, tb, arcCen = armCornerCentre(cen, in, blend), blend.tan[in.a.ID()], blend.tan[in.b.ID()], blend.center
	default:
		if end = endFaceAt(v, in.a, in.b); end == nil {
			return cen, ta, tb, arcCen, nil, nil, fmt.Errorf("fillet: edge endpoint has no end face to round")
		}
	}
	return cen, ta, tb, arcCen, end, seam, nil
}

// armCornerCentre returns the ARM SURFACE's rolling-ball centre at a shared (blend) corner, gating the
// blend-ball override on SPINE CONCURRENCE (F3a). The arm's offset spine is the LINE through the
// frame-derived centre along the edge axis; the blend ball sits ON that line — but OFFSET by the setback
// distance ALONG the axis, so the test is the ball's PERPENDICULAR distance to the spine line, not the raw
// point distance (which the setback would always overshoot). It adopts the blend ball only when that
// perpendicular gap ≤ in.weld. Where the blend ball lies on THIS arm's spine (perp ≈ 0 — every planar
// box/round/setback corner, and the arms of a partly-concurrent corner) this adopts the blend ball,
// byte-identical to the pre-F3a override. Where it does not — the s_10 canal arm, whose ball is 10 off its
// x=55 spine — the frame-derived centre is kept, so the arm cylinder is NOT built on the mirrored x=45
// side; the corner-blend arc still registers on the ball (blend.center/mid), decoupled from this centre.
// (Note: a corner may be concurrent for some arms and not others — L6 adopts on 1 of its 3 arms and keeps
// frame on the other 2; its byte-identity across F3a comes from that per-arm split + the curved-corner
// machinery rebuilding the kept-frame arms, NOT from a single ball lying on every arm's spine.)
func armCornerCentre(frame math.Point3, in cornerInputs, blend *cornerBlend) math.Point3 {
	d := frame.VectorTo(blend.center)
	perp := d.Sub(in.axis.Scale(d.Dot(in.axis))) // component of ball→spine offset ⊥ to the edge axis
	if perp.Length() <= in.weld {
		return blend.center
	}
	return frame
}

// registerBlendArc records the corner's boundary arc on the sphere patch when v is a blend corner. A
// variable edge stores the arc as the cone's chord polyline so the patch and cone meet edge-for-edge.
func registerBlendArc(blend *cornerBlend, c corner, in cornerInputs, variable bool) {
	if blend == nil {
		return
	}
	arc := blendArc{ta: c.ta, tb: c.tb, mid: c.mid}
	if variable {
		arc.chords = arcChords(c, in, cornerChordCount(in))
	}
	blend.arcs = append(blend.arcs, arc)
}

// miterTangents returns this edge's corner tangents and the seam oriented ta→tb: the shared
// face carries the seam's top (sTop, on the shared face), the outer face carries its bottom
// (sBot, on the now-shortened sharp edge). The seam is the SAME point list for both edges of
// the miter — reversed for the edge whose A face is the outer one — so the two cylinders weld
// along it watertight.
func miterTangents(in cornerInputs, m *cornerMiter) (ta, tb math.Point3, seam []math.Point3) {
	if m.shared == nil {
		return vertexOnlyMiterTangents(in, m) // D4: no shared face — orient by which sharp edge bounds in.a
	}
	if in.a == m.shared {
		return m.seam[0], m.sBot, m.seam
	}
	return m.sBot, m.seam[0], reversePoints(m.seam)
}

// reversePoints returns a reversed copy of pts.
func reversePoints(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

// perpToward returns the unit direction from cen toward p projected into the plane
// perpendicular to axis — the in-cross-section direction to the rounded corner.
func perpToward(cen, p math.Point3, axis math.Vector3) math.Vector3 {
	d := cen.VectorTo(p)
	perp := d.Sub(axis.Scale(d.Dot(axis)))
	u, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return d
	}
	return u.AsVector()
}

// endFaceAt returns the face meeting at v that is neither a nor b (the end cap the fillet
// rounds), or nil if there is none.
func endFaceAt(v *topo.Vertex, a, b *topo.Face) *topo.Face {
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if f != a && f != b {
				return f
			}
		}
	}
	return nil
}

// blendArc is one boundary arc of a corner sphere patch (shared with a cylinder fillet). chords is
// nil for an analytic single arc (a constant cylinder), or the chord polyline ta…tb when the arc is
// shared with a VARIABLE cone, whose faceted end must match the patch edge-for-edge to stay watertight.
type blendArc struct {
	ta, tb, mid math.Point3
	chords      []math.Point3
}

// cornerBlend is a spherical corner patch where several filleted edges meet at one vertex:
// the rolling-ball sphere tangent to the corner's faces, its tangent point on each face
// (keyed by face id), and the arcs (filled in as the edges are solved) that bound the patch.
type cornerBlend struct {
	vertex *topo.Vertex
	center math.Point3
	sphere geom.Sphere
	tan    map[uint64]math.Point3
	arcs   []blendArc
	// radiusTorus marks a mixed-radius trihedral TORUS corner (A4: rB on the wall∧wall edge, equal
	// rS on the two top edges): the bands build against this transient blend and the unified setback
	// pass retracts them onto the solved torus (fillet_corner_radiustorus.go). Nil everywhere else,
	// so every equal-radius corner path is untouched.
	radiusTorus *radiusTorusCornerGeom
}
