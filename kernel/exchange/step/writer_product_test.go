// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/topo"
)

// #2055: the exported file was a DATA section of raw geometry ending at MANIFOLD_SOLID_BREP,
// with no APPLICATION_CONTEXT, no PRODUCT and no shape representation. Our own reader scans for
// the b-rep entity directly so the round trip passed, but a reader that enters the standard way
// — from SHAPE_DEFINITION_REPRESENTATION down — found nothing and opened an empty model.
//
// Verified against OCCT (an independent reader) while fixing: the pre-fix file yields NO shape
// at all, the post-fix file yields SOLID/6 faces/12 edges/8 vertices. These tests encode the
// structural invariant so CI holds it without OCCT.

// stepEntities indexes a written file by entity id.
func stepEntities(t *testing.T, data []byte) map[int]string {
	t.Helper()
	out := map[int]string{}
	re := regexp.MustCompile(`^#(\d+)=(.*);$`)
	for line := range strings.SplitSeq(string(data), "\n") {
		if m := re.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			id, _ := strconv.Atoi(m[1])
			out[id] = m[2]
		}
	}
	if len(out) == 0 {
		t.Fatal("the exported file has no DATA entities at all")
	}
	return out
}

// findEntity returns the id of the single entity whose body starts with keyword.
func findEntity(t *testing.T, ents map[int]string, keyword string) int {
	t.Helper()
	found := -1
	for id, body := range ents {
		if strings.HasPrefix(body, keyword+"(") {
			if found >= 0 {
				t.Fatalf("%s appears more than once (#%d and #%d)", keyword, found, id)
			}
			found = id
		}
	}
	if found < 0 {
		t.Fatalf("the exported file has no %s — a reader entering the standard way finds no "+
			"geometry and opens an empty model (#2055)", keyword)
	}
	return found
}

// refsOf lists the #ids an entity body references.
func refsOf(body string) []int {
	var out []int
	for _, m := range regexp.MustCompile(`#(\d+)`).FindAllStringSubmatch(body, -1) {
		id, _ := strconv.Atoi(m[1])
		out = append(out, id)
	}
	return out
}

// reachesKeyword walks references from start and reports whether any reachable entity's body
// starts with keyword — the traversal a reader performs.
func reachesKeyword(ents map[int]string, start int, keyword string) bool {
	seen := map[int]bool{}
	stack := []int{start}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[id] {
			continue
		}
		seen[id] = true
		body, ok := ents[id]
		if !ok {
			continue
		}
		if strings.HasPrefix(body, keyword+"(") {
			return true
		}
		stack = append(stack, refsOf(body)...)
	}
	return false
}

// exportBox writes the cube fixture (or an open shell made from it) and returns the file bytes.
func exportBox(t *testing.T, solid bool) []byte {
	t.Helper()
	data, _, err := Writer{}.ExportSolids([]*topo.Body{cubeOrShell(t, solid)},
		exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("ExportSolids: %v", err)
	}
	return data
}

// cubeOrShell returns the imported cube, or an open-shell body over the same faces — the
// surface-model case the writer must represent differently.
func cubeOrShell(t *testing.T, solid bool) *topo.Body {
	t.Helper()
	body := importOneSolid(t, "cube.step")
	if solid {
		return body
	}
	return topo.BodyFromShells(body.Lineage(), false, body.Shells()...)
}

// TestExportedSolidIsReachableFromTheProductStructure walks the file the way a conformant reader
// does — SHAPE_DEFINITION_REPRESENTATION down — and requires the b-rep to be reachable.
func TestExportedSolidIsReachableFromTheProductStructure(t *testing.T) {
	ents := stepEntities(t, exportBox(t, true))
	sdr := findEntity(t, ents, "SHAPE_DEFINITION_REPRESENTATION")

	for _, keyword := range []string{
		"PRODUCT_DEFINITION_SHAPE",
		"PRODUCT_DEFINITION",
		"PRODUCT",
		"APPLICATION_CONTEXT",
		"ADVANCED_BREP_SHAPE_REPRESENTATION",
		"MANIFOLD_SOLID_BREP",
		"CLOSED_SHELL",
		"ADVANCED_FACE",
	} {
		if !reachesKeyword(ents, sdr, keyword) {
			t.Errorf("%s is not reachable from SHAPE_DEFINITION_REPRESENTATION — a reader "+
				"walking the file cannot find it (#2055)", keyword)
		}
	}
}

