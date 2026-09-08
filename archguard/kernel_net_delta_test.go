// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// "Every kernel PR reports the net change in recognizers, tolerance constants, fallback sites,
// and type assertions. The net is ≤ 0 unless an ADR says why." (kernel ground rules.)
//
// Nothing counted any of the four (#2184), so "net ≤ 0" was an intention rather than a fact.
// This is the reporting layer: four numbers, pinned, and a failure on ANY change — up OR down.
// A rise is not forbidden, it is made DELIBERATE: you edit the pin, and the diff to the pin is
// the report the rule asks for. A fall is caught too, because a floor nobody lowers stops being
// a floor.
//
// Three of the four are already measured by a dedicated ratchet in this package, and this test
// reads those rather than counting again — one mechanism per quantity:
//
//	tolerance constants  toleranceDebt      (TestNoUnjustifiedAbsoluteEpsilons, #2189)
//	type assertions      geomSwitchDebt     (TestGeometryKindSwitchesLiveInGeom, #2188)
//	recognizers          the dispatch tables TestNoFirstFitDispatchLadders pins (#2186)
//	fallback sites       diag.Code declarations under kernel/
//
// Fallback sites are counted as declared diag.Code kinds because the rules require a degradation
// to BE one: "Never degrade silently. A fallback, approximation, or dropped element is a
// diag.Defect that reaches feature health, the API, and the UI." That makes the count honest in
// a way a grep for the word "fallback" would not be — but it also means a rise can be an
// IMPROVEMENT, when a previously-silent degradation is finally reported. That is exactly why
// this test fails on any change instead of only on a rise: the number is a prompt to explain,
// not a limit to obey.

// kernelNetDeltaPin is the checked-in baseline. Update it in the same commit as the change that
// moves it, and say in the PR which direction each moved and why.
var kernelNetDeltaPin = map[string]int{
	// 232 → 231 (2026-09-06, ADR-0061 stage 4): a FALL — the ruled∩quadric gate's branch-separation
	// margin (a twentieth of the largest gap) is gone; the gate reads the exact minimum of the gap
	// against the stitch resolution instead, and the near-pinch crossings it refused are exact.
	// 226 → 216 (2026-09-07, ADR-0061 stage 4): a FALL of 10 — the deleted recognizer drivers carried
	// their own calibrated tolerances (the corner-junction weld, the scallop and boss wall snaps, the
	// cap-crossing corner bracket), and the general pipeline reads the model-relative resolution instead.
	"tolerance-constants": 216,
	// 765 → 754 (2026-09-05, ADR-0061 stage 2): a FALL — the analytic half-space pipeline is deleted,
	// and its per-primitive dispatch took eleven geometry-kind assertions with it.
	// 754 → 746 (2026-09-06, ADR-0061 stage 4): a FALL — restricting an edge's curve to its own
	// sub-range moved out of the stitch and into geom.SubCurve, where the rules put a switch over
	// curve kinds; the stitch now asks for the piece and gets back whatever kind owns it.
	// 727 → 692 (2026-09-07, ADR-0061 stage 4): a FALL of 35 — the 15 deleted brep drivers each opened
	// by asserting its operands' surface kinds (a cylinder side, a cone frustum, a bare sphere, a
	// planar cap), which is how a per-pair recognizer recognises. The general pipeline classifies a
	// face by its chart, not by a type switch on its surface.
	"type-assertions": 692,
	// 37 → 11 (2026-09-07, ADR-0061 stage 4): a FALL of 26 — curvedExactPaths is DELETED. It was an
	// ordered first-fit ladder of 26 bespoke recognizers tried before the general per-face pipeline,
	// the shape the ground rules forbid ("dispatch is a classification that selects exactly one path"),
	// and every pair it claimed — the ruled crossings, the equal-radius Steinmetz family, the drill
	// through-hole and the cylinder boss, the four cap-crossing slices, the partial rim and its corner
	// junction, the coaxial ball and rod — now goes through brep's one dispatch. The 15 brep driver
	// files behind them (~2700 lines) went with them, and every corpus row they carried was re-pointed
	// at the general entry rather than deleted.
	"recognizers": 11, // the tessellator's specialCurvedMeshers ladder, the last one left
	// 28 → 29 (2026-09-03, ADR-0061): CodeBooleanAnalyticInvalid. A RISE that is an improvement — the
	// public curved-boolean entry had no Validate post-condition, so a recognizer returning a torn body
	// shipped it silently; the degradation is now refused AND reported.
	// 29 → 30 (2026-09-05, ADR-0061): CodeBooleanNoExactCurvedPath. A RISE that is an improvement — the
	// guarded curved entry's fourth exit, "no exact path claims this", returned silently while the other
	// three reported, so a boolean with a curved operand could fall to triangle soup with nothing
	// downstream able to say why. The degradation is the same; it is now named.
	// 30 → 31 (2026-09-06, ADR-0061 stage 4): CodeBooleanWindingReject. A RISE that names a degradation
	// nothing reported before: a boolean result with a face wound against its outward normal, which the
	// per-edge validity test admits and which shipped as a valid solid meshing as its own complement.
	// A recognizer body that fails the winding certificate now demotes to the general pipeline and
	// says so; a general-pipeline body that fails declines and says so.
	// 26 → 24 (2026-09-07, ADR-0061 stage 4): a FALL — CodeImprintNearPinchDeclined and the
	// near-pinch gate that recorded it are deleted. The gate declined a crossing whose two lens loops
	// leave a narrow neck so the bespoke Steinmetz constructor could take it below the snap ceiling and
	// the faceted route above; both destinations are gone and the general trace resolves the neck
	// itself, so the whole band is ordinary geometry with nothing to report.
	// 24 → 25 (2026-09-08, ADR-0061 stage 5): CodeTrimIgnoredFullDomain. A RISE that names a
	// degradation nothing reported before, which is what the ratchet exists to allow. The curved-face
	// router ends at the surface's WHOLE parametric domain for a boundary no wrapping mesher
	// recognised; on a TRIMMED face that mesh carries material the face does not have and omits the
	// face's own boundary, and it shipped silently. The stage-5 booleans — a ring meeting a ball, a
	// ring bored by a coaxial shaft — are exact B-reps whose meshes land there, so what was an
	// invisible wrong picture is now a reported one. The degradation is the same; it is now named.
	"fallback-sites": 25,
}

