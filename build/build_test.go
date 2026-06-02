// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"strings"
	"testing"
)

func TestNotYetImplementedCarriesIssueID(t *testing.T) {
	err := NotYetImplemented("PBI-042")
	if err == nil {
		t.Fatal("NotYetImplemented returned nil; want an error")
	}
	if !strings.Contains(err.Error(), "PBI-042") {
		t.Fatalf("error %q does not mention the issue id PBI-042", err.Error())
	}
}

func TestVersionMetadataHasDefaults(t *testing.T) {
	for name, got := range map[string]string{"Version": Version, "Commit": Commit, "Date": Date} {
		if got == "" {
			t.Errorf("%s is empty; link-time vars must keep a non-empty default", name)
		}
	}
}

// Referencing the mode flags keeps them part of the compiled API surface and
// documents that they are build-time constants, not runtime configuration.
func TestModeFlagsAreCompileTimeConstants(t *testing.T) {
	const _ = Debug && Profile && Editor || true
}
