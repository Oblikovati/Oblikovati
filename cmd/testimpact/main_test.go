// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRunSelectsThisModulesOwnPackages drives the real go-list and git seams against
// this checkout: the tool is only useful if it works on the module it ships in.
func TestRunSelectsThisModulesOwnPackages(t *testing.T) {
	var out bytes.Buffer
	// An empty base limits the change set to the working tree, so the test does not
	// depend on a remote branch existing in a shallow CI checkout.
	if err := run(&out, "..", ""); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line != "" && !strings.HasPrefix(line, "oblikovati.org/") {
			t.Errorf("selected %q, want only oblikovati.org/... import paths", line)
		}
	}
}

func TestRunReportsAnUnresolvableRoot(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out, "\x00not-a-directory", ""); err == nil {
		t.Fatal("run() succeeded, want an error for an unusable root")
	}
}