// The representation must sit in a geometric context that assigns units and an uncertainty; a
// bare GLOBAL_UNIT_ASSIGNED_CONTEXT satisfies neither the schema nor a reader.
func TestExportedRepresentationHasAGeometricContext(t *testing.T) {
	data := exportBox(t, true)
	s := string(data)
	for _, want := range []string{
		"GEOMETRIC_REPRESENTATION_CONTEXT(3)",
		"GLOBAL_UNCERTAINTY_ASSIGNED_CONTEXT",
		"GLOBAL_UNIT_ASSIGNED_CONTEXT",
		"UNCERTAINTY_MEASURE_WITH_UNIT",
		"PLANE_ANGLE_UNIT",
		"SOLID_ANGLE_UNIT",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the exported file declares no %s", want)
		}
	}
	// The representation's context reference must BE that combined instance.
	ents := stepEntities(t, data)
	rep := findEntity(t, ents, "ADVANCED_BREP_SHAPE_REPRESENTATION")
	refs := refsOf(ents[rep])
	ctx := refs[len(refs)-1] // the context is the representation's last parameter
	if !strings.Contains(ents[ctx], "GEOMETRIC_REPRESENTATION_CONTEXT") {
		t.Errorf("the representation's context #%d is %q, want the geometric context", ctx, ents[ctx])
	}
}

// A surface body takes MANIFOLD_SURFACE_SHAPE_REPRESENTATION: a reader rejects a representation
// whose items are of the wrong kind, so an open shell must not claim to be a solid b-rep.
func TestExportedSurfaceBodyUsesTheSurfaceRepresentation(t *testing.T) {
	s := string(exportBox(t, false))
	if strings.Contains(s, "ADVANCED_BREP_SHAPE_REPRESENTATION") {
		t.Error("an open shell was exported as a solid b-rep representation")
	}
	if !strings.Contains(s, "MANIFOLD_SURFACE_SHAPE_REPRESENTATION") {
		t.Error("the surface body has no MANIFOLD_SURFACE_SHAPE_REPRESENTATION")
	}
	if !strings.Contains(s, "SHELL_BASED_SURFACE_MODEL") {
		t.Error("the surface body has no SHELL_BASED_SURFACE_MODEL")
	}
}

// The product carries the name the caller asked for, so a reader's model tree shows something
// meaningful rather than a placeholder.
func TestExportedProductCarriesTheName(t *testing.T) {
	data, _, err := Writer{}.ExportSolids([]*topo.Body{cubeOrShell(t, true)},
		exchange.TranslationOptions{Name: "bracket"})
	if err != nil {
		t.Fatalf("ExportSolids: %v", err)
	}
	if !strings.Contains(string(data), "PRODUCT('bracket','bracket'") {
		t.Error("the exported product does not carry the requested name")
	}
	// With no name the writer falls back rather than emitting an empty product.
	plain := string(exportBox(t, true))
	if !strings.Contains(plain, "PRODUCT('"+defaultProductName+"'") {
		t.Errorf("an unnamed export has no fallback product name")
	}
}

// Mixing solids and shells in one file is reported rather than silently flattened.
func TestMixedBodiesWarn(t *testing.T) {
	_, warns, err := Writer{}.ExportSolids(
		[]*topo.Body{cubeOrShell(t, true), cubeOrShell(t, false)}, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("ExportSolids: %v", err)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "shape representation") {
		t.Errorf("warnings = %v, want one about the shared shape representation", warns)
	}
	// A homogeneous export stays silent.
	_, quiet, err := Writer{}.ExportSolids([]*topo.Body{cubeOrShell(t, true)}, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("ExportSolids: %v", err)
	}
	if len(quiet) != 0 {
		t.Errorf("a solids-only export warned: %v", quiet)
	}
}
