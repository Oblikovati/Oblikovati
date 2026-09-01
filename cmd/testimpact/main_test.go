// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRunSelectsThisModulesOwnPackages drives the real go-list and git seams against
// this checkout: the tool is only useful if it works on the module it ships in. An
// empty -base limits the change set to the working tree, so the test does not need a
// remote branch that a shallow CI checkout may not have.
func TestRunSelectsThisModulesOwnPackages(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-root", "..", "-base", ""}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line != "" && !strings.HasPrefix(line, "oblikovati.org/") {
			t.Errorf("selected %q, want only oblikovati.org/... import paths", line)
		}
	}
}

func TestRunRejectsAnUnknownFlag(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-nonsense"}, &out); err == nil {
		t.Fatal("run() succeeded, want a flag-parse error")
	}
}

func TestPrintImpactedReportsAnUnresolvableRoot(t *testing.T) {
	var out bytes.Buffer
	if err := printImpacted(&out, "\x00not-a-directory", ""); err == nil {
		t.Fatal("printImpacted() succeeded, want an error for an unusable root")
	}
}
