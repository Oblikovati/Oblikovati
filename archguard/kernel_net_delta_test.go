// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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
//	recognizers          the classification arms (and any surviving dispatch table) below
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
	// 216 → 214 (2026-09-08, ADR-0061 stage 5): a FALL — the torus band loft's two calibrated spiric
	// coefficients went with the guard that used them. It accepted a pair of boundaries by asking
	// whether two spiric arcs were the opposite roots of ONE plane's section, which needed a tolerance
	// on each coefficient; it now asks whether each boundary goes the whole way round the tube, which
	// is a NET turn against half a period and needs none.
	"tolerance-constants": 214,
	// 765 → 754 (2026-09-05, ADR-0061 stage 2): a FALL — the analytic half-space pipeline is deleted,
	// and its per-primitive dispatch took eleven geometry-kind assertions with it.
	// 754 → 746 (2026-09-06, ADR-0061 stage 4): a FALL — restricting an edge's curve to its own
	// sub-range moved out of the stitch and into geom.SubCurve, where the rules put a switch over
	// curve kinds; the stitch now asks for the piece and gets back whatever kind owns it.
	// 727 → 692 (2026-09-07, ADR-0061 stage 4): a FALL of 35 — the 15 deleted brep drivers each opened
	// by asserting its operands' surface kinds (a cylinder side, a cone frustum, a bare sphere, a
	// planar cap), which is how a per-pair recognizer recognises. The general pipeline classifies a
	// face by its chart, not by a type switch on its surface.
	// 692 → 691 (2026-09-08, ADR-0061 stage 5): a FALL — the curved-face router's `s.(geom.Torus)`
	// went with torusComplementMesh. An outerless face on any periodic surface is now meshed from the
	// chart it carries, so the router asks what the FACE records, not what its surface is.
	// 691 → 684 (2026-09-08, ADR-0061 stage 5): a FALL of 7 — the same seven geometry-kind assertions
	// the curved-trim classification collapsed, counted by the net-delta ratchet.
	"type-assertions": 684,
	// 37 → 11 (2026-09-07, ADR-0061 stage 4): a FALL of 26 — curvedExactPaths is DELETED. It was an
	// ordered first-fit ladder of 26 bespoke recognizers tried before the general per-face pipeline,
	// the shape the ground rules forbid ("dispatch is a classification that selects exactly one path"),
	// and every pair it claimed — the ruled crossings, the equal-radius Steinmetz family, the drill
	// through-hole and the cylinder boss, the four cap-crossing slices, the partial rim and its corner
	// junction, the coaxial ball and rod — now goes through brep's one dispatch. The 15 brep driver
	// files behind them (~2700 lines) went with them, and every corpus row they carried was re-pointed
	// at the general entry rather than deleted.
	// 11 → 12 (2026-09-08, ADR-0061 stage 5): a RISE that is a CORRECTION OF THE MEASUREMENT, not new
	// code, and it is the honest number the guard's own words ask for ("bespoke shapes the general
	// pipeline has not yet absorbed"). The old 11 counted LADDER ENTRIES, and an entry was never one
	// recognizer: entry 0 recognized TWO cone shapes (an apex cap and an apex-collapsed sector), and
	// three entries read three RIM FORMS into one buildSphereCap. Counting the shape recognizers behind
	// the arms (curvedTrimRecognizers) gives 12 BEFORE this slice and 12 after: specialCurvedMeshers is
	// gone and its eleven entries are eight classification arms, but no bespoke SHAPE was deleted. What
	// was deleted is builder duplication — coneApexFan and coneSectorFan became one apexFan, and
	// sphereZoneCapFan, sphereSeamedCapFan, notchedRimBandMesh, twoClosedRimBandMesh and SphereCapFan
	// became arms over shared builders. The number falls when a SHAPE goes, which is what it is for.
	"recognizers": 12,
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
	// 25 → 26 (2026-09-08, ADR-0061 stage 5, third slice): CodeSectionConditioningDemotion. A RISE that
	// names a degradation nothing reported before. The analytic intersector refused two different things
	// with one anonymous ok=false: "no bucket claims this pair", which is the ordinary case and no loss
	// at all, and a CONDITIONING demotion — the closed form applies to the pair and cannot name its own
	// answer at these numbers, so the exact pipeline gives up ground it normally holds. Only the second
	// is a fallback, and it was indistinguishable from the first. geom now returns the reason
	// (geom.SectionDecline) and brep records it as a Defect naming which certificate refused; the
	// ordinary refusal still records nothing, because a diagnostic that fires on every marched boolean
	// in the system is noise.
	// 26 → 27 (2026-09-08, ADR-0061 stage 5, cocylindrical wall merge):
	// CodeCocylindricalMergeUndecided. A RISE that names a degradation nothing reported before. Two
	// kept faces on ONE surface whose shared boundary dissolves are one face, and the merged face's
	// parametric trim is the union of the two in the covering space. Where the fused loops do not
	// determine that trim, ADR-0063 refuses to guess a side — and the pair was then left as two faces
	// with nothing said. It now says so, and the merge is post-conditioned on a chart it verified
	// rather than shipping one nobody did.
	// 27 → 28 (2026-09-08, ADR-0061 stage 5, cocylindrical wall merge, review round 1):
	// CodeMeshNotWatertight. A RISE that names a degradation nothing reported before. Every per-face
	// mesher certifies its own patch, but nothing certified the BODY: two faces can each mesh
	// correctly and still discretise the boundary they SHARE differently, and the crack that leaves
	// was invisible until somebody counted free edges. TessellateBody now carries its own
	// post-condition — a closed solid's mesh is a closed surface — and reports the tear with the
	// faces it touches. The check reads the B-REP for closure, so it cannot fire on a body that is
	// genuinely open.
	"fallback-sites": 28,
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

