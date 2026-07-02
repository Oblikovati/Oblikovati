// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/model/sketch"
)

// 3D-sketch constraint tools (issue #142): tangent, smooth, helical, and spline-fit
// reach the user the same tool-first way as the 2D Constrain panel. They reuse the
// generic pick-driven [ConstraintTool] — 3D entities implement sketch.Entity and
// arrive as SketchEntityHandle picks from the viewport's 3D-sketch picking — and
// apply against the active 3D sketch's constraint collection.

// geom3D returns the active 3D sketch's geometric-constraint collection (the same
// collection type 2D sketches use; the entities differ).
func (s *Session) geom3D() *sketch.GeometricConstraints {
	return s.activeSketch3D.GeometricConstraints3D()
}

// afterConstraint3D re-solves the active 3D sketch and clears the selection — the 3D
// counterpart of afterConstraint.
func (s *Session) afterConstraint3D() error {
	if s.activeSketch3D == nil {
		return errors.New("no active 3D sketch")
	}
	s.activeSketch3D.Solve()
	s.Select(nil)
	return nil
}

// sketch3DConstraintToolDefs is the 3D Sketch tab's Constrain panel, in display
// order. Icons reuse the 2D constraint glyphs (tangent/smooth) and the entity glyphs
// (helix/spline) so the panel needs no new assets.
var sketch3DConstraintToolDefs = []struct {
	id, name, icon, tooltip, prompt string
	new                             func() *ConstraintTool
}{
	{
		"Sketch3D.Tangent", "Tangent", "tangent",
		"Tangent — pick two connected curves (line/arc/spline) to join tangentially (G1).",
		"Select two curves to join tangentially",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Tangent (3D)", prompt: "Select two curves to join tangentially",
				accepts: acceptSmoothCurve3D, ready: ready2SmoothCurves3D, apply: entityApply(applyTangent3D)}
		},
	},
	{
		"Sketch3D.Smooth", "Smooth", "smooth",
		"Smooth (G2) — pick a spline and an adjacent curve to join curvature-continuously.",
		"Select a spline and an adjacent curve to join smoothly (G2)",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Smooth (3D)", prompt: "Select a spline and an adjacent curve to join smoothly (G2)",
				accepts: acceptSmoothCurve3D, ready: readySmooth3D, apply: entityApply(applySmooth3D)}
		},
	},
	{
		"Sketch3D.Helical", "Helical", "helix",
		"Helical — pick a helix and the circle it starts on (coincident origin, equal radius).",
		"Select a helix and a circle",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Helical", prompt: "Select a helix and a circle",
				accepts: acceptHelixOrCircle3D, ready: readyHelical3D, apply: entityApply(applyHelical3D)}
		},
	},
	{
		"Sketch3D.SplineFit", "Spline Fit", "spline",
		"Spline Fit — attach a point to the nearest fit point of a fit spline.",
		"Select a fit spline and a point",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Spline Fit", prompt: "Select a fit spline and a point",
				accepts: acceptSplineOrPoint3D, ready: readySplineFit3D, apply: entityApply(applySplineFit3D)}
		},
	},
}

// --- accepts / ready predicates ---------------------------------------------------------

func acceptSmoothCurve3D(e sketch.Entity) bool {
	_, ok := e.(sketch.SmoothCurve3D)
	return ok
}

func ready2SmoothCurves3D(ents []sketch.Entity) bool { return len(smoothCurves3DOf(ents)) >= 2 }

// readySmooth3D needs two curves, at least one being a spline (G2 needs an end the
// solver can bend — mirrors the 2D Smooth tool).
func readySmooth3D(ents []sketch.Entity) bool {
	curves := smoothCurves3DOf(ents)
	return len(curves) >= 2 && hasSpline3D(curves)
}

func acceptHelixOrCircle3D(e sketch.Entity) bool {
	return entityKindIs(e, sketch.HelicalKind, sketch.CircleKind)
}

func readyHelical3D(ents []sketch.Entity) bool {
	h, c := helixAndCircleOf(ents)
	return h != nil && c != nil
}

func acceptSplineOrPoint3D(e sketch.Entity) bool {
	// A fit spline's kind is SplineKind; control-point splines don't qualify.
	return entityKindIs(e, sketch.SplineKind, sketch.PointKind)
}

func readySplineFit3D(ents []sketch.Entity) bool {
	sp, p := splineAndPointOf(ents)
	return sp != nil && p != nil
}

