// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/seq"
)

// This file serializes a part's USER work features (planes/axes/points) into the
// recipe. The origin coordinate system is regenerated, never serialized — only its
// stable references are recorded. User features are stored in global creation order so
// a feature that references an earlier one restores after it; their definitions name
// the geometry they are built on by WorkRef (origin well-known keys, or earlier user
// features by position), which re-resolve on recompute.

// WorkFeatureData is the recipe form of one user work feature: which collection it
// belongs to, its definition kind, the references it is built on, and any parameter.
type WorkFeatureData struct {
	Collection   string      `yaml:"collection"` // plane | axis | point
	Kind         string      `yaml:"kind"`
	Seq          uint64      `yaml:"seq,omitempty"`          // global creation stamp; see model/seq
	Construction bool        `yaml:"construction,omitempty"` // hidden, consumer-tied datum (#1849)
	Deleted      bool        `yaml:"deleted,omitempty"`      // tombstoned; slot kept for ref stability (#1855)
	Flipped      bool        `yaml:"flipped,omitempty"`      // plane normal reversed by FlipNormal (#1851)
	AutoResize   bool        `yaml:"autoResize,omitempty"`   // plane displayed size tracks the component box (#1851)
	Grounded     bool        `yaml:"grounded,omitempty"`     // plane grounded flag (#1851)
	SizeCorners  [][]float64 `yaml:"sizeCorners,omitempty"`  // explicit displayed-rectangle corners [x,y,z] (#1851)
	Solution     []float64   `yaml:"solution,omitempty"`     // tangent/bisector proximity/quadrant point [x,y,z] (#1844)
	Refs         []string    `yaml:"refs,omitempty"`
	Offset       float64     `yaml:"offset,omitempty"`   // plane-offset
	Angle        float64     `yaml:"angle,omitempty"`    // line-plane-angle
	Position     []float64   `yaml:"position,omitempty"` // point position / fixed-frame origin [x,y,z]
	XAxis        []float64   `yaml:"xaxis,omitempty"`    // fixed-frame X axis [x,y,z]
	YAxis        []float64   `yaml:"yaxis,omitempty"`    // fixed-frame Y axis [x,y,z]
	CloudID      string      `yaml:"cloud,omitempty"`    // point-cloud-fit: the source cloud's id (provenance, #645)
}

// MarshalWork projects the user work features into the recipe, in creation order.
func MarshalWork(g *WorkGeometry) ([]WorkFeatureData, error) {
	out := make([]WorkFeatureData, 0, len(g.userSeq))
	for i, e := range g.userSeq {
		d, err := serializeWorkFeature(g, e)
		if err != nil {
			return nil, fmt.Errorf("work feature %d (%s): %w", i, e.collection, err)
		}
		out = append(out, d)
	}
	return out, nil
}

// serializeWorkFeature encodes one user work feature: its definition (references, scalars, solution
// point) plus the creation stamp and lifecycle/display flags shared across collections (#132, #1849,
// #1855) and — for a plane — its flip/auto-resize/grounded/size display state (#1851).
func serializeWorkFeature(g *WorkGeometry, e userEntry) (WorkFeatureData, error) {
	switch e.collection {
	case "plane":
		w := g.planes.Item(e.index)
		d, err := serializePlaneDef(w.def)
		if err != nil {
			return WorkFeatureData{}, err
		}
		d.Seq, d.Construction, d.Deleted = w.seq, w.construction, w.deleted
		d.Flipped, d.AutoResize, d.Grounded = w.flipped, w.autoResize, w.grounded
		if w.hasSize {
			d.SizeCorners = [][]float64{p3Slice(w.sizeC1), p3Slice(w.sizeC2)}
		}
		return d, nil
	case "axis":
		w := g.axes.Item(e.index)
		d, err := serializeAxisDef(w.def)
		if err != nil {
			return WorkFeatureData{}, err
		}
		d.Seq, d.Construction, d.Deleted = w.seq, w.construction, w.deleted
		return d, nil
	case "point":
		w := g.points.Item(e.index)
		d, err := serializePointDef(w.def)
		if err != nil {
			return WorkFeatureData{}, err
		}
		d.Seq, d.Construction, d.Deleted = w.seq, w.construction, w.deleted
		return d, nil
	default:
		return WorkFeatureData{}, fmt.Errorf("unknown work collection %q", e.collection)
	}
}

// ApplyWork rebuilds the user work features onto g (which already holds the origin
// frame), in order, resolving each one's references as it goes.
func ApplyWork(g *WorkGeometry, data []WorkFeatureData) error {
	for i, d := range data {
		if err := restoreWorkFeature(g, d); err != nil {
			return fmt.Errorf("work feature %d (%s/%s): %w", i, d.Collection, d.Kind, err)
		}
		restoreWorkSeq(g, d)
		restoreWorkFlags(g, d)
	}
	return nil
}

