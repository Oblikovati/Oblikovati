// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"io/fs"
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
	t.Parallel()
	// In-scope hot-path files, named by their path under kernel/. Kept explicit rather than
	// globbed so widening the guard's reach is a deliberate, reviewable act. The paths are
	// RESOLVED BY BASENAME under kernel/ (resolveScope), because splitting a package moves a
	// file without changing what it does: a stale path here must report as a moved file, not
	// as a tolerance regression.
	scope := []string{
		// geom: the analytic + numeric surface-intersection classifiers and the SSI tracer.
		"../geom/intersect_analytic.go",
		"../geom/intersect2d.go",
		"../geom/intersect2d_circle.go",
		"../geom/intersect_surface_trace.go",
		"../geom/intersect_surface_sweep.go",     // split out of intersect_surface_trace.go (#2214)
		"../geom/intersect_surface_corrector.go", // ditto — the guard follows the code, not the filename
		"../geom/trace_surface_zero.go",
		"../geom/network_surface.go",
		// geom: line/plane/line queries, 2D arc containment, and the curve evaluators
		// (stroking deflection, arc-length integration, point-inversion clustering) + the
		// B-spline / NURBS knot-removal & degree-reduction fitting tolerances (#1504).
		"../geom/query.go",
		"../geom/arc2d.go",
		"../geom/evaluator_common.go",
		"../geom/evaluator_curve2.go",
		"../geom/evaluator_curve3.go",
		"../geom/evaluator_point2.go",
		"../geom/evaluator_point3.go",
		"../geom/bspline_remove.go",
		"../geom/bspline_reduce.go",
		"../geom/nurbs_remove.go",
		"../geom/nurbs_reduce.go",
		// brep: the curved-boolean weld / imprint / half-space machinery.
		"../brep/curved_halfspace_general.go",
		"../brep/curved_halfspace_cylinder.go",
		"../brep/curved_halfspace_cone.go",
		"../brep/curved_halfspace_torus.go",
		"../brep/curved_halfspace_looped.go",
		"../brep/curved_halfspace_cone_side.go",
		"../brep/curved_crossing_imprint.go",
		"../brep/curved_crossing_imprint_general.go", // the merged imprint (ADR-0058 phase 3); cone_cone_imprint.go was folded in
		"../brep/curved_cone_cylinder_imprint.go",
		"../brep/curved_cylinder_membership.go",
		"../brep/curved_general_boolean.go",
		"../brep/curved_general_ruled_cutjoin.go",
		"../brep/curved_general_wrapping_band.go",
		// brep: the mixed per-face boolean's uv×wall conic pairing and its island sampling (#3460).
		"../brep/boolean_mixed_uvwall.go",
		"../brep/curved_plane_face_uv_island.go",
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
		// brep: the analytic minimum-distance recursion and its numeric minimisers (#3458).
		"../brep/distance.go",
		"../brep/distance_min.go",
		// ops: the mesher vertex-weld path.
		"../ops/tessellate/edge_discretize.go",
		"../ops/tessellate/closed_surface_mesh.go",
		"../ops/tessellate/closed_band_loft.go",
		"../ops/tessellate/saddle_band_loft.go",
		"../ops/tessellate/nurbs_pcurve_mesh.go",
		// ops: the mid-span obstacle fillet-blend slice — rail-corner weld, obstacle feature,
		// and the rim/boundary crossing detector (ADR-0042; the rail-corner weld must stay
		// model-relative, not a bare cm-anchored epsilon).
		"../ops/blend/corner_blend_obstacle_rails.go",
		"../ops/blend/corner_blend_obstacle.go",
		"../ops/blend/corner_blend_obstacle_certify.go",
		"../ops/blend/fillet_obstacle_detect.go",
		"../ops/blend/fillet_obstacle_merge.go",
		// ops: self-intersection, section, stitch, and the degeneracy/weld helpers.
		"../ops/validate/self_intersect.go",
		"../ops/section_plane.go",
		"../ops/heal/stitch.go",
		"../ops/query/oriented_box.go",
		"../ops/ruled_surface.go",
		"../ops/transform/deform.go",
		"../ops/tessellate/smooth_normals.go",
		"../ops/boolean/csg_body.go",
		"../ops/blend/assemble_curved.go",
		// solve + sketch: the constraint solver and 2D arrangement.
		"../../solve/solve.go",
		"../../model/sketch/arrangement.go",
		"../../model/sketch/profile.go",
		"../../model/sketch/path_3d.go",
		// ops: the CDT / conformance / orientation weld grids and trim-grid gates (#1610).
		"../ops/tessellate/conformance_repair.go",
		"../ops/tessellate/tessellate_trim.go",
		"../ops/tessellate/patch_acceptance.go",
		"../ops/tessellate/planar_faithful.go",
		"../ops/tessellate/refined_patch.go",
		"../ops/tessellate/mesh_orient.go",
		"../ops/tessellate/orient_heal.go",
		"../ops/tessellate/holed_cylinder_wall.go",
		"../ops/wire_offset.go",
		"../ops/wire_offset_corners.go",
		"../ops/heal/fill_nsided.go",
		"../ops/heal/fill_opening.go",
		// topo: evaluator / wire / provenance coincidence gates (#1610).
		"../topo/evaluators.go",
		"../topo/face_evaluator.go",
		"../topo/provenance.go",
		"../topo/wire.go",
		// brep: revolution surface classification (#1603: dimensionless slope ratio + meridian-
		// relative axis weld; the only literal left is the annotated angular revSlopeTol).
		"../brep/revolution.go",
	}
	var offenders []string
	for _, rel := range resolveScope(t, scope) {
		src, err := os.ReadFile(rel)
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			code, comment, _ := strings.Cut(line, "//")
			if !toleranceLiteral(code) {
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

// epsilonLiteral matches a bare small scientific literal (1e-N).
var epsilonLiteral = regexp.MustCompile(`[^0-9a-zA-Z_.]1(?:\.0)?e-[0-9]+`)

// reciprocalGrid matches the evasion forms of an absolute tolerance written as its
// reciprocal (#1610): a quantization multiplier (`* 1e5`, `* 100000.0`) or a division of
// one by a named tolerance/grid (`1.0/tol`). These carry the same cm-anchored scale
// assumption as a bare 1e-N and must be relativised or annotated identically.
var reciprocalGrid = regexp.MustCompile(
	`\*\s*1(?:\.0)?e\+?[0-9]+|\*\s*10{4,}(?:\.0)?\b|[^0-9a-zA-Z_.]1(?:\.0)?\s*/\s*[A-Za-z_]*(?:eps|tol|grid|Eps|Tol|Grid)`)

// toleranceLiteral reports whether a code line (comment already stripped) carries an
// absolute-tolerance literal in either direct or reciprocal form.
func toleranceLiteral(code string) bool {
	return epsilonLiteral.MatchString(" "+code) || reciprocalGrid.MatchString(" "+code)
}

// TestToleranceGuardCatchesEvasionForms is the guard-the-guard self-test (#1610): the
// matcher must catch each known evasion spelling, and must not fire on ordinary
// arithmetic — otherwise the whitelist gives false confidence.
func TestToleranceGuardCatchesEvasionForms(t *testing.T) {
	t.Parallel()
	caught := []string{
		"k := int64(x * 1e6)",     // the old quantCoord reciprocal grid
		"const q = x * 1e5",       // the old weldKey reciprocal grid
		"grid := v * 100000.0",    // spelled-out reciprocal
		"cells := 1.0 / tol",      // one-over-a-named-tolerance
		"n := 1 / weldEps",        // one-over-eps identifier
		"if d < 1e-6 {",           // the classic direct form
		"tol := span*1e-9 + 1e-9", // absolute floor hiding behind a relative term
	}
	for _, line := range caught {
		if !toleranceLiteral(line) {
			t.Errorf("evasion form not caught: %q", line)
		}
	}
	clean := []string{
		"area := w * h",
		"x := 2 * stdmath.Pi",
		"s := v.Scale(1 / norm)",
		"r := 1 / (radius * radius)",
		"i := n * 100",
	}
	for _, line := range clean {
		if toleranceLiteral(line) {
			t.Errorf("ordinary arithmetic falsely flagged: %q", line)
		}
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

// resolveScope maps each listed path to the file that is there now. A path that still exists is
// returned unchanged; one that does not is looked up by basename under kernel/, so a file that
// moved with a package split is still guarded. An unresolvable or ambiguous name is a hard
// failure: silently dropping a hot-path file would make the guard pass by not looking.
func resolveScope(t *testing.T, scope []string) []string {
	t.Helper()
	out := make([]string, 0, len(scope))
	for _, rel := range scope {
		if _, err := os.Stat(rel); err == nil {
			out = append(out, rel)
			continue
		}
		matches := findByBase(t, filepath.Base(rel))
		if len(matches) != 1 {
			t.Fatalf("hot-path file %q is gone and %d files under kernel/ are named %q: %v — "+
				"update the scope list", rel, len(matches), filepath.Base(rel), matches)
		}
		out = append(out, matches[0])
	}
	return out
}

// findByBase returns every non-test .go file under kernel/ with the given base name.
func findByBase(t *testing.T, base string) []string {
	t.Helper()
	var hits []string
	err := filepath.WalkDir("..", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != base {
			return err
		}
		hits = append(hits, p)
		return nil
	})
	if err != nil {
		t.Fatalf("walking kernel/ for %q: %v", base, err)
	}
	return hits
}
