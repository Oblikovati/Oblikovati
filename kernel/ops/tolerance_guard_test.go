// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestNoUnjustifiedAbsoluteEpsilons is the regression guard for ADR-0042 / #1399: the kernel's
// intersection, weld and degeneracy HOT PATHS must not gate a decision on a cm-anchored absolute
// length epsilon. A model-relative tolerance carries no literal — it reads res.Weld()/res.Plane()
// from a geom.Resolution — so any bare `1e-N` literal that survives in these files is either a
// genuine length tolerance that still needs relativising (a regression) OR a deliberately
// dimensionless / calibrated one that must SAY SO with a `// tol:<kind>` annotation on its line.
//
// Allowed annotations (the literal stays absolute, justified):
//
//	tol:angular     — a cosine/sine/dot threshold on unit vectors (dimensionless)
//	tol:parametric  — a normalised curve/surface parameter u/v/t in [0,1] or knot space
//	tol:numeric     — a convergence / determinant / rank / slope guard, not a model length
//	tol:area        — scales with size² (a near-zero-area test)
//	tol:volume      — scales with size³
//	tol:calibrated  — a length tolerance deliberately kept absolute because it is validated
//	                  against an external oracle in a normalised frame (e.g. the ruled (u,v)
//	                  arrangement split); converting it shifts borderline classifications
//
// A length tolerance with none of these and no res.-derived value is a FAILURE: relativise it
// (geom.ResolutionForBox/ForSize(...).Weld()/.Plane()) or justify it with the right annotation.
func TestNoUnjustifiedAbsoluteEpsilons(t *testing.T) {
	// In-scope hot-path files (relative to this package dir, kernel/ops). Kept explicit rather than
	// globbed so widening the guard's reach is a deliberate, reviewable act.
	scope := []string{
		// geom: the analytic + numeric surface-intersection classifiers and the SSI tracer.
		"../geom/intersect_analytic.go",
		"../geom/intersect2d.go",
		"../geom/intersect_surface_trace.go",
		"../geom/trace_surface_zero.go",
		"../geom/network_surface.go",
		// brep: the curved-boolean weld / imprint / half-space machinery.
		"../brep/curved_halfspace_general.go",
		"../brep/curved_halfspace_cylinder.go",
		"../brep/curved_halfspace_cone.go",
		"../brep/curved_halfspace_torus.go",
		"../brep/curved_halfspace_looped.go",
		"../brep/curved_halfspace_cone_side.go",
		"../brep/curved_crossing_imprint.go",
		"../brep/curved_crossing_cut.go",
		"../brep/curved_crossing_intersect.go",
		"../brep/curved_cone_cylinder_intersect.go",
		"../brep/curved_partial_penetration.go",
		"../brep/curved_subtract_prism.go",
		"../brep/curved_coaxial_cylinder.go",
		"../brep/curved_cylinder_boss.go",
		"../brep/curved_steinmetz.go",
		// brep: the planar-arrangement boolean (weld grid, T-junctions, imprint, stitch).
		"../brep/arrange2d.go",
		"../brep/arrange2d_trace.go",
		"../brep/boolean.go",
		"../brep/boolean_classify.go",
		"../brep/boolean_coplanar.go",
		"../brep/boolean_geom.go",
		"../brep/boolean_split.go",
		"../brep/boolean_stitch.go",
		"../brep/boolean_provenance.go",
		"../brep/definition.go",
		"../brep/drill_blind.go",
		// ops: the mesher vertex-weld path.
		"../ops/edge_discretize.go",
		"../ops/closed_surface_mesh.go",
		"../ops/closed_band_loft.go",
		"../ops/saddle_band_loft.go",
		"../ops/nurbs_pcurve_mesh.go",
	}
	epsilon := regexp.MustCompile(`[^0-9a-zA-Z_.]1(?:\.0)?e-[0-9]+`)
	var offenders []string
	for _, rel := range scope {
		src, err := os.ReadFile(rel)
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			code, comment, _ := strings.Cut(line, "//")
			if !epsilon.MatchString(" " + code) {
				continue
			}
			if strings.Contains(comment, "tol:") {
				continue // justified on its own line
			}
			offenders = append(offenders, filepath.Clean(rel)+":"+itoa(i+1)+":  "+strings.TrimSpace(line))
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("unjustified absolute length epsilon(s) in #1399 hot paths — relativise (res.Weld()/"+
			"res.Plane()) or annotate `// tol:<kind>`:\n%s", strings.Join(offenders, "\n"))
	}
}

// itoa is a tiny dependency-free int→string (avoids importing strconv just for the failure message).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
