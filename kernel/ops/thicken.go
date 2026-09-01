// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"bytes"
	"fmt"

	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// ThickenDirection is the side(s) a Thicken offsets a surface toward (Inventor's
// PartFeatureExtentDirection for Thicken): the +normal side, the −normal side, or half the
// thickness each way (#1876).
type ThickenDirection int

const (
	// ThickenSymmetric offsets ±t/2, keeping the surface centred (the pre-#1876 behaviour).
	ThickenSymmetric ThickenDirection = iota
	// ThickenPositive offsets 0…+t (all material on the +normal side).
	ThickenPositive
	// ThickenNegative offsets −t…0 (all material on the −normal side).
	ThickenNegative
)

// String returns the stable wire/serialization spelling of a thicken direction.
func (d ThickenDirection) String() string {
	switch d {
	case ThickenPositive:
		return "positive"
	case ThickenNegative:
		return "negative"
	default:
		return "symmetric"
	}
}

// ParseThickenDirection maps the wire spelling to a direction; "" defaults to positive (Inventor's
// kPositive default for Thicken). It reports false for an unknown spelling.
func ParseThickenDirection(s string) (ThickenDirection, bool) {
	switch s {
	case "", "positive":
		return ThickenPositive, true
	case "negative":
		return ThickenNegative, true
	case "symmetric":
		return ThickenSymmetric, true
	default:
		return ThickenSymmetric, false
	}
}

// slabOffsets returns the top/bottom offset distances for a direction and thickness.
func (d ThickenDirection) slabOffsets(t float64) (top, bottom float64) {
	switch d {
	case ThickenPositive:
		return t, 0
	case ThickenNegative:
		return 0, -t
	default:
		return t / 2, -t / 2
	}
}

// Thicken turns a planar surface (sheet) body into a solid of wall thickness t, symmetric about
// the surface — the whole-body default (see ThickenSolid for the directed / face-subset form).
func Thicken(surface *topo.Body, t float64) (*topo.Body, error) {
	return ThickenSolid(surface, t, ThickenSymmetric, nil, true)
}

// ThickenSolid turns (a subset of) a planar surface body into a solid of wall thickness t. Each
// selected face is copied to both offset sides (per dir) and the subset-boundary edges — those
// used by exactly one selected face — are closed with side walls when walls is true
// (Inventor's CreateVerticalSurfaces). A nil/empty faceKeys thickens every face. Curved or creased
// sheets (where per-face offsets don't meet) are a follow-up; it errors when the result is not a
// valid solid.
func ThickenSolid(surface *topo.Body, t float64, dir ThickenDirection, faceKeys [][]byte, walls bool) (*topo.Body, error) {
	if t <= 0 {
		return nil, fmt.Errorf("thicken: thickness %g must be > 0", t)
	}
	selected := selectedFaceSet(surface, faceKeys)
	if len(selected) == 0 {
		return nil, fmt.Errorf("thicken: no faces selected")
	}
	top, bottom := dir.slabOffsets(t)
	var loops []retopo.PlanarLoop
	for _, f := range surface.Faces() {
		if !selected[f.ID()] {
			continue
		}
		n := f.Geometry().NormalAt(0, 0)
		loops = append(loops, slabCaps(f, n, top, bottom)...)
		if walls {
			loops = append(loops, slabWalls(f, n, top, bottom, selected)...)
		}
	}
	body := retopo.BuildSolidFromLoops(loops)
	if r := Validate(body); !r.Valid || !body.IsSolid() {
		return nil, fmt.Errorf("thicken: result is not a valid solid %v", r.Issues)
	}
	return body, nil
}

// selectedFaceSet maps the chosen face ids (all faces when faceKeys is empty).
func selectedFaceSet(surface *topo.Body, faceKeys [][]byte) map[uint64]bool {
	set := map[uint64]bool{}
	if len(faceKeys) == 0 {
		for _, f := range surface.Faces() {
			set[f.ID()] = true
		}
		return set
	}
	for _, f := range surface.Faces() {
		for _, k := range faceKeys {
			if bytes.Equal(f.ReferenceKey(), k) {
				set[f.ID()] = true
			}
		}
	}
	return set
}

// slabCaps returns the top (offset +top, same winding) and bottom (offset +bottom, reversed)
// copies of a face — the two outer skins of the slab.
func slabCaps(f *topo.Face, n math.Vector3, top, bottom float64) []retopo.PlanarLoop {
	tc := retopo.PlanarLoop{Normal: n}
	bc := retopo.PlanarLoop{Normal: n.Scale(-1)}
	for _, l := range f.Loops() {
		pts := loopPoints3(l)
		tc.Rings = append(tc.Rings, offsetRing(pts, n, top))
		bc.Rings = append(bc.Rings, probe.ReversedPoints(offsetRing(pts, n, bottom)))
	}
	return []retopo.PlanarLoop{tc, bc}
}

// slabWalls returns a side-wall quad for each subset-boundary edge of the face — an edge with
// exactly one use on a selected face — connecting the top and bottom skins.
func slabWalls(f *topo.Face, n math.Vector3, top, bottom float64, selected map[uint64]bool) []retopo.PlanarLoop {
	var walls []retopo.PlanarLoop
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if selectedUses(u.Edge(), selected) != 1 {
				continue // interior to the selected subset — the offset skins already meet here
			}
			a, b := useEndpoints(u)
			at, ab := a.TranslateBy(n.Scale(top)), a.TranslateBy(n.Scale(bottom))
			bt, bb := b.TranslateBy(n.Scale(top)), b.TranslateBy(n.Scale(bottom))
			wn := outwardWall(a.VectorTo(b), n)
			walls = append(walls, retopo.PlanarLoop{Normal: wn, Rings: [][]math.Point3{{at, ab, bb, bt}}})
		}
	}
	return walls
}

// selectedUses counts how many of an edge's uses belong to a selected face.
func selectedUses(e *topo.Edge, selected map[uint64]bool) int {
	n := 0
	for _, u := range e.Uses() {
		if f := u.Loop().Face(); f != nil && selected[f.ID()] {
			n++
		}
	}
	return n
}

// useEndpoints returns an edge use's start and end points in its traversal direction.
func useEndpoints(u *topo.EdgeUse) (math.Point3, math.Point3) {
	s, e := u.Edge().StartVertex(), u.Edge().EndVertex()
	if u.Reversed() {
		s, e = e, s
	}
	return s.Point(), e.Point()
}

// outwardWall returns the wall's outward Normal: perpendicular to the edge and the face
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
