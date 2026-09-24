// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// This file is the #3526 guard. `make gate` is the pre-push gate, and its worth is a single
// claim: it cannot silently cover less than the repo holds. That claim used to rest on a
// hand-typed list of three module paths — so the day someone added a module, the gate would
// have gone green without it, which is the very defect #3526 exists to remove, deferred by
// one module. The gate now DERIVES its module set from `git ls-files`; these tests assert
// the derivation is complete, and that CI's own hand-typed lists have not drifted from it.
//
// Probe used to prove the guard bites (write it, watch it fail, delete it): add
// `experiments/probe/go.mod`, `git add` it, run this package — TestGateReachesEveryTrackedModule
// passes (the derivation picks it up), while pinning GATE_NESTED_MODULES back to a literal
// list in the Makefile makes it FAIL naming `experiments/probe`.

// gateOwnedLegs are the tracked modules that have a leg of their own in `make gate`, so the
// derived nested-module set deliberately excludes them.
var gateOwnedLegs = map[string]string{
	".":    "make ci (fmt-check vet lint cover)",
	"head": "make test-head (cgo module, own Makefile, native deps)",
}

// headFixturePrefix holds the c-shared add-in fixture modules. They get no leg of their own
// because head's own loader tests compile them; TestGateReachesEveryTrackedModule proves that
// rather than trusting it.
const headFixturePrefix = "head/internal/addinhost/testdata/"

// runInRepoRoot runs a tool at the repo root and returns its stdout. A missing tool skips
// (a Windows runner may have no make); a tool that RUNS and fails is a hard error, because
// then the guard would be reporting on a measurement it did not get.
func runInRepoRoot(t *testing.T, name string, args ...string) string {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s is not installed; the gate-coverage guard needs it to measure what `make gate` runs", name)
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = ".."
	out, err := cmd.Output()
	if err != nil {
		// Output() keeps the failing command's stderr on the ExitError; printing it is the difference
		// between "exit status 2" and the reason. Without it, make 3.81 reading a `#` inside a function
		// call as a comment cost PR #3558 a whole CI round to diagnose.
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			t.Fatalf("running %s %v in the repo root: %v\n%s", name, args, err, exit.Stderr)
		}
		t.Fatalf("running %s %v in the repo root: %v", name, args, err)
	}
	return string(out)
}

// moduleDirOfGoMod turns a tracked go.mod path into its module directory, reporting the root
// module as ".".
func moduleDirOfGoMod(goModPath string) string {
	dir := strings.TrimSuffix(filepath.ToSlash(goModPath), "go.mod")
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		return "."
	}
	return dir
}

// trackedModuleDirs lists the directory of every go.mod git tracks in this repo.
func trackedModuleDirs(t *testing.T) []string {
	t.Helper()
	var dirs []string
	for _, f := range strings.Fields(runInRepoRoot(t, "git", "ls-files", "*go.mod")) {
		dirs = append(dirs, moduleDirOfGoMod(f))
	}
	if len(dirs) == 0 {
		t.Fatal("git ls-files '*go.mod' matched nothing: this guard cannot measure gate coverage, " +
			"and neither can `make gate` (expected at least go.mod and head/go.mod)")
	}
	return dirs
}

// gateNestedModules asks the Makefile which nested modules `make gate` will run. It reads the
// gate's OWN derivation rather than repeating the expression here, so this is a measurement of
// the gate and not a copy of it.
func gateNestedModules(t *testing.T) []string {
	t.Helper()
	mods := strings.Fields(runInRepoRoot(t, "make", "--no-print-directory", "print-gate-modules"))
	if len(mods) == 0 {
		t.Fatal("make print-gate-modules printed nothing: `make gate` would run no nested module at all " +
			"(expected the three model/exchange/translators/* modules)")
	}
	return mods
}

// addinhostTestSources is every _test.go in head's add-in loader package, concatenated. The
// fixture allowlist is checked against it.
func addinhostTestSources(t *testing.T) string {
	t.Helper()
	paths, err := filepath.Glob("../head/internal/addinhost/*_test.go")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no test sources at head/internal/addinhost (glob err %v): the fixture-module "+
			"allowlist cannot be verified", err)
	}
	var all strings.Builder
	for _, p := range paths {
		src, rerr := os.ReadFile(p)
		if rerr != nil {
			t.Fatalf("reading %s: %v", p, rerr)
		}
		all.Write(src)
	}
	return all.String()
}

// gateReachedModules maps every tracked module `make gate` runs to the leg that runs it.
func gateReachedModules(t *testing.T, tracked []string) map[string]string {
	t.Helper()
	reached := map[string]string{}
	for dir, leg := range gateOwnedLegs {
		reached[dir] = leg
	}
	for _, m := range gateNestedModules(t) {
		reached[m] = "make test-nested-modules"
	}
	loaderTests := addinhostTestSources(t)
	for _, dir := range tracked {
		if strings.HasPrefix(dir, headFixturePrefix) && strings.Contains(loaderTests, filepath.Base(dir)) {
			reached[dir] = "compiled by head's add-in loader tests"
		}
	}
	return reached
}

