// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

// Audit I6: the STEP analytic export is table-driven (stepSurfaceWriters / stepCurveWriters
// keyed on geom kind). These coverage tests enumerate the geom kind enum and assert every
// kind is EITHER handled by a writer OR explicitly declared unsupported with a reason — so a
// new geom kind that no one wired fails here, at CI, instead of silently hitting the
// table-miss default at runtime (the #1403 class this refactor closes for STEP export).

// stepSurfaceUnsupported are the surface kinds with no STEP analytic entity; export falls
// back to a tessellated face (deferred to PBI-E). Each names why — an undeclared gap fails.
var stepSurfaceUnsupported = map[geom.SurfaceKind]string{
	geom.SurfaceBSpline:            "B_SPLINE_SURFACE export deferred to PBI-E (tessellated-face fallback for now)",
	geom.SurfaceEllipticalCylinder: "no STEP analytic entity for an elliptical cylinder; exported via tessellation",
	geom.SurfaceEllipticalCone:     "no STEP analytic entity for an elliptical cone; exported via tessellation",
	geom.SurfaceOffset:             "offset surface has no direct STEP entity; exported via tessellation",
	geom.SurfaceThreadedCylinder:   "thread is cosmetic; export the underlying cylinder, not a distinct entity",
}

func TestStepSurfaceWriterCoverage(t *testing.T) {
	for _, k := range geom.SurfaceKinds() {
		_, hasWriter := stepSurfaceWriters[k]
		reason, declared := stepSurfaceUnsupported[k]
		if hasWriter && declared {
			t.Errorf("surface kind %v is both a STEP writer and declared unsupported — remove one", k)
		}
		if !hasWriter && !declared {
			t.Errorf("surface kind %v has no STEP writer and is not in stepSurfaceUnsupported — add a writer "+
				"or declare it unsupported with a reason (audit I6)", k)
		}
		if declared && reason == "" {
			t.Errorf("surface kind %v is declared unsupported with an empty reason", k)
		}
	}
	assertNoStaleSurfaceUnsupported(t)
}

func assertNoStaleSurfaceUnsupported(t *testing.T) {
	t.Helper()
	for k := range stepSurfaceUnsupported {
		if _, hasWriter := stepSurfaceWriters[k]; hasWriter {
			t.Errorf("stepSurfaceUnsupported entry %v now has a writer — delete the stale entry", k)
		}
	}
}

// stepCurveUnsupported are the curve kinds with no STEP analytic writer yet; export falls
// back to a polyline/tessellation.
var stepCurveUnsupported = map[geom.CurveKind]string{
	geom.CurvePolyline:      "polyline is emitted as its constituent segments, not a single STEP entity",
	geom.CurveEllipse:       "ELLIPSE export not yet implemented; falls back to a polyline",
	geom.CurveEllipticalArc: "elliptical-arc export not yet implemented; falls back to a polyline",
	geom.CurveHyperbolicArc: "HYPERBOLA export not yet implemented; falls back to a polyline",
	geom.CurveParabola:      "PARABOLA export not yet implemented; falls back to a polyline",
	geom.CurveBSpline:       "B_SPLINE_CURVE export handled by bspline_from_step's inverse path, not this table",
	geom.CurveHelix:         "helix has no STEP analytic entity; exported via a polyline",
	geom.CurveVariableHelix: "variable-pitch helix has no STEP analytic entity; exported via a polyline",
	geom.CurveSpiric:        "spiric arc (torus-section) has no STEP analytic entity; exported via a polyline",
	geom.CurveTorusCyl:      "torus∩cylinder section arc has no STEP analytic entity; exported via a polyline",
	geom.CurveRuledQuadric:  "ruled∩quadric section arc (a quartic space curve) has no STEP analytic entity; exported via a polyline",
}

func TestStepCurveWriterCoverage(t *testing.T) {
	for _, k := range geom.CurveKinds() {
		_, hasWriter := stepCurveWriters[k]
		reason, declared := stepCurveUnsupported[k]
		if hasWriter && declared {
			t.Errorf("curve kind %v is both a STEP writer and declared unsupported — remove one", k)
		}
		if !hasWriter && !declared {
			t.Errorf("curve kind %v has no STEP writer and is not in stepCurveUnsupported — add a writer "+
				"or declare it unsupported with a reason (audit I6)", k)
		}
		if declared && reason == "" {
			t.Errorf("curve kind %v is declared unsupported with an empty reason", k)
		}
	}
	for k := range stepCurveUnsupported {
		if _, hasWriter := stepCurveWriters[k]; hasWriter {
			t.Errorf("stepCurveUnsupported entry %v now has a writer — delete the stale entry", k)
		}
	}
}
