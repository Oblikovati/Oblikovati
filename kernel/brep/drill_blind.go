// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CutBlindCylindricalHole drills a flat-bottomed blind hole (K1b): the cylinder enters one
// planar face and stops at `depth` inside the material, so the result adds a circular hole in
// the entry face, a true cylinder wall, and a flat circular bottom disk (facing back toward
// the opening). The hole must stay entirely inside the part (the bottom and its rim interior),
// the entry face must be planar and perpendicular to the axis with the circle fitting inside,
// and every other face is copied unchanged (keeping its reference key). A conical drill point
// and a hole that exits/clips a face are later slices — those return an error here so the
// caller can fall back to the faceted boolean.
func CutBlindCylindricalHole(slab *topo.Body, base math.Point3, axisDir math.Vector3, radius, depth float64) (*topo.Body, error) {
	if radius <= 0 || depth <= 0 {
		return nil, fmt.Errorf("brep: blind hole needs radius>0 and depth>0, got r=%g depth=%g", radius, depth)
	}
	ua := unit(axisDir)
	if ua.LengthSquared() < 0.5 {
		return nil, fmt.Errorf("brep: drill axis direction is degenerate: %+v", axisDir)
	}
	copied, entry, err := classifyBlindDrill(slab, base, ua, radius)
	if err != nil {
		return nil, err
	}
	bottom := base.TranslateBy(ua.Scale(math.Scalar(depth)))
	if err := checkBlindFits(slab, entry, bottom, radius); err != nil {
		return nil, err
	}
	return assembleBlind(copied, entry, base, bottom, ua, radius)
}

// classifyBlindDrill finds the single planar entry face (perpendicular to the axis, with the
// base on its plane and the circle inside) and returns every other face as copied-unchanged.
func classifyBlindDrill(slab *topo.Body, base math.Point3, ua math.Vector3, radius float64) (copied []planarFace, entry planarFace, err error) {
	faces, ok := facesOf(slab)
	if !ok {
		return nil, planarFace{}, ErrNonPlanar
	}
	found := false
	for _, f := range faces {
		isEntry := float64(f.normal.Dot(ua)) < -1+1e-7 && stdmath.Abs(pierceParam(base, ua, f.plane)) < 1e-6 // tol:angular (cosine) + calibrated (pierce dist)
		if !isEntry {
			copied = append(copied, f)
			continue
		}
		if !circleInsideFace(base, f, radius) {
			return nil, planarFace{}, fmt.Errorf("brep: blind hole circle (r=%g) does not fit inside the entry face", radius)
		}
		if found {
			return nil, planarFace{}, fmt.Errorf("brep: ambiguous entry face for blind hole at %+v", base)
		}
		entry, found = f, true
	}
	if !found {
		return nil, planarFace{}, fmt.Errorf("brep: no planar entry face perpendicular to the drill axis at %+v", base)
	}
	return copied, entry, nil
}

// checkBlindFits verifies the hole bottom (centre and rim) stays strictly inside the part, so
// the blind pocket does not exit or clip another face.
func checkBlindFits(slab *topo.Body, entry planarFace, bottom math.Point3, radius float64) error {
	if !insideSolid(slab, bottom) {
		return fmt.Errorf("brep: blind hole bottom at %+v is outside the part (depth too large)", bottom)
	}
	u, v := entry.plane.UAxis.AsVector(), entry.plane.VAxis.AsVector()
	const samples = 8
	for i := 0; i < samples; i++ {
		a := 2 * stdmath.Pi * float64(i) / samples
		rim := bottom.TranslateBy(u.Scale(math.Scalar(radius * stdmath.Cos(a)))).TranslateBy(v.Scale(math.Scalar(radius * stdmath.Sin(a))))
		if !insideSolid(slab, rim) {
			return fmt.Errorf("brep: blind hole (r=%g) does not stay inside the part", radius)
		}
	}
	return nil
}

// assembleBlind welds the bore (entry hole + cylinder wall) and caps it with a flat bottom disk
// whose outward normal faces back toward the opening (−axis).
func assembleBlind(copied []planarFace, entry planarFace, base, bottom math.Point3, ua math.Vector3, radius float64) (*topo.Body, error) {
	bld, holeBot, err := blindBore(copied, entry, base, bottom, ua, radius)
	if err != nil {
		return nil, err
	}
	botPlane, err := geom.NewPlane(bottom, ua.Scale(-1))
	if err != nil {
		return nil, err
	}
	bld.AddFace(botPlane, topo.NewLineage(topo.Tok("brep", "holebottom", 0)), topo.OuterLoop(topo.Fwd(holeBot)))
	return bld.Build(), nil
}

// blindBore welds the copied + entry planar faces, holes the entry, and adds the cylinder wall.
// It returns the builder and the bottom rim edge so the caller can cap the bore (a flat disk for
// a drilled flat bottom, or a cone for a conical drill point).
func blindBore(copied []planarFace, entry planarFace, base, bottom math.Point3, ua math.Vector3, radius float64) (*topo.Builder, *topo.Edge, error) {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("brep", "drill", 0)))
	planar := append(append([]planarFace{}, copied...), entry)

	w := newWelder3()
	rings, edgeUse := weldPlanarFaces(w, planar)
	tv := make([]*topo.Vertex, len(w.points))
	for i, p := range w.points {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("brep", "vertex", i)))
	}
	lineEdges := buildEdges(bld, w.points, tv, edgeUse)
	holeEntry, holeBot, seam, cyl, err := buildHoleEdges(bld, []drillCap{{center: base}, {center: bottom}}, ua, radius)
	if err != nil {
		return nil, nil, err
	}

	entryIdx := len(planar) - 1
	for fi, f := range planar {
		specs := planarLoopSpecs(rings[fi], lineEdges)
		if fi == entryIdx {
			specs = append(specs, topo.InnerLoop(topo.Fwd(holeEntry))) // entry keeps its key (K1a)
		}
		bld.AddFace(f.plane, f.lineage, specs...)
	}
	// Hole wall: surface normal is outward-radial, so its material side faces the axis.
	bld.AddReversedFace(cyl, topo.NewLineage(topo.Tok("brep", "wall", 0)),
		topo.OuterLoop(topo.Rev(holeEntry), topo.Fwd(seam), topo.Rev(holeBot), topo.Rev(seam)))
	return bld, holeBot, nil
}
