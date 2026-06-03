// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// Thicken turns a planar surface (sheet) body into a solid of wall thickness t: each face is
// copied to both sides (offset ±t/2 along its normal) and the free-boundary edges (used by a
// single face) are closed with side walls. A single planar patch becomes a slab; a connected
// planar sheet becomes a thick shell. Curved or creased sheets (where per-face offsets don't
// meet) are a follow-up. It errors when the result is not a valid solid.
func Thicken(surface *topo.Body, t float64) (*topo.Body, error) {
	if t <= 0 {
		return nil, fmt.Errorf("thicken: thickness %g must be > 0", t)
	}
	half := t / 2
	var loops []ploop
	for _, f := range surface.Faces() {
		n := f.Geometry().NormalAt(0, 0)
		loops = append(loops, slabCaps(f, n, half)...)
		loops = append(loops, slabWalls(f, n, half)...)
	}
	body := buildSolidFromLoops(loops)
	if r := Validate(body); !r.Valid || !body.IsSolid() {
		return nil, fmt.Errorf("thicken: result is not a valid solid %v", r.Issues)
	}
	return body, nil
}

// slabCaps returns the top (offset +half, same winding) and bottom (offset −half, reversed)
// copies of a face — the two outer skins of the slab.
func slabCaps(f *topo.Face, n math.Vector3, half float64) []ploop {
	top := ploop{normal: n}
	bottom := ploop{normal: n.Scale(-1)}
	for _, l := range f.Loops() {
		pts := loopPoints3(l)
		top.rings = append(top.rings, offsetRing(pts, n, half))
		bottom.rings = append(bottom.rings, reverse3(offsetRing(pts, n, -half)))
	}
	return []ploop{top, bottom}
}

// slabWalls returns a side-wall quad for each free-boundary edge of the face (an edge used by
// a single face), connecting the top and bottom skins.
func slabWalls(f *topo.Face, n math.Vector3, half float64) []ploop {
	var walls []ploop
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if len(u.Edge().Uses()) != 1 {
				continue // shared (interior) edge — the offset skins already meet here
			}
			a, b := useEndpoints(u)
			at, ab := a.TranslateBy(n.Scale(half)), a.TranslateBy(n.Scale(-half))
			bt, bb := b.TranslateBy(n.Scale(half)), b.TranslateBy(n.Scale(-half))
			wn := outwardWall(a.VectorTo(b), n)
			walls = append(walls, ploop{normal: wn, rings: [][]math.Point3{{at, ab, bb, bt}}})
		}
	}
	return walls
}

// useEndpoints returns an edge use's start and end points in its traversal direction.
func useEndpoints(u *topo.EdgeUse) (math.Point3, math.Point3) {
	s, e := u.Edge().StartVertex(), u.Edge().EndVertex()
	if u.Reversed() {
		s, e = e, s
	}
	return s.Point(), e.Point()
}

// outwardWall returns the wall's outward normal: perpendicular to the edge and the face
// normal, pointing away from the face interior (edge × normal for a CCW loop).
func outwardWall(edge, n math.Vector3) math.Vector3 {
	w, err := math.UnitVector3FromVector(edge.Cross(n))
	if err != nil {
		return n
	}
	return w.AsVector()
}

// offsetRing returns the loop points shifted by d along n.
func offsetRing(pts []math.Point3, n math.Vector3, d float64) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[i] = p.TranslateBy(n.Scale(d))
	}
	return out
}

// loopPoints3 returns a loop's vertices in traversal order.
func loopPoints3(l *topo.Loop) []math.Point3 {
	var pts []math.Point3
	for _, u := range l.EdgeUses() {
		s, _ := useEndpoints(u)
		pts = append(pts, s)
	}
	return pts
}
