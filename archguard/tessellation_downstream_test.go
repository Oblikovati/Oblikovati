// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Tessellation is a DERIVED view of the B-rep, so no modelling or topological decision may read
// it: "not classification, containment, mass properties, validity, section curves, or pick →
// reference key" (kernel ground rules). A mesh is an approximation with a chord tolerance; a
// decision made from one is an approximation dressed as a fact, and it changes when the display
// quality changes.
//
// The rule had no mechanical guard (#2187), and the kernel/ops split is what finally makes one
// possible: before it, "does the query layer import the tessellator" was not a question the
// import graph could answer, because both were the same package.
//
// The answer today is that it does, in 18 files. That is the M48/C3 backlog (#3420 and its
// children #3459-#3463), not something to declare fixed — so this is a RATCHET, the same shape
// as toleranceDebt: the count per file is a cap, a rise fails, a fall demands the number come
// down, and a file that stops importing the tessellator must lose its entry.
//
// validate/ is deliberately absent: it is already clean, and validity is the decision this rule
// protects most.

// tessellationDebt counts, per file, the references to the tessellator in packages whose job is
// to DECIDE something. Baseline 2026-09-01. It may only shrink.
var tessellationDebt = map[string]int{
	// query/ — mass properties, containment, identity and pick. #3420: these must integrate the
	// analytic B-rep, not a mesh of it. An oracle that gates a result has to be more exact than
	// the result it gates.
	"query/aliases.go":               2,
	"query/body_mesh_diagnostics.go": 1,
	"query/body_query.go":            2,
	"query/identical_bodies.go":      1,
	"query/inertia.go":               3,
	"query/massprops.go":             1,
	"query/pick_analytic.go":         1,
	"query/precise_range_box.go":     1,
	"query/shell_query.go":           2,
	"query/visible_edges.go":         1,

	// boolean/ — the mesh-arrangement operand path (ADR-0052/0056). #3459-#3462 remove the
	// tessellation-first arrangement input once analytic reconstruction covers it.
	"boolean/aliases.go":              1,
	"boolean/csg_body.go":             1,
	"boolean/meshbool_imprint.go":     1,
	"boolean/meshbool_reconstruct.go": 1,
	"boolean/meshbool_soup.go":        1,
	"boolean/meshbool_tagged_soup.go": 2,

	// heal/ — repair reading a mesh to decide what to snap or rebuild.
	"heal/pcurve_reconstruct.go": 1,
	"heal/snap_edges.go":         1,
}

// decidingFamilies are the kernel/ops families whose output is a DECISION rather than a derived
// view, so a tessellator reference in one is what this guard measures.
var decidingFamilies = []string{"query", "boolean", "heal", "validate", "surface", "transform"}

func TestTessellationIsDownstream(t *testing.T) {
	t.Parallel()
	got := scanTessellationDebt(t)
	var rose, fell, stale []string
	for path, n := range got {
		switch owed := tessellationDebt[path]; {
		case n > owed:
			rose = append(rose, path+": "+strconv.Itoa(n)+" tessellator reference(s), budget "+strconv.Itoa(owed))
		case n < owed:
			fell = append(fell, `"`+path+`": `+strconv.Itoa(n)+",")
		}
	}
	for path := range tessellationDebt {
		if _, ok := got[path]; !ok {
			stale = append(stale, path)
		}
	}
	sort.Strings(rose)
	sort.Strings(fell)
	sort.Strings(stale)
	if len(rose) > 0 {
		t.Errorf("a deciding package read the TESSELLATOR — classification, containment, mass "+
			"properties, validity, section curves and pick must read the analytic B-rep "+
			"(#2187, #3420):\n%s", strings.Join(rose, "\n"))
	}
	if len(fell) > 0 {
		t.Errorf("tessellation debt FELL — good; lower these tessellationDebt entries so the "+
			"ratchet holds the new floor:\n%s", strings.Join(fell, "\n"))
	}
	if len(stale) > 0 {
		t.Errorf("these tessellationDebt files no longer read the tessellator — DELETE their "+
			"entries:\n  %s", strings.Join(stale, "\n  "))
	}
}

// scanTessellationDebt counts tessellator references per file in the deciding families. The key
// is the path relative to kernel/ops, which is where these families live.
func scanTessellationDebt(t *testing.T) map[string]int {
	t.Helper()
	got := map[string]int{}
	for _, fam := range decidingFamilies {
		dir := filepath.Join("..", "kernel", "ops", fam)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("reading %s: %v — update decidingFamilies if a family was renamed", dir, err)
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			src, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("reading %s/%s: %v", dir, name, err)
			}
			if n := codeReferences(string(src), "tessellate."); n > 0 {
				got[fam+"/"+name] = n
			}
		}
	}
	return got
}

// codeReferences counts occurrences of sel in CODE only. A doc comment that names the
// tessellator to say a decision must not read it is the opposite of a violation, so counting it
// would punish exactly the comment this rule wants written.
func codeReferences(src, sel string) int {
	n := 0
	for _, line := range strings.Split(src, "\n") {
		code, _, _ := strings.Cut(line, "//")
		n += strings.Count(code, sel)
	}
	return n
}
