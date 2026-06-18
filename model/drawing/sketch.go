// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Drawing sketches (M14-F08, #638): 2D geometry drawn directly in sheet space (millimetres) on a
// sheet — linework and detail enrichment, and the closed boundaries that hatch regions fill. A
// sketch holds entities (lines, circles, rectangles) that render as drawing curves the head draws
// alongside the views and annotations.

// DrawingSketchEntity is one piece of sketch geometry: its kind, sheet-millimetre points, and (for a
// circle) radius.
type DrawingSketchEntity struct {
	kind   types.DrawingSketchEntityKind
	points [][2]float64
	radius float64
}

// DrawingSketch is a named collection of 2D entities in sheet space, rendered as drawing curves.
type DrawingSketch struct {
	name     string
	entities []DrawingSketchEntity
	hatches  []hatchRegion
	curves   []DrawingCurve
}

// Name, Curves, EntityCount and CurveCount expose the sketch.
func (s *DrawingSketch) Name() string           { return s.name }
func (s *DrawingSketch) Curves() []DrawingCurve { return s.curves }
func (s *DrawingSketch) EntityCount() int       { return len(s.entities) }
func (s *DrawingSketch) CurveCount() int        { return len(s.curves) }

// DrawingSketches is a sheet's collection of drawing sketches.
type DrawingSketches struct {
	items []*DrawingSketch
}

// Count and Item expose the collection.
func (ss *DrawingSketches) Count() int                { return len(ss.items) }
func (ss *DrawingSketches) Item(i int) *DrawingSketch { return ss.items[i] }

// ByName returns the sketch with the given name.
func (ss *DrawingSketches) ByName(name string) (*DrawingSketch, bool) {
	for _, s := range ss.items {
		if s.name == name {
			return s, true
		}
	}
	return nil, false
}

// uniqueName returns the requested name if free, else a generated SKETCH:n name.
func (ss *DrawingSketches) uniqueName(requested string) string {
	if requested != "" {
		if _, exists := ss.ByName(requested); !exists {
			return requested
		}
	}
	for n := len(ss.items) + 1; ; n++ {
		name := fmt.Sprintf("SKETCH:%d", n)
		if _, exists := ss.ByName(name); !exists {
			return name
		}
	}
}

// Add creates a new empty sketch.
func (ss *DrawingSketches) Add(name string) *DrawingSketch {
	s := &DrawingSketch{name: ss.uniqueName(name)}
	ss.items = append(ss.items, s)
	return s
}

// AddEntity adds one entity to the named sketch and rebuilds its curves. It errors on an unknown
// sketch or malformed entity (wrong point count / non-positive radius).
func (ss *DrawingSketches) AddEntity(sketchName string, kind types.DrawingSketchEntityKind, points [][2]float64, radius float64) (*DrawingSketch, error) {
	s, ok := ss.ByName(sketchName)
	if !ok {
		return nil, fmt.Errorf("drawing: no sketch %q to add an entity to", sketchName)
	}
	if err := validateSketchEntity(kind, points, radius); err != nil {
		return nil, err
	}
	s.entities = append(s.entities, DrawingSketchEntity{kind: kind, points: points, radius: radius})
	s.rebuild()
	return s, nil
}

// validateSketchEntity checks an entity has the geometry its kind needs.
func validateSketchEntity(kind types.DrawingSketchEntityKind, points [][2]float64, radius float64) error {
	switch kind {
	case types.SketchLineEntity, types.SketchRectangleEntity:
		if len(points) != 2 {
			return fmt.Errorf("drawing: a %s sketch entity needs 2 points, got %d", kind, len(points))
		}
	case types.SketchCircleEntity:
		if len(points) != 1 {
			return fmt.Errorf("drawing: a circle sketch entity needs 1 centre point, got %d", len(points))
		}
		if radius <= 0 {
			return fmt.Errorf("drawing: a circle sketch entity needs a positive radius, got %g", radius)
		}
	default:
		return fmt.Errorf("drawing: unknown sketch entity kind %v", kind)
	}
	return nil
}

// rebuild regenerates the sketch's drawing curves from its entities and hatch regions.
func (s *DrawingSketch) rebuild() {
	s.curves = nil
	for _, e := range s.entities {
		s.curves = append(s.curves, sketchEntityCurves(e)...)
	}
	for _, h := range s.hatches {
		s.curves = append(s.curves, hatchRegionCurves(h)...)
	}
}

// sketchEntityCurves renders one entity as drawing curves (sheet millimetres).
func sketchEntityCurves(e DrawingSketchEntity) []DrawingCurve {
	switch e.kind {
	case types.SketchLineEntity:
		return []DrawingCurve{dimSegment(e.points[0][0], e.points[0][1], e.points[1][0], e.points[1][1])}
	case types.SketchCircleEntity:
		return circlePolyline(e.points[0][0], e.points[0][1], e.radius)
	case types.SketchRectangleEntity:
		x0, y0, x1, y1 := e.points[0][0], e.points[0][1], e.points[1][0], e.points[1][1]
		return []DrawingCurve{
			dimSegment(x0, y0, x1, y0), dimSegment(x1, y0, x1, y1),
			dimSegment(x1, y1, x0, y1), dimSegment(x0, y1, x0, y0),
		}
	}
	return nil
}
