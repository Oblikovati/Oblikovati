// SPDX-License-Identifier: GPL-2.0-only

// This file has no cgo build tag: the ViewCube's geometry — projecting the cube onto the
// camera's screen plane and hit-testing the cursor against its 26 regions — is pure Go, so
// it is unit-tested without the native stack (like axis_gizmo.go / navigate.go). The cgo
// side (viewcube_draw.go) anchors and paints it in a tile's top-right corner.
//
// Convention: Z-up CAD. TOP=+Z, BOTTOM=−Z, FRONT=−Y, BACK=+Y, RIGHT=+X, LEFT=−X — so the
// XY plane is the ground and the default top-down view looks along −Z. A region's "from"
// direction is the unit vector the camera looks FROM (eye = center + dir·distance).
package ui

import (
	stdmath "math"
	"sort"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

// cubeBasis is the camera's screen basis expressed in the cube's local frame: the camera
// right/up/forward rotated by the document's ViewCube orientation (Set Front). With the
// identity orientation it equals camBasis, so the cube math below stays axis-aligned and a
// front redefinition is a pure pre-rotation here — nothing else changes.
func cubeBasis(cam scene.Camera, o doc.CubeOrient) (right, up, fwd math.Vector3) {
	r, u, f := camBasis(cam)
	return o.ToLocal(r), o.ToLocal(u), o.ToLocal(f)
}

// edgeZone is the fraction of a face (from its edge inward) that counts as an edge/corner
// region rather than the face center, when classifying a hit. 0.5 splits each axis into
// outer-third-ish edge bands vs the central face.
const edgeZone = 0.5

// RegionKind is whether a ViewCube region is a face, an edge, or a corner.
type RegionKind int

const (
	RegionFace RegionKind = iota
	RegionEdge
	RegionCorner
)

// Region is one of the cube's 26 clickable zones, identified by its sign on each axis
// (each of x,y,z ∈ {-1,0,1}, not all zero). Faces have one non-zero axis, edges two,
// corners three. Label names the six faces.
type Region struct {
	X, Y, Z int
	Kind    RegionKind
	Label   string
}

var faceLabels = map[[3]int]string{
	{0, 0, 1}:  "TOP",
	{0, 0, -1}: "BOTTOM",
	{0, -1, 0}: "FRONT",
	{0, 1, 0}:  "BACK",
	{1, 0, 0}:  "RIGHT",
	{-1, 0, 0}: "LEFT",
}

// allRegions enumerates the 26 regions: every {-1,0,1}³ except the origin.
func allRegions() []Region {
	var rs []Region
	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			for z := -1; z <= 1; z++ {
				n := abs(x) + abs(y) + abs(z)
				if n == 0 {
					continue
				}
				r := Region{X: x, Y: y, Z: z, Kind: RegionKind(n - 1)}
				if n == 1 {
					r.Label = faceLabels[[3]int{x, y, z}]
				}
				rs = append(rs, r)
			}
		}
	}
	return rs
}

// dir is the unit direction the camera looks from for this region.
func (r Region) dir() math.Vector3 {
	return normVec(math.V3(float64(r.X), float64(r.Y), float64(r.Z)))
}

// up is a sensible up vector for the region's snapped view: world +Z, except for the
// near-vertical TOP/BOTTOM views (where +Z is the view axis) it falls back to +Y.
func (r Region) up() math.Vector3 {
	d := r.dir()
	if stdmath.Abs(d.Dot(math.V3(0, 0, 1))) > 0.9 {
		return math.V3(0, 1, 0)
	}
	return math.V3(0, 0, 1)
}

// SnapCamera returns cur reframed to look at center from this region's direction, keeping
// the current eye→target distance (zoom). The eye is forced onto the region's side (unlike
// scene.Camera.Facing, which preserves the current side).
func (r Region) SnapCamera(cur scene.Camera, center math.Point3, o doc.CubeOrient) scene.Camera {
	dist := cur.Eye.DistanceTo(cur.Target)
	if dist < 1 {
		dist = 1
	}
	cur.Target = center
	cur.Eye = center.TranslateBy(o.ToWorld(r.dir()).Scale(dist)) // local dir → world via the front orientation
	cur.Up = normVec(o.ToWorld(r.up()))
	return cur
}