// TestGateReachesEveryTrackedModule fails when the repo tracks a Go module that no leg of
// `make gate` runs. "The gate reaches every module" has to be a measurement over the tree,
// never a claim about a variable someone remembered to edit (#3526).
func TestGateReachesEveryTrackedModule(t *testing.T) {
	tracked := trackedModuleDirs(t)
	reached := gateReachedModules(t, tracked)
	var unreached []string
	for _, dir := range tracked {
		if reached[dir] == "" {
			unreached = append(unreached, dir)
		}
	}
	if len(unreached) > 0 {
		t.Fatalf("tracked Go module(s) %v are not run by any `make gate` leg.\n"+
			"A nested module is picked up automatically by `make test-nested-modules`; if this one "+
			"needs a leg of its own (cgo, a private toolchain), add it to the Makefile and to "+
			"gateOwnedLegs. Reached today: %v", unreached, reached)
	}
}

// ciWorkflowText is .github/workflows/ci.yml, the workflow that carries the hand-typed module
// lists this guard compares against.
func ciWorkflowText(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("../.github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("reading .github/workflows/ci.yml: %v", err)
	}
	return string(src)
}

// yamlContinuation joins a shell line continued over several YAML lines into one line.
var yamlContinuation = regexp.MustCompile(`\\\s*\n\s*`)

// ciCoverageLoopModules is the module list in ci.yml's "Translator-module coverage" step.
func ciCoverageLoopModules(t *testing.T) []string {
	t.Helper()
	flat := yamlContinuation.ReplaceAllString(ciWorkflowText(t), " ")
	found := regexp.MustCompile(`for m in ([^;]+); do`).FindAllStringSubmatch(flat, -1)
	if len(found) != 1 {
		t.Fatalf("expected exactly one `for m in ...; do` module loop in ci.yml, found %d: "+
			"the guard no longer knows which list to compare", len(found))
	}
	return strings.Fields(found[0][1])
}

// modulesInput reads a `modules: a b c` YAML input line, returning the paths it names.
func modulesInput(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "modules:") {
		return nil, false
	}
	return strings.Fields(strings.TrimPrefix(trimmed, "modules:")), true
}

// ciAPIContractModules is the `modules:` input ci.yml hands the api-contract action for the
// nested modules — the only such input that names a path, the others being `.` or `head`.
func ciAPIContractModules(t *testing.T) []string {
	t.Helper()
	var lists [][]string
	for _, line := range strings.Split(ciWorkflowText(t), "\n") {
		mods, ok := modulesInput(line)
		if ok && slices.ContainsFunc(mods, func(m string) bool { return strings.Contains(m, "/") }) {
			lists = append(lists, mods)
		}
	}
	if len(lists) != 1 {
		t.Fatalf("expected exactly one api-contract `modules:` input naming nested module paths "+
			"in ci.yml, found %d: %v", len(lists), lists)
	}
	return lists[0]
}

// apiRequiringModules are the nested modules whose go.mod requires the Apache-2.0 contract, so
// CI has to rewrite their `replace` before their tests can build. olecf is standard-library
// only and correctly absent — the shorter CI list is deliberate, not drift.
func apiRequiringModules(t *testing.T, mods []string) []string {
	t.Helper()
	var need []string
	for _, m := range mods {
		src, err := os.ReadFile(filepath.Join("..", m, "go.mod"))
		if err != nil {
			t.Fatalf("reading %s/go.mod: %v", m, err)
		}
		if strings.Contains(string(src), "oblikovati.org/api") {
			need = append(need, m)
		}
	}
	return need
}

// TestCIWiresTheAPIContractIntoEveryModuleThatNeedsIt fails when a nested module gains a
// dependency on the Apache-2.0 contract and nobody adds it to ci.yml's api-contract input —
// its tests would then fail in CI on a `replace` pointing at a sibling checkout CI does not
// have. Derived from the go.mod files, so a new module is covered the day it lands (#3526).
func TestCIWiresTheAPIContractIntoEveryModuleThatNeedsIt(t *testing.T) {
	want := append([]string{"."}, apiRequiringModules(t, gateNestedModules(t))...)
	got := slices.Clone(ciAPIContractModules(t))
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(want, got) {
		t.Fatalf("ci.yml gives the api-contract action %v, but the modules that require "+
			"oblikovati.org/api are %v.\nUpdate the `modules:` input in "+
			".github/workflows/ci.yml (#3526).", got, want)
	}
}

// TestCIRunsTheSameNestedModulesAsTheGate fails when ci.yml's hand-typed coverage loop and the
// gate's derived set disagree. Two lists that must match and only one of them derived is a
// drift waiting to happen; this is the assertion that makes CI's copy a copy on purpose.
func TestCIRunsTheSameNestedModulesAsTheGate(t *testing.T) {
	gate := slices.Clone(gateNestedModules(t))
	ci := slices.Clone(ciCoverageLoopModules(t))
	slices.Sort(gate)
	slices.Sort(ci)
	if !slices.Equal(gate, ci) {
		t.Fatalf("ci.yml's \"Translator-module coverage\" loop runs %v but `make gate` derives %v.\n"+
			"CI would then report coverage over a different module set than the pre-push gate ran; "+
			"update the loop in .github/workflows/ci.yml to the derived set (#3526).", ci, gate)
	}
}
