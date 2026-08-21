// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// apiModuleDir resolves the on-disk directory of the Apache-2.0 oblikovati.org/api module,
// honoring the go.mod `replace` (a sibling checkout / go.work locally, `_api` under CI via the
// api-contract action). The archguard guards that read the contract source used to hardcode
// ../../Oblikovati.API, a LOCAL-DEV layout CI does not have — so they skipped in CI and could not
// catch, e.g., a newly declared contract interface with no host assertion (#1976 slipped through
// this way; the completeness check only ran on a developer's machine). `go list -m` reports the
// same directory whichever layout is in effect, so resolving through it lets these guards run
// everywhere the contract is present — including CI, where it is checked out into _api.
//
// ok is false (skip, don't fail) only when the module genuinely cannot be located on disk — the
// contract absent is not an architecture violation.
func apiModuleDir(t *testing.T) (string, bool) {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "oblikovati.org/api").Output()
	if err != nil {
		return "", false
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", false
	}
	if _, err := os.Stat(dir); err != nil {
		return "", false
	}
	return dir, true
}

// apiSubdir resolves a package directory inside the api module (e.g. "contract", "wire"), skipping
// the calling test when the contract module is not checked out.
func apiSubdir(t *testing.T, sub string) string {
	t.Helper()
	dir, ok := apiModuleDir(t)
	if !ok {
		t.Skipf("api module not resolvable (go list -m oblikovati.org/api); contract source absent")
	}
	return filepath.Join(dir, sub)
}