// cubeCorner is one projected cube vertex: its screen offset from the cube center (px, y
// down) and its camera-forward depth (larger ⇒ farther), for back-to-front painting.
type cubeCorner struct {
	sx, sy float32
	depth  float64
}

// camBasis returns the camera's orthonormal screen basis: right, up (screen-up), forward.
func camBasis(cam scene.Camera) (right, up, fwd math.Vector3) {
	fwd = normVec(cam.Forward())
	right = normVec(fwd.Cross(cam.Up))
	up = right.Cross(fwd) // unit (right ⟂ fwd, both unit)
	return right, up, fwd
}

// project maps a unit-cube point p ([-1,1]³, world axes) onto the gizmo screen plane: x =
// p·right, y = −p·up (y-down, matching renderer.Project), scaled to radius px; depth = p·fwd.
func project(p math.Vector3, right, up, fwd math.Vector3, radius float32) cubeCorner {
	return cubeCorner{
		sx:    float32(p.Dot(right)) * radius,
		sy:    float32(-p.Dot(up)) * radius,
		depth: p.Dot(fwd),
	}
}

// HitTest returns the region under a cursor at (localX, localY) px relative to the cube
// center, for a cube drawn at the given radius and camera, or nil for a miss. It casts a
// ray (along the camera forward) from the cursor into the world-axis-aligned unit cube and
// classifies the entry point into a face/edge/corner zone.
func HitTest(localX, localY, radius float32, cam scene.Camera, o doc.CubeOrient) *Region {
	right, up, fwd := cubeBasis(cam, o)
	sx, sy := float64(localX/radius), float64(localY/radius)
	if stdmath.Abs(sx) > 1.6 || stdmath.Abs(sy) > 1.6 { // outside the cube's projected reach
		return nil
	}
	// Cursor → world ray. screenX = p·right, screenY = −p·up ⇒ the in-plane point is
	// right·sx − up·sy; the ray runs along the view forward.
	origin := right.Scale(sx).Add(up.Scale(-sy)).Add(fwd.Scale(-3)) // start outside the cube
	hit, ok := rayUnitCube(origin, fwd)
	if !ok {
		return nil
	}
	return classify(hit)
}

// rayUnitCube intersects a ray with the axis-aligned cube [-1,1]³ (slab method), returning
// the near entry point.
func rayUnitCube(o, d math.Vector3) (math.Vector3, bool) {
	tmin, tmax := stdmath.Inf(-1), stdmath.Inf(1)
	oc := []float64{o.X, o.Y, o.Z}
	dc := []float64{d.X, d.Y, d.Z}
	for i := 0; i < 3; i++ {
		if stdmath.Abs(dc[i]) < 1e-12 {
			if oc[i] < -1 || oc[i] > 1 {
				return math.Vector3{}, false
			}
			continue
		}
		t1 := (-1 - oc[i]) / dc[i]
		t2 := (1 - oc[i]) / dc[i]
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tmin {
			tmin = t1
		}
		if t2 < tmax {
			tmax = t2
		}
		if tmin > tmax {
			return math.Vector3{}, false
		}
	}
	return o.Add(d.Scale(tmin)), true
}

// classify maps an entry point on the unit cube to a face/edge/corner region: the axis it
// is ±1 on is that face's sign; the other axes count as ±1 when the coordinate is past the
// edge zone, promoting the hit to an edge (one extra) or corner (two extra).
func classify(p math.Vector3) *Region {
	c := [3]float64{p.X, p.Y, p.Z}
	s := zoneSigns(c, faceAxis(c))
	if s == [3]int{0, 0, 0} {
		return nil
	}
	n := abs(s[0]) + abs(s[1]) + abs(s[2])
	r := Region{X: s[0], Y: s[1], Z: s[2], Kind: RegionKind(n - 1)}
	if n == 1 {
		r.Label = faceLabels[s]
	}
	return &r
}