// --- apply functions ----------------------------------------------------------------------

// applyTangent3D joins two curves with a tangent (G1) constraint at their nearest
// endpoints.
func applyTangent3D(s *Session, ents []sketch.Entity) error {
	curves := smoothCurves3DOf(ents)
	if len(curves) >= 2 {
		if p1, p2, ok := sketch.NearestEndpointPair3D(curves[0], curves[1]); ok {
			s.geom3D().Add(sketch.NewTangent3D(curves[0], curves[1], p1, p2))
			return s.afterConstraint3D()
		}
	}
	return errNeed("tangent", "two curves with free endpoints (line/arc/spline)")
}

// applySmooth3D joins two curves with a smooth (G2) constraint at their nearest
// endpoints — at least one side must be a spline.
func applySmooth3D(s *Session, ents []sketch.Entity) error {
	curves := smoothCurves3DOf(ents)
	if len(curves) >= 2 && hasSpline3D(curves) {
		if p1, p2, ok := sketch.NearestEndpointPair3D(curves[0], curves[1]); ok {
			s.geom3D().Add(sketch.NewSmooth3D(curves[0], curves[1], p1, p2))
			return s.afterConstraint3D()
		}
	}
	return errNeed("smooth", "a spline and an adjacent curve with free endpoints")
}

// applyHelical3D ties the picked helix to the picked circle.
func applyHelical3D(s *Session, ents []sketch.Entity) error {
	h, circle := helixAndCircleOf(ents)
	if h == nil || circle == nil {
		return errNeed("helical", "a helix and a circle")
	}
	c, err := sketch.NewHelical3D(h, circle)
	if err != nil {
		return fmt.Errorf("helical: %w", err)
	}
	s.geom3D().Add(c)
	return s.afterConstraint3D()
}

// applySplineFit3D attaches the picked point to the picked fit spline.
func applySplineFit3D(s *Session, ents []sketch.Entity) error {
	sp, p := splineAndPointOf(ents)
	if sp == nil || p == nil {
		return errNeed("spline fit", "a fit spline and a point")
	}
	c, err := sketch.NewSplineFitPoints3D(sp, p)
	if err != nil {
		return fmt.Errorf("spline fit: %w", err)
	}
	s.geom3D().Add(c)
	return s.afterConstraint3D()
}

// --- pick filtering helpers -----------------------------------------------------------------

// smoothCurves3DOf extracts the tangent/smooth-capable curves from the picks.
func smoothCurves3DOf(ents []sketch.Entity) []sketch.SmoothCurve3D {
	var out []sketch.SmoothCurve3D
	for _, e := range ents {
		if c, ok := e.(sketch.SmoothCurve3D); ok {
			out = append(out, c)
		}
	}
	return out
}

func hasSpline3D(curves []sketch.SmoothCurve3D) bool {
	for _, c := range curves {
		if _, ok := c.(*sketch.Spline3D); ok {
			return true
		}
	}
	return false
}

// helixAndCircleOf extracts the first helix and first circle from the picks.
func helixAndCircleOf(ents []sketch.Entity) (*sketch.HelicalCurve3D, *sketch.Circle3D) {
	var h *sketch.HelicalCurve3D
	var c *sketch.Circle3D
	for _, e := range ents {
		if v, ok := e.(*sketch.HelicalCurve3D); ok && h == nil {
			h = v
		}
		if v, ok := e.(*sketch.Circle3D); ok && c == nil {
			c = v
		}
	}
	return h, c
}

// splineAndPointOf extracts the first fit spline and first point from the picks.
func splineAndPointOf(ents []sketch.Entity) (*sketch.Spline3D, *sketch.Point3D) {
	var sp *sketch.Spline3D
	var p *sketch.Point3D
	for _, e := range ents {
		if v, ok := e.(*sketch.Spline3D); ok && sp == nil && v.IsFitType() {
			sp = v
		}
		if v, ok := e.(*sketch.Point3D); ok && p == nil {
			p = v
		}
	}
	return sp, p
}

// entityKindIs reports whether the entity names one of the given kinds — the
// capability-based accept check the pick predicates share (#1624).
func entityKindIs(e sketch.Entity, kinds ...sketch.EntityKind) bool {
	ke, ok := e.(interface{ Kind() sketch.EntityKind })
	if !ok {
		return false
	}
	for _, k := range kinds {
		if ke.Kind() == k {
			return true
		}
	}
	return false
}
