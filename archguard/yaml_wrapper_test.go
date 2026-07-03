// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os/exec"
	"strings"
	"testing"
)

// The YAML-wrapper guard (#1622, audit B11): gopkg.in/yaml.v3 is the project's single
// third-party YAML surface and must be reached ONLY through the yamlcodec wrapper
// (CLAUDE.md dependency rule + ADR-0020), so a library swap or a format quirk (indent
// style, anchor handling, !!binary encoding) is absorbed in one place — the .obk document
// format depends on that stability. Any other package that imports yaml.v3 directly
// bypasses the seam; this test fails unless the package is on the explicit, commented
// allowlist below. Red-verify by adding a raw `gopkg.in/yaml.v3` import to a non-exempt
// package — the test then names it.
var yamlDirectImportAllowlist = map[string]string{
	// The wrapper itself — the one legitimate home of the yaml.v3 dependency.
	"oblikovati.org/persistence/yamlcodec": "the owned wrapper (ADR-0020)",
	// release/version.go parses version.yaml. Routing it through yamlcodec would add a
	// release -> persistence edge (persistence pulls in model/api), coupling a zero-dep
	// version leaf to the document layer. Exempt until B4 (#1615) relocates yamlcodec to a
	// neutral leaf, after which this entry should go and release should use the wrapper.
	"oblikovati.org/release": "version.yaml parsing; awaiting B4 yamlcodec relocation (#1615)",
	// model/feature's emboss_text_test.go marshals a fixture struct for a round-trip
	// assertion — a test helper, not the .obk document path the wrapper guards.
	"oblikovati.org/model/feature": "test-only fixture marshal (emboss_text_test.go)",
}

// yamlV3Path is the third-party YAML import the guard fences off.
const yamlV3Path = "gopkg.in/yaml.v3"

// TestRawYamlImportsAreWrapped fails if a first-party package outside the allowlist imports
// gopkg.in/yaml.v3 directly (in production, internal-test, or external-test files).
func TestRawYamlImportsAreWrapped(t *testing.T) {
	for _, pkg := range packagesImportingYaml(t) {
		if _, ok := yamlDirectImportAllowlist[pkg]; ok {
			continue
		}
		t.Errorf("package %s imports %s directly — route YAML through "+
			"oblikovati.org/persistence/yamlcodec (the owned wrapper, ADR-0020) or add an "+
			"explicit, commented exemption in yamlDirectImportAllowlist (#1622).", pkg, yamlV3Path)
	}
}

// packagesImportingYaml returns every first-party package that lists yaml.v3 among its
// production, internal-test, or external-test imports (one `go list` over the module).
func packagesImportingYaml(t *testing.T) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f",
		`{{.ImportPath}}::{{join .Imports " "}} {{join .TestImports " "}} {{join .XTestImports " "}}`, "./...")
	cmd.Dir = ".."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list imports ./...: %v", err)
	}
	return matchYamlImporters(string(out))
}

// matchYamlImporters keeps the import paths whose import lists contain yaml.v3.
func matchYamlImporters(listing string) []string {
	var pkgs []string
	for _, line := range strings.Split(strings.TrimSpace(listing), "\n") {
		path, imports, ok := strings.Cut(line, "::")
		if ok && strings.Contains(imports, yamlV3Path) {
			pkgs = append(pkgs, path)
		}
	}
	return pkgs
}