// faceAxis returns the index (0/1/2) of the coordinate closest to ±1 — the entry face's axis.
func faceAxis(c [3]float64) int {
	face := 0
	for i := 1; i < 3; i++ {
		if stdmath.Abs(c[i]) > stdmath.Abs(c[face]) {
			face = i
		}
	}
	return face
}

// zoneSigns assigns each axis its sign on the entry face (always) and where the coordinate
// is past the edge zone (promoting a face hit to an edge/corner), else 0.
func zoneSigns(c [3]float64, face int) [3]int {
	var s [3]int
	for i := 0; i < 3; i++ {
		if i == face || stdmath.Abs(c[i]) >= edgeZone {
			s[i] = sign(c[i])
		}
	}
	return s
}

// cubeFace is a projected, visible cube face for painting: its four screen-corner offsets
// (in draw order) and depth (face-center forward component) for back-to-front sorting.
type cubeFace struct {
	region Region
	corner [4]cubeCorner
	depth  float64
}

// faceDef is a unit-cube face: its outward-normal axis (0/1/2) + sign, and the four corners
// in CCW order.
type faceDef struct {
	axis int
	sign int
	quad [4][3]float64
}

// cubeFaceDefs are the six unit-cube faces (normal axis/sign + CCW corners).
var cubeFaceDefs = []faceDef{
	{2, 1, [4][3]float64{{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1}}},      // TOP +Z
	{2, -1, [4][3]float64{{-1, 1, -1}, {1, 1, -1}, {1, -1, -1}, {-1, -1, -1}}}, // BOTTOM -Z
	{1, -1, [4][3]float64{{-1, -1, -1}, {1, -1, -1}, {1, -1, 1}, {-1, -1, 1}}}, // FRONT -Y
	{1, 1, [4][3]float64{{1, 1, -1}, {-1, 1, -1}, {-1, 1, 1}, {1, 1, 1}}},      // BACK +Y
	{0, 1, [4][3]float64{{1, -1, -1}, {1, 1, -1}, {1, 1, 1}, {1, -1, 1}}},      // RIGHT +X
	{0, -1, [4][3]float64{{-1, 1, -1}, {-1, -1, -1}, {-1, -1, 1}, {-1, 1, 1}}}, // LEFT -X
}

// normal is the face's outward unit normal in cube-local space.
func (d faceDef) normal() math.Vector3 {
	switch d.axis {
	case 0:
		return math.V3(float64(d.sign), 0, 0)
	case 1:
		return math.V3(0, float64(d.sign), 0)
	default:
		return math.V3(0, 0, float64(d.sign))
	}
}

// project returns the face projected for painting, or ok=false when it is back-facing (its
// normal points away from the viewer along the cube-local forward).
func (d faceDef) project(right, up, fwd math.Vector3, radius float32) (cubeFace, bool) {
	nrm := d.normal()
	if nrm.Dot(fwd) >= -1e-6 {
		return cubeFace{}, false
	}
	f := cubeFace{depth: nrm.Dot(fwd)}
	key := [3]int{int(nrm.X), int(nrm.Y), int(nrm.Z)}
	f.region = Region{X: key[0], Y: key[1], Z: key[2], Kind: RegionFace, Label: faceLabels[key]}
	for i, q := range d.quad {
		f.corner[i] = project(math.V3(q[0], q[1], q[2]), right, up, fwd, radius)
	}
	return f, true
}

// visibleFaces returns the cube's front-facing faces (those whose outward normal points
// toward the viewer), projected and sorted back-to-front, for the painter.
func visibleFaces(cam scene.Camera, o doc.CubeOrient, radius float32) []cubeFace {
	right, up, fwd := cubeBasis(cam, o)
	var faces []cubeFace
	for _, d := range cubeFaceDefs {
		if f, ok := d.project(right, up, fwd, radius); ok {
			faces = append(faces, f)
		}
	}
	sort.SliceStable(faces, func(i, j int) bool { return faces[i].depth > faces[j].depth })
	return faces
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func sign(v float64) int {
	if v < 0 {
		return -1
	}
	return 1
}
