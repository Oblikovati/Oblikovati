// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"strings"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
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
	WorkGeometry() *feature.WorkGeometry // resolve orientation-axis refs (#1920)
}

// createSketch adds a sketch on an origin or work plane of the active sketch host (part
// or assembly) and returns its index (for sketch.rectangle / features.add).
func createSketch(s *app.Session, host sketchHost, in wire.CreateSketchArgs) (wire.CreateSketchResult, error) {
	base, name, wp, err := sketchCreatePlane(host, in)
	if err != nil {
		return wire.CreateSketchResult{}, err
	}
	// Validate the orientation up-front so a bad axis fails the create call, not a later
	// recompute; then track the (re-)oriented plane so the frame survives plane motion.
	if in.Orientation != nil {
		if _, err := orientSketchPlane(host, base, in.Orientation); err != nil {
			return wire.CreateSketchResult{}, err
		}
	}
	planeOf := sketchPlaneFn(host, base, wp, in.Orientation)
	sk := host.Sketches().Add(planeOf())
	// Share the host's parameter DAG so dimension expressions can reference user
	// parameters (e.g. "od/2") and the dimension's own d0,d1… parameters live in
	// the host's table — the way Inventor sketch dimensions work. Without this the
	// sketch keeps an isolated param store and "od/2" resolves to 0, collapsing
	// the geometry.
	sk.SetParameters(host.Parameters())
	// Give it the same projected origin the interactive Create 2D Sketch gives, so a sketch made
	// over the wire opens with the same (0,0) anchor as one drawn by hand (#2016).
	if part, ok := host.(*compdef.PartComponentDefinition); ok {
		s.AutoProjectOriginInto(part, sk)
	}
	// On a work plane, track it so the sketch follows when the plane moves, and fold the
	// plane's offset parameter into the sketch's footprint so an offset edit targets this
	// sketch's features instead of forcing a wholesale rebuild (ADR-0044).
	if wp != nil {
		sk.SetPlaneHost(planeOf)
		sk.SetHostFootprint(func() []depend.Key { return wp.ParameterFootprint() })
		sk.SetHostWorkRef(string(wp.Key())) // consumer link: this sketch consumes the work plane (#1849)
	}
	return wire.CreateSketchResult{SketchIndex: host.Sketches().Count() - 1, Plane: name}, nil
}

// sketchPlaneFn returns the source of a sketch's host plane on each recompute: the base
// plane (or the live work plane when wp is set), reframed by an optional orientation. The
// orientation is re-applied every recompute so the pinned frame follows the plane if it
// moves; if the reframe later degenerates it falls back to the base frame (it was validated
// at create time). See [orientSketchPlane] and #1920.
func sketchPlaneFn(host sketchHost, base sketch.Plane, wp *feature.WorkPlane, o *wire.SketchOrientation) func() sketch.Plane {
	basePlane := func() sketch.Plane {
		if wp != nil {
			return wp.Plane()
		}
		return base
	}
	if o == nil {
		return basePlane
	}
	return func() sketch.Plane {
		p, err := orientSketchPlane(host, basePlane(), o)
		if err != nil {
			return basePlane()
		}
		return p
	}
}

// orientSketchPlane reframes a sketch plane so a reference axis (projected into the plane)
// becomes its X or Y axis — Inventor's PlanarSketches.AddWithOrientation. The plane's origin
// and normal are preserved (so the sketch faces the same way and normal = X×Y still holds);
// only the in-plane rotation and, optionally, the origin change. Errors if the axis is
// perpendicular to the plane (its in-plane projection is degenerate). See #1920.
func orientSketchPlane(host sketchHost, base sketch.Plane, o *wire.SketchOrientation) (sketch.Plane, error) {
	dir, err := orientationAxisDir(host, o.Axis)
	if err != nil {
		return sketch.Plane{}, err
	}
	normal := base.Normal().AsVector()
	chosen, err := math.UnitVector3FromVector(dir.Sub(normal.Scale(dir.Dot(normal))))
	if err != nil {
		return sketch.Plane{}, fmt.Errorf("sketch.create: orientation axis %q is perpendicular to the plane "+
			"(normal %v); its in-plane projection is degenerate", o.Axis, normal)
	}
	if o.Reverse {
		chosen, _ = math.UnitVector3FromVector(chosen.AsVector().Scale(-1))
	}
	origin := base.Origin()
	if len(o.Origin) == 3 {
		origin = math.P3(o.Origin[0], o.Origin[1], o.Origin[2])
	}
	x, y, err := orientedAxes(chosen, base.Normal(), o.AxisIsX)
	if err != nil {
		return sketch.Plane{}, err
	}
	return sketch.NewPlane(origin, x, y)
}

// orientedAxes completes an orthonormal in-plane frame from the chosen axis and the plane
// normal: the chosen axis is X (then Y = normal×X) or Y (then X = Y×normal), each choice
// giving normal = X×Y so the plane's facing is preserved.
func orientedAxes(chosen, normal math.UnitVector3, axisIsX bool) (x, y math.UnitVector3, err error) {
	if axisIsX {
		x = chosen
		y, err = math.UnitVector3FromVector(normal.Cross(x))
		return x, y, err
	}
	y = chosen
	x, err = math.UnitVector3FromVector(y.Cross(normal).Scale(-1)) // X = Y × normal
	return x, y, err
}

// orientationAxisDir resolves an orientation-axis reference (an origin-axis constant, a work
// axis, or a linear edge) to its unit direction in model space.
func orientationAxisDir(host sketchHost, ref string) (math.Vector3, error) {
	wa, ok := host.WorkGeometry().AxisByRef(feature.ParseWorkRef(ref))
	if !ok {
		return math.Vector3{}, fmt.Errorf("sketch.create: orientation axis %q not found", ref)
	}
	return wa.Direction().AsVector(), nil
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
