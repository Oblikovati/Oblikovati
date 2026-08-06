// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Where a hole drills — Inventor's HolePlacementTypeEnum (Oblikovati#1861).
//
// A hole used to be pinned to ONE absolute model point. That is not how holes are authored: they
// are placed AGAINST something — a sketch's centre points, a circular edge to sit concentric with,
// two edges to measure off, a work point — and the whole value of the placement is that editing
// what it was placed against MOVES the hole. Freezing the resolved coordinate loses exactly that,
// which is why an imported sketch-driven or concentric hole stopped tracking its driving geometry.
//
// So a placement is not a coordinate: it is a rule, RE-RESOLVED against the running body every
// recompute. One placement may yield MANY bores (a sketch of six centre points is one hole feature
// with six holes, as it is in Inventor), which is also why sites are a slice.

// HoleSites is one placement's answer for the current body: where to start each bore and which way
// to drill. Every site shares the direction — a hole feature drills one way.
type HoleSites struct {
	Centers []math.Point3
	Into    math.UnitVector3
}

// HolePlacement is the rule locating a hole feature's bores. Implementations resolve against the
// running body, so a lost reference is an error the engine turns into feature health.
type HolePlacement interface {
	// Kind is the placement's recipe name — "sketch", "linear", "concentric" or "point".
	Kind() string
	// Sites locates the bores on the running body.
	Sites(body *topo.Body) (HoleSites, error)
}

// SketchHolePlacement drills one bore per CENTRE POINT of a sketch — Inventor's most common hole,
// and the only placement that makes one feature into many holes. Plain sketch points are ignored:
// a curve's endpoints live in the same collection, so drilling every point would bore a hole at
// each corner of a rectangle. Flipped drills along the sketch normal instead of into it.
type SketchHolePlacement struct {
	Sketch  *sketch.Sketch
	Flipped bool
}

// Kind implements [HolePlacement].
func (p SketchHolePlacement) Kind() string { return "sketch" }

// Sites maps the sketch's centre points into model space. A sketch with none marked is a caller
// error naming what to fix, not an empty (silently no-op) hole feature.
func (p SketchHolePlacement) Sites(*topo.Body) (HoleSites, error) {
	if p.Sketch == nil {
		return HoleSites{}, errors.New("hole: sketch placement has no sketch")
	}
	centers := centrePointsOf(p.Sketch)
	if len(centers) == 0 {
		return HoleSites{}, fmt.Errorf("hole: sketch placement found no centre points among the sketch's %d points "+
			"(mark the drill positions as centre points — a curve's endpoints are points too)", p.Sketch.Points().Count())
	}
	return HoleSites{Centers: centers, Into: sketchDrillDir(p.Sketch, p.Flipped)}, nil
}

// centrePointsOf collects the sketch's hole-centre markers in model space.
func centrePointsOf(sk *sketch.Sketch) []math.Point3 {
	var out []math.Point3
	for i := 0; i < sk.Points().Count(); i++ {
		if pt := sk.Points().Item(i); pt.IsCenterPoint() {
			out = append(out, sk.Plane().ToModel(pt.Position()))
		}
	}
	return out
}

// sketchDrillDir is the drilling direction of a sketch placement: INTO the sketch plane (against
// its normal), which is what a sketch drawn on a face wants, or along it when flipped.
func sketchDrillDir(sk *sketch.Sketch, flipped bool) math.UnitVector3 {
	n := sk.Plane().Normal()
	if flipped {
		return n
	}
	return n.Negate()
}

// ConcentricHolePlacement centres a bore on the axis of a circular edge (or a cylindrical face's
// rim) — a boss to be bored, a counterbored pad to be tapped. The bore starts on the placement
// face, so the reference's centre is dropped onto that face's plane rather than used as-is.
type ConcentricHolePlacement struct {
	Face     HoleFaceRef // the planar face the bore starts on
	RefEdge  []byte      // lineage key of the circular edge whose axis the bore takes
	GeomEdge *topo.GeometricEdgeRef
}

// Kind implements [HolePlacement].
func (p ConcentricHolePlacement) Kind() string { return "concentric" }

// Sites resolves the reference circle's centre onto the placement face.
func (p ConcentricHolePlacement) Sites(body *topo.Body) (HoleSites, error) {
	face, into, err := p.Face.resolve(body)
	if err != nil {
		return HoleSites{}, err
	}
	centre, err := circularEdgeCentre(body, p.RefEdge, p.GeomEdge)
	if err != nil {
		return HoleSites{}, err
	}
	return HoleSites{Centers: []math.Point3{projectOntoFacePlane(centre, face)}, Into: into}, nil
}

// LinearHolePlacement locates a bore by two perpendicular offsets from two reference edges of the
// placement face — the dimensioned hole of a machining drawing. The offsets run TOWARD the face's
// interior, so the point they name is on the face rather than off one of its corners.
type LinearHolePlacement struct {
	Face             HoleFaceRef
	Edge1, Edge2     []byte
	GeomEdge1        *topo.GeometricEdgeRef
	GeomEdge2        *topo.GeometricEdgeRef
	Offset1, Offset2 func() float64
}

// Kind implements [HolePlacement].
func (p LinearHolePlacement) Kind() string { return "linear" }

// Sites intersects the two offset lines on the placement face.
func (p LinearHolePlacement) Sites(body *topo.Body) (HoleSites, error) {
	face, into, err := p.Face.resolve(body)
	if err != nil {
		return HoleSites{}, err
	}
	first, err := offsetLineOnFace(body, face, p.Edge1, p.GeomEdge1, callOrZero(p.Offset1), "edge1")
	if err != nil {
		return HoleSites{}, err
	}
	second, err := offsetLineOnFace(body, face, p.Edge2, p.GeomEdge2, callOrZero(p.Offset2), "edge2")
	if err != nil {
		return HoleSites{}, err
	}
	centre, err := crossOnPlane(first, second, into.AsVector())
	if err != nil {
		return HoleSites{}, err
	}
	return HoleSites{Centers: []math.Point3{centre}, Into: into}, nil
}

// PointHolePlacement drills at a work point along a work axis — the placement that needs no face at
// all, for a bore through a part at an angle nothing else names.
type PointHolePlacement struct {
	Point *WorkPoint
	Axis  *WorkAxis
	// Flipped drills against the axis direction rather than along it.
	Flipped bool
}

// Kind implements [HolePlacement].
func (p PointHolePlacement) Kind() string { return "point" }

// Sites reads the work point and axis directly: with no placement face there is nothing on the body
// to resolve against, which is the point of this placement.
func (p PointHolePlacement) Sites(*topo.Body) (HoleSites, error) {
	if p.Point == nil || p.Axis == nil {
		return HoleSites{}, errors.New("hole: on-point placement needs both a work point and a work axis to drill along")
	}
	dir := p.Axis.Direction()
	if p.Flipped {
		dir = dir.Negate()
	}
	return HoleSites{Centers: []math.Point3{p.Point.Point()}, Into: dir}, nil
}
