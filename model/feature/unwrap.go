// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Unwrap is a model feature (M20-F13): it flattens a cylindrical face into a planar patch — the
// developable surface unrolled to a flat rectangle (arc length × axial height). The flattened
// patch is appended as a sheet body (an open surface), leaving the source solid in place; it
// lands in the part's WorkSurfaces collection.

// UnwrapDefinition names the cylindrical face to flatten.
type UnwrapDefinition struct {
	FaceKey []byte
	// FaceAnchors maps FaceKey to its mint-time centroid for the geometric recovery tier
	// (ADR-0043 P6 / #1579); see FilletDefinition.EdgeAnchors.
	FaceAnchors map[string]math.Point3
}

// UnwrapFeature appends the flattened patch of its cylindrical face.
type UnwrapFeature struct {
	def      *UnwrapDefinition
	featName string
}

// Definition returns the unwrap recipe.
func (u *UnwrapFeature) Definition() *UnwrapDefinition { return u.def }

// Kind implements [Feature].
func (u *UnwrapFeature) Kind() string { return "unwrap" }

// Recompute flattens the named cylindrical face and appends the flat sheet to the result.
func (u *UnwrapFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "unwrap")
	if err != nil {
		return Output{}, err
	}
	face, mt, err := bindFace(body, u.def.FaceKey, anchorFor(u.def.FaceKey, u.def.FaceAnchors))
	if err != nil {
		return Output{}, fmt.Errorf("unwrap: %w", err)
	}
	patch, err := unwrapCylindricalFace(face, featOr(u.featName, "unwrap"))
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), patch), Heals: faceHeal(u.def.FaceKey, mt)}, nil
}

// unwrapCylindricalFace unrolls a cylindrical face into a flat rectangle sheet of arc-length ×
// axial-height, where arc length = radius × the face's angular span (a full/periodic face
// unrolls the whole circumference).
func unwrapCylindricalFace(face *topo.Face, feat string) (*topo.Body, error) {
	cyl, ok := face.Geometry().(geom.Cylinder)
	if !ok {
		return nil, fmt.Errorf("unwrap: face is %T, want a cylinder", face.Geometry())
	}
	height, span := cylindricalFaceExtent(cyl, face.Vertices())
	if height <= 1e-9 {
		return nil, fmt.Errorf("unwrap: degenerate axial height %g", height)
	}
	arc := cyl.Radius * span
	return flatSheet(arc, height, feat), nil
}

// cylindricalFaceExtent returns the face's axial height and angular span (radians) from its
// vertices. A periodic/full face (vertices clustered at a seam, or spanning the full turn)
// unrolls the whole circumference (2π).
func cylindricalFaceExtent(cyl geom.Cylinder, verts []*topo.Vertex) (height, span float64) {
	axis := cyl.AxisDir.AsVector()
	ref := cyl.Ref.AsVector()
	bi := axis.Cross(ref)
	loAx, hiAx := stdmath.Inf(1), stdmath.Inf(-1)
	loAng, hiAng := stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range verts {
		d := cyl.Origin.VectorTo(v.Point())
		ax := d.Dot(axis)
		radial := d.Sub(axis.Scale(ax))
		ang := stdmath.Atan2(radial.Dot(bi), radial.Dot(ref))
		loAx, hiAx = stdmath.Min(loAx, ax), stdmath.Max(hiAx, ax)
		loAng, hiAng = stdmath.Min(loAng, ang), stdmath.Max(hiAng, ang)
	}
	angSpan := hiAng - loAng
	if angSpan < 0.5 || angSpan > 2*stdmath.Pi-0.5 { // seam-only or near-full ⇒ a periodic full turn
		angSpan = 2 * stdmath.Pi
	}
	return hiAx - loAx, angSpan
}

// flatSheet builds a flat [0,w]×[0,h] rectangle as an open (sheet) surface body on the XY plane.
func flatSheet(w, h float64, feat string) *topo.Body {
	lin := topo.NewLineage(topo.Tok(feat, "patch", 0))
	bld := topo.NewBuilder(false, lin) // false ⇒ an open surface body
	corners := []math.Point3{math.P3(0, 0, 0), math.P3(w, 0, 0), math.P3(w, h, 0), math.P3(0, h, 0)}
	vs := make([]*topo.Vertex, 4)
	for i, c := range corners {
		vs[i] = bld.AddVertex(c, lin)
	}
	uses := make([]topo.Use, 4)
	for i := range corners {
		e := bld.AddEdge(geom.NewLineSegment(corners[i], corners[(i+1)%4]), vs[i], vs[(i+1)%4], lin)
		uses[i] = topo.Use{Edge: e}
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, lin, topo.OuterLoop(uses...))
	return bld.Build()
}

// AddUnwrap appends the flattened patch of the cylindrical face referenced by faceKey. It captures
// the face's mint-time anchor against the running body for the geometric recovery tier (ADR-0043
// P6 / #1579); the recipe restore uses addUnwrap so reopening never recaptures.
func (c *ModifyFeatures) AddUnwrap(faceKey []byte) *PartFeature {
	def := &UnwrapDefinition{FaceKey: faceKey}
	def.FaceAnchors = captureFaceAnchors(featuresTipBody(c.engine), [][]byte{faceKey})
	return c.addUnwrap(def)
}

// addUnwrap registers an unwrap from a fully-built definition without capturing anchors (the
// recipe restore path, which carries the persisted anchors of its own).
func (c *ModifyFeatures) addUnwrap(def *UnwrapDefinition) *PartFeature {
	uf := &UnwrapFeature{def: def}
	pf := c.engine.Add(uf)
	pf.SetName(c.engine.UniqueName("Unwrap"))
	uf.featName = pf.name
	return pf
}
