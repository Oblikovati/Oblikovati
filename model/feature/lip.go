// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Lip is a plastic-part feature (M20-F10): a rectangular bead run along selected edges, either
// raised as a mating lip (Join) or recessed as a groove (Cut). Width sizes the bead along the
// first adjacent face, Height along the second; the bead is centred on the edge so one quadrant
// overlaps the body (bonding the lip / biting the groove) and the rest protrudes.

// LipDefinition is the lip/groove recipe.
type LipDefinition struct {
	EdgeKeys [][]byte
	Width    func() float64
	Height   func() float64
	Groove   bool // cut a groove instead of raising a lip
}

// LipFeature runs a bead along the selected edges.
type LipFeature struct {
	def      *LipDefinition
	featName string
}

// Definition returns the lip recipe.
func (l *LipFeature) Definition() *LipDefinition { return l.def }

// Kind implements [Feature].
func (l *LipFeature) Kind() string { return "lip" }

// Recompute joins (lip) or cuts (groove) a bead along each selected edge into the running body.
func (l *LipFeature) Recompute(in Input) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	w, h := callOrZero(l.def.Width), callOrZero(l.def.Height)
	if w <= 0 || h <= 0 {
		return Output{}, fmt.Errorf("lip: width %g and height %g must both be > 0", w, h)
	}
	edges, heals, err := resolveEdges(body, l.def.EdgeKeys, nil)
	if err != nil {
		return Output{}, err
	}
	op := ops.Join
	if l.def.Groove {
		op = ops.Cut
	}
	result := body
	for i, edge := range edges {
		bead, err := lipBead(edge, w, h, fmt.Sprintf("%s/b%d", featOr(l.featName, "lip"), i))
		if err != nil {
			return Output{}, err
		}
		if result, err = ops.Boolean(op, result, bead); err != nil {
			return Output{}, err
		}
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result), Heals: heals}, nil
}

// lipBead builds the Width×Height bead swept along an edge: a rectangle in the plane
// perpendicular to the edge, centred on the edge line and oriented to the adjacent faces (Width
// along face 0's interior, Height along face 1's), swept the edge's length with a small overhang.
func lipBead(edge *topo.Edge, w, h float64, feat string) (*topo.Body, error) {
	faces := edge.Faces()
	if len(faces) != 2 {
		return nil, fmt.Errorf("lip: edge bounds %d faces, need 2", len(faces))
	}
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	e, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return nil, fmt.Errorf("lip: degenerate edge")
	}
	mid := v0.Midpoint(v1)
	t1, t2 := interiorDir(faces[0], mid, e), interiorDir(faces[1], mid, e)
	if t1.LengthSquared() == 0 || t2.LengthSquared() == 0 {
		return nil, fmt.Errorf("lip: cannot orient the bead against the edge faces")
	}
	plane := planePerp(v0, e)
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	proj := func(vec math.Vector3) math.Point2 { return math.P2(vec.Dot(u), vec.Dot(v)) }
	a, b := t1.Scale(w/2), t2.Scale(h/2) // half-extents along the two face interiors
	poly := []math.Point2{proj(a.Add(b)), proj(a.Sub(b)), proj(a.Scale(-1).Sub(b)), proj(a.Scale(-1).Add(b))}
	return buildPrism(poly, plane, span{near: -cutterOverhang, far: v0.DistanceTo(v1) + cutterOverhang}, 0, feat), nil
}

// AddLip runs a bead along the given edges — a raised lip (groove=false) or a recessed groove
// (groove=true), sized width × height.
func (c *DressUpFeatures) AddLip(edgeKeys [][]byte, width, height func() float64, groove bool) *PartFeature {
	lf := &LipFeature{def: &LipDefinition{EdgeKeys: edgeKeys, Width: width, Height: height, Groove: groove}}
	pf := c.engine.Add(lf)
	pf.SetName(c.engine.UniqueName("Lip"))
	lf.featName = pf.name
	return pf
}