// curvedTrimRecognizers is what "recognizer" MEANS once a ladder becomes a classification: the bespoke
// SHAPE recognizers still standing behind the arms, keyed by the arm that selects them. An arm is not
// one recognizer — kindConeApexFan reads two cone topologies, kindSphereCapFan three rim forms,
// kindRuledBandLoft two band shapes — so counting arms would have counted a MERGE as a deletion, which
// is how "11 → 8" first got written here. Every name below must be a function declared in
// kernel/ops/tessellate and every key must be a case of specialCurvedMesh's switch; the test checks
// both, so a recognizer cannot be renamed or dropped without moving this number.
var curvedTrimRecognizers = map[string][]string{
	"kindConeApexFan":     {"coneApexTrimOf", "faceIsConeApexCap"},
	"kindSphereCapFan":    {"planarCircleCapRim", "poleSeamedCapRim", "multiArcSeamCapRim"},
	"kindSphereZoneBand":  {"sphereBeltTrimOf"},
	"kindSpherePatch":     {"spherePatchTrimOf"},
	"kindRuledBandLoft":   {"hasTwoClosedRimsNoOpen", "hasFullCircleAndNotchedRim"},
	"kindSpiricBand":      {"spiricTubeTrimOf"},
	"kindTwoRimHoledBand": {"twoRimHoledTrimOf"},
	"kindWedgeBand":       {"wedgeBandTrimOf"},
}

// curvedTrimSwitch is where those arms are selected; its case clauses must match the keys above.
const curvedTrimSwitch = "kernel/ops/tessellate/tessellate_trim_special.go:specialCurvedMesh"

// countRecognizers counts the entries of the ordered dispatch tables plus the shape recognizers behind
// the classification arms that replaced them — the count "generality over special cases" is measured by.
func countRecognizers(t *testing.T) int {
	t.Helper()
	assertCurvedTrimArmsMatchSwitch(t)
	assertRecognizersAreDeclared(t)
	n := countLadderEntries(t)
	for _, names := range curvedTrimRecognizers {
		n += len(names)
	}
	if n == 0 {
		t.Fatal("counted no recognizers — the dispatch tables moved; update dispatchLadders/curvedTrimRecognizers")
	}
	return n
}

// assertCurvedTrimArmsMatchSwitch keeps the registry from listing a phantom arm or missing a new one.
func assertCurvedTrimArmsMatchSwitch(t *testing.T) {
	t.Helper()
	file, fn, _ := strings.Cut(curvedTrimSwitch, ":")
	arms := switchCaseNames(t, parseKernelFile(t, file), fn)
	for _, arm := range arms {
		if _, ok := curvedTrimRecognizers[arm]; !ok {
			t.Errorf("%s selects %s but curvedTrimRecognizers does not list its recognizers", fn, arm)
		}
	}
	for arm := range curvedTrimRecognizers {
		if !slices.Contains(arms, arm) {
			t.Errorf("curvedTrimRecognizers lists %s but %s has no such case", arm, fn)
		}
	}
}

// switchCaseNames returns the identifier of every case clause of the switch inside the named function.
// A default clause carries no expression and is not an arm — it is where the general pipeline takes
// the face.
func switchCaseNames(t *testing.T, f *ast.File, fn string) []string {
	t.Helper()
	var names []string
	ast.Inspect(f, func(n ast.Node) bool {
		decl, ok := n.(*ast.FuncDecl)
		if !ok || decl.Name.Name != fn {
			return true
		}
		ast.Inspect(decl.Body, func(inner ast.Node) bool {
			cc, isCase := inner.(*ast.CaseClause)
			if !isCase {
				return true
			}
			for _, e := range cc.List {
				if id, isIdent := e.(*ast.Ident); isIdent {
					names = append(names, id.Name)
				}
			}
			return true
		})
		return false
	})
	return names
}

// assertRecognizersAreDeclared keeps the registry honest: every name it counts must still be a
// function of kernel/ops/tessellate, so deleting one MOVES the number instead of leaving it stale.
func assertRecognizersAreDeclared(t *testing.T) {
	t.Helper()
	declared := packageFuncNames(t, filepath.Join("..", "kernel", "ops", "tessellate"))
	for arm, names := range curvedTrimRecognizers {
		for _, n := range names {
			if !declared[n] {
				t.Errorf("curvedTrimRecognizers counts %s for %s, but no such function is declared in "+
					"kernel/ops/tessellate — delete the entry with the recognizer", n, arm)
			}
		}
	}
}

// packageFuncNames is every function declared in the package's non-test files.
func packageFuncNames(t *testing.T, dir string) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, parseErr := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, e.Name()), nil, 0)
		if parseErr != nil {
			t.Fatalf("parsing %s: %v", e.Name(), parseErr)
		}
		for _, d := range f.Decls {
			if fd, isFunc := d.(*ast.FuncDecl); isFunc {
				names[fd.Name.Name] = true
			}
		}
	}
	return names
}

// countLadderEntries counts the []func entries of every registered first-fit ladder.
func countLadderEntries(t *testing.T) int {
	t.Helper()
	n := 0
	for file := range dispatchLadders {
		ast.Inspect(parseKernelFile(t, file), func(node ast.Node) bool {
			cl, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			at, isSlice := cl.Type.(*ast.ArrayType)
			if isSlice && at.Len == nil {
				if _, isFunc := at.Elt.(*ast.FuncType); isFunc {
					n += len(cl.Elts)
				}
			}
			return true
		})
	}
	return n
}

// parseKernelFile parses one file of the kernel module, failing the test rather than returning an error.
func parseKernelFile(t *testing.T, file string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", file), nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}
	return f
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