func TestKernelNetDelta(t *testing.T) {
	t.Parallel()
	got := map[string]int{
		"tolerance-constants": sumInts(toleranceDebt),
		"type-assertions":     sumInts(geomSwitchDebt),
		"recognizers":         countRecognizers(t),
		"fallback-sites":      countDiagCodes(t),
	}
	var moved []string
	for _, k := range []string{"tolerance-constants", "type-assertions", "recognizers", "fallback-sites"} {
		want, have := kernelNetDeltaPin[k], got[k]
		if want == have {
			continue
		}
		dir := "ROSE"
		if have < want {
			dir = "fell"
		}
		moved = append(moved, "  "+k+": "+strconv.Itoa(want)+" → "+strconv.Itoa(have)+
			"  ("+dir+" by "+strconv.Itoa(abs(have-want))+")")
	}
	if len(moved) > 0 {
		t.Errorf("kernel net delta moved — report it and update kernelNetDeltaPin in this commit. "+
			"A RISE needs a reason in the PR, and an ADR if it is a new engine, recognizer or "+
			"tolerance rather than a reported degradation; a FALL just needs the pin lowered so it "+
			"holds:\n%s", strings.Join(moved, "\n"))
	}
}

// countRecognizers counts the entries of the ordered dispatch tables — each entry is one
// analytic recognizer, and the count is what "generality over special cases" is measured by.
func countRecognizers(t *testing.T) int {
	t.Helper()
	fset := token.NewFileSet()
	n := 0
	for file := range dispatchLadders {
		f, err := parser.ParseFile(fset, filepath.Join("..", file), nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		ast.Inspect(f, func(node ast.Node) bool {
			cl, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			at, ok := cl.Type.(*ast.ArrayType)
			if !ok || at.Len != nil {
				return true
			}
			if _, isFunc := at.Elt.(*ast.FuncType); isFunc {
				n += len(cl.Elts)
			}
			return true
		})
	}
	if n == 0 {
		t.Fatal("counted no recognizers — the dispatch tables moved; update dispatchLadders")
	}
	return n
}

// countDiagCodes counts the declared diag.Code kinds under kernel/: one per way the kernel can
// degrade and say so.
func countDiagCodes(t *testing.T) int {
	t.Helper()
	fset := token.NewFileSet()
	n := 0
	err := filepath.WalkDir(filepath.Join("..", "kernel"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", p, err)
		}
		ast.Inspect(f, func(node ast.Node) bool {
			vs, ok := node.(*ast.ValueSpec)
			if !ok || vs.Type == nil {
				return true
			}
			sel, ok := vs.Type.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Code" {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "diag" {
				n += len(vs.Names)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking kernel/: %v", err)
	}
	return n
}

// sumInts totals a debt map's counts.
func sumInts(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
