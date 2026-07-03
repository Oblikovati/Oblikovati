// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"strings"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/depend"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// activeSketchHost resolves the active document's content as a sketch host (a part or
// an assembly), erroring if there is no active document or its content hosts no sketches.
func activeSketchHost(s *app.Session) (sketchHost, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, modelaccess.ErrNoActiveDocument
	}
	host, ok := d.Content().(sketchHost)
	if !ok {
		return nil, fmt.Errorf("router: active document %q does not host sketches", d.DisplayName())
	}
	return host, nil
}

// sketchHost is the active-document content a sketch is authored on — a part or an
// assembly. Both own a sketch collection, the parameter DAG dimensions share, display
// units, and datum planes, so the sketch-authoring methods (create/rectangle) work
// against either without knowing which (#739).
type sketchHost interface {
	Sketches() *sketch.Sketches
	Parameters() *param.Parameters
	Units() param.UnitsOfMeasure
	WorkPlanes() *feature.WorkPlanes
}

// createSketch adds a sketch on an origin or work plane of the active sketch host (part
// or assembly) and returns its index (for sketch.rectangle / features.add).
func createSketch(_ *app.Session, host sketchHost, in wire.CreateSketchArgs) (wire.CreateSketchResult, error) {
	plane, name, wp, err := sketchCreatePlane(host, in)
	if err != nil {
		return wire.CreateSketchResult{}, err
	}
	sk := host.Sketches().Add(plane)
	// Share the host's parameter DAG so dimension expressions can reference user
	// parameters (e.g. "od/2") and the dimension's own d0,d1… parameters live in
	// the host's table — the way Inventor sketch dimensions work. Without this the
	// sketch keeps an isolated param store and "od/2" resolves to 0, collapsing
	// the geometry.
	sk.SetParameters(host.Parameters())
	// On a work plane, track it so the sketch follows when the plane moves, and fold the
	// plane's offset parameter into the sketch's footprint so an offset edit targets this
	// sketch's features instead of forcing a wholesale rebuild (ADR-0044).
	if wp != nil {
		sk.SetPlaneHost(func() sketch.Plane { return wp.Plane() })
		sk.SetHostFootprint(func() []depend.Key { return wp.ParameterFootprint() })
	}
	return wire.CreateSketchResult{SketchIndex: host.Sketches().Count() - 1, Plane: name}, nil
}

// sketchCreatePlane resolves the plane a new sketch starts on: a user work plane (when
// WorkPlaneIndex is set — the way to sketch on a plane built on a feature-created face) or an
// origin plane otherwise.
func sketchCreatePlane(host sketchHost, in wire.CreateSketchArgs) (sketch.Plane, string, *feature.WorkPlane, error) {
	if in.WorkPlaneIndex == nil {
		plane, name, err := parsePlane(in.Plane)
		return plane, name, nil, err // origin planes are fixed, no host to track
	}
	planes := host.WorkPlanes()
	i := *in.WorkPlaneIndex
	if i < 0 || i >= planes.Count() {
		return sketch.Plane{}, "", nil, fmt.Errorf("sketch.create: work plane %d out of range (have %d)", i, planes.Count())
	}
	wp := planes.Item(i)
	// The caller re-reads the work plane (recomputed in place) so the sketch tracks it, and
	// reads its footprint so the offset parameter is attributed to the sketch.
	return wp.Plane(), wp.Name(), wp, nil
}

// sketchRectangle adds a closed rectangle (one profile) to a sketch, ready to extrude.
func sketchRectangle(_ *app.Session, host sketchHost, in wire.SketchRectangleArgs) (wire.SketchRectangleResult, error) {
	sk, err := sketchAtIndex(host, in.SketchIndex)
	if err != nil {
		return wire.SketchRectangleResult{}, err
	}
	w, err := resolveQuantity(host, in.Width, param.Length)
	if err != nil {
		return wire.SketchRectangleResult{}, fmt.Errorf("sketch.rectangle: width %q: %w", in.Width, err)
	}
	h, err := resolveQuantity(host, in.Height, param.Length)
	if err != nil {
		return wire.SketchRectangleResult{}, fmt.Errorf("sketch.rectangle: height %q: %w", in.Height, err)
	}
	addRectangle(sk, w.Value, h.Value)
	return wire.SketchRectangleResult{SketchIndex: in.SketchIndex, Profiles: sk.Profiles().Count()}, nil
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

// sketchAtIndex returns the host's sketch at i, bounds-checked.
func sketchAtIndex(host sketchHost, i int) (*sketch.Sketch, error) {
	if i < 0 || i >= host.Sketches().Count() {
		return nil, fmt.Errorf("sketch index %d out of range (have %d sketches)", i, host.Sketches().Count())
	}
	return host.Sketches().Item(i), nil
}
