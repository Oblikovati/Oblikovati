// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"strings"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// createSketch adds a sketch on an origin plane of the active part and returns its
// index (for sketch.rectangle / features.add).
func createSketch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.CreateSketchArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	plane, name, host, err := sketchCreatePlane(part, in)
	if err != nil {
		return nil, err
	}
	sk := part.Sketches().Add(plane)
	// Share the part's parameter DAG so dimension expressions can reference user
	// parameters (e.g. "od/2") and the dimension's own d0,d1… parameters live in
	// the part's table — the way Inventor sketch dimensions work. Without this the
	// sketch keeps an isolated param store and "od/2" resolves to 0, collapsing
	// the geometry.
	sk.SetParameters(part.Parameters())
	// On a work plane, track it so the sketch follows when the plane moves.
	if host != nil {
		sk.SetPlaneHost(host)
	}
	return json.Marshal(wire.CreateSketchResult{SketchIndex: part.Sketches().Count() - 1, Plane: name})
}

// sketchCreatePlane resolves the plane a new sketch starts on: a user work plane (when
// WorkPlaneIndex is set — the way to sketch on a plane built on a feature-created face) or an
// origin plane otherwise.
func sketchCreatePlane(part *compdef.PartComponentDefinition, in wire.CreateSketchArgs) (sketch.Plane, string, func() sketch.Plane, error) {
	if in.WorkPlaneIndex == nil {
		plane, name, err := parsePlane(in.Plane)
		return plane, name, nil, err // origin planes are fixed, no host to track
	}
	planes := part.WorkPlanes()
	i := *in.WorkPlaneIndex
	if i < 0 || i >= planes.Count() {
		return sketch.Plane{}, "", nil, fmt.Errorf("sketch.create: work plane %d out of range (part has %d)", i, planes.Count())
	}
	wp := planes.Item(i)
	// The host re-reads the work plane (recomputed in place) so the sketch tracks it.
	return wp.Plane(), wp.Name(), func() sketch.Plane { return wp.Plane() }, nil
}

// sketchRectangle adds a closed rectangle (one profile) to a sketch, ready to extrude.
func sketchRectangle(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.SketchRectangleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	w, err := part.Units().Parse(in.Width, param.Length)
	if err != nil {
		return nil, fmt.Errorf("sketch.rectangle: width %q: %w", in.Width, err)
	}
	h, err := part.Units().Parse(in.Height, param.Length)
	if err != nil {
		return nil, fmt.Errorf("sketch.rectangle: height %q: %w", in.Height, err)
	}
	addRectangle(sk, w.Value, h.Value)
	return json.Marshal(wire.SketchRectangleResult{SketchIndex: in.SketchIndex, Profiles: sk.Profiles().Count()})
}

// addRectangle draws a closed w×h rectangle at the sketch origin (one profile).
func addRectangle(sk *sketch.Sketch, w, h float64) {
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(w, 0))
	c2 := sk.Points().Add(math.P2(w, h))
	c3 := sk.Points().Add(math.P2(0, h))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

// parsePlane maps a plane name to its sketch plane and a normalized label.
func parsePlane(name string) (sketch.Plane, string, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "", "XY":
		return sketch.XYPlane(), "XY", nil
	case "XZ":
		return sketch.XZPlane(), "XZ", nil
	case "YZ":
		return sketch.YZPlane(), "YZ", nil
	default:
		return sketch.Plane{}, "", fmt.Errorf("sketch.create: unknown plane %q (want XY|XZ|YZ)", name)
	}
}

// sketchAtIndex returns the active part's sketch at i, bounds-checked.
func sketchAtIndex(part *compdef.PartComponentDefinition, i int) (*sketch.Sketch, error) {
	if i < 0 || i >= part.Sketches().Count() {
		return nil, fmt.Errorf("sketch index %d out of range (part has %d sketches)", i, part.Sketches().Count())
	}
	return part.Sketches().Item(i), nil
}
