// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "math"

// Brep is the analytic B-rep extracted from a PmBRepSegment: the solid's vertices
// and its planar faces. Curved (cone/cylinder) faces are recorded separately for a
// later tessellation pass; this milestone reconstructs planar solids exactly.
type Brep struct {
	Points [][3]float64 // vertex positions, in cm
	Planes []Plane      // planar faces (live topology only)
	Cones  []Cone       // cone/cylinder faces (axis + radius), for later tessellation
}

// Plane is a planar face: a point on the plane and its outward-ish normal.
type Plane struct {
	Origin [3]float64
	Normal [3]float64
}

// Cone is a cone/cylinder face: axis root, axis direction, and radius (cm). A zero
// half-angle sine (recorded upstream) means a cylinder.
type Cone struct {
	Root   [3]float64
	Axis   [3]float64
	Radius float64
}

// ExtractBrep parses the SAB and returns the vertices and the live faces' surfaces.
func ExtractBrep(seg []byte) Brep {
	recs := ParseSAB(seg)
	b := Brep{Points: pointPositions(recs)}
	live := liveFaceSet(recs)
	for i, r := range recs {
		if r.Name != "face" || !live[i] {
			continue
		}
		addFaceSurface(&b, recs, r)
	}
	return b
}

// pointPositions returns every ACIS point record's position (the vertex geometry).
func pointPositions(recs []Record) [][3]float64 {
	var out [][3]float64
	for _, r := range recs {
		if r.Name != "point" {
			continue
		}
		if ps := r.Positions(); len(ps) > 0 {
			out = append(out, ps[0])
		}
	}
	return out
}

// addFaceSurface resolves a face's surface record and appends it as a plane or cone.
func addFaceSurface(b *Brep, recs []Record, face Record) {
	for _, ref := range face.Refs() {
		s, ok := recAt(recs, ref)
		if !ok {
			continue
		}
		switch {
		case s.Name == "plane-surface":
			if p, v := s.Positions(), s.Vectors(); len(p) > 0 && len(v) > 0 {
				b.Planes = append(b.Planes, Plane{Origin: p[0], Normal: v[0]})
			}
		case s.Name == "cone-surface":
			if p, v := s.Positions(), s.Vectors(); len(p) > 0 && len(v) > 1 {
				b.Cones = append(b.Cones, Cone{Root: p[0], Axis: v[0], Radius: vecLen(v[1])})
			}
		}
	}
}

// liveFaceSet returns the face record indices reachable via each lump's face-next
// chain (entity-ref slot 0), excluding faces retained in ACIS history (delta_state).
func liveFaceSet(recs []Record) map[int]bool {
	name := func(i int) string {
		if r, ok := recAt(recs, int32(i)); ok {
			return r.Name
		}
		return ""
	}
	live := map[int]bool{}
	for _, r := range recs {
		if r.Name != "lump" {
			continue
		}
		head := firstRefTo(recs, r, "face")
		for head >= 0 && name(head) == "face" && !live[head] {
			live[head] = true
			refs := recs[head].Refs()
			if len(refs) == 0 || refs[0] < 0 || name(int(refs[0])) != "face" {
				break
			}
			head = int(refs[0])
		}
	}
	if len(live) == 0 { // no lump/chain: fall back to every face record
		for i, r := range recs {
			if r.Name == "face" {
				live[i] = true
			}
		}
	}
	return live
}

func firstRefTo(recs []Record, r Record, target string) int {
	for _, ref := range r.Refs() {
		if s, ok := recAt(recs, ref); ok && s.Name == target {
			return int(ref)
		}
	}
	return -1
}

func recAt(recs []Record, i int32) (Record, bool) {
	if i < 0 || int(i) >= len(recs) {
		return Record{}, false
	}
	return recs[i], true
}

func vecLen(v [3]float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}