// restoreWorkFlags re-applies the just-restored work feature's lifecycle flags (construction/
// deleted) — the Add* call above created it live and visible — and, for a plane, its flip/
// auto-resize/grounded/size display state. The restored feature is the last in its collection. A
// deleted datum is rebuilt as a tombstone so its slot holds and surviving datums keep their
// positional references; a flipped plane re-applies its normal flip on the next recompute (#1849,
// #1855, #1851).
func restoreWorkFlags(g *WorkGeometry, d WorkFeatureData) {
	switch d.Collection {
	case "plane":
		w := g.planes.Item(g.planes.Count() - 1)
		w.construction, w.deleted = d.Construction, d.Deleted
		w.flipped, w.autoResize, w.grounded = d.Flipped, d.AutoResize, d.Grounded
		if len(d.SizeCorners) == 2 {
			if c1, err := point3From(d.SizeCorners[0], "plane size corner 1"); err == nil {
				if c2, err := point3From(d.SizeCorners[1], "plane size corner 2"); err == nil {
					w.sizeC1, w.sizeC2, w.hasSize = c1, c2, true
				}
			}
		}
	case "axis":
		w := g.axes.Item(g.axes.Count() - 1)
		w.construction, w.deleted = d.Construction, d.Deleted
	case "point":
		w := g.points.Item(g.points.Count() - 1)
		w.construction, w.deleted = d.Construction, d.Deleted
	}
}

// restoreWorkSeq pins the just-restored work feature's creation stamp to its saved value
// (the Add* call above gave it a fresh one), so a reopened document keeps the original
// sketch/feature/work interleaving. The restored feature is the last in its collection.
func restoreWorkSeq(g *WorkGeometry, d WorkFeatureData) {
	switch d.Collection {
	case "plane":
		seq.Restore(&g.planes.Item(g.planes.Count()-1).seq, d.Seq)
	case "axis":
		seq.Restore(&g.axes.Item(g.axes.Count()-1).seq, d.Seq)
	case "point":
		seq.Restore(&g.points.Item(g.points.Count()-1).seq, d.Seq)
	}
}

func restoreWorkFeature(g *WorkGeometry, d WorkFeatureData) error {
	switch d.Collection {
	case "plane":
		return restorePlaneFeature(g.WorkPlanes(), d)
	case "axis":
		return restoreAxisFeature(g.WorkAxes(), d)
	case "point":
		return restorePointFeature(g.WorkPoints(), d)
	default:
		return fmt.Errorf("unknown work collection %q", d.Collection)
	}
}

// allWorkRefs casts every persisted reference string to a WorkRef — the variable-arity counterpart
// of workRefs, for kinds like centroid that take an unbounded edge list.
func allWorkRefs(refs []string) []WorkRef {
	out := make([]WorkRef, len(refs))
	for i, r := range refs {
		out[i] = WorkRef(r)
	}
	return out
}

// refStrings renders work references as their string form for YAML.
func refStrings(refs []WorkRef) []string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = string(r)
	}
	return out
}

// workRefs requires exactly n references, converting them from their string form.
func workRefs(refs []string, n int) ([]WorkRef, error) {
	if len(refs) != n {
		return nil, fmt.Errorf("expected %d references, got %d", n, len(refs))
	}
	out := make([]WorkRef, n)
	for i, r := range refs {
		out[i] = WorkRef(r)
	}
	return out, nil
}

// unitSlice renders a unit vector as its [x,y,z] components for YAML.
func unitSlice(u math.UnitVector3) []float64 {
	v := u.AsVector()
	return []float64{float64(v.X), float64(v.Y), float64(v.Z)}
}

// p3Slice renders a point as its [x,y,z] components for YAML.
func p3Slice(p math.Point3) []float64 {
	return []float64{float64(p.X), float64(p.Y), float64(p.Z)}
}

// solutionPoint returns the recipe's tangent/bisector proximity/quadrant point (and true) when one
// was recorded, so restore rebuilds the plane with the same solution side (#1844).
func solutionPoint(d WorkFeatureData) (math.Point3, bool) {
	if len(d.Solution) != 3 {
		return math.Point3{}, false
	}
	return math.P3(d.Solution[0], d.Solution[1], d.Solution[2]), true
}

// point3From reads a 3-component coordinate slice into a point, naming what for errors.
func point3From(s []float64, what string) (math.Point3, error) {
	if len(s) != 3 {
		return math.Point3{}, fmt.Errorf("%s needs 3 coordinates, got %d", what, len(s))
	}
	return math.P3(s[0], s[1], s[2]), nil
}

// unit3From reads a 3-component slice into a unit vector (erroring on a zero vector).
func unit3From(s []float64, what string) (math.UnitVector3, error) {
	if len(s) != 3 {
		return math.UnitVector3{}, fmt.Errorf("%s needs 3 components, got %d", what, len(s))
	}
	return math.NewUnitVector3(s[0], s[1], s[2])
}
