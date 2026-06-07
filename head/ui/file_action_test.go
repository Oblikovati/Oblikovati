//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"
	"testing"

	"oblikovati/app"
)

// TestApplyFileActionSurfacesImportError guards that a failed import reports the underlying
// (kernel) error in the status bar (s.Notice), not just to stderr — so an import that does
// nothing tells the user why, instead of failing silently.
func TestApplyFileActionSurfacesImportError(t *testing.T) {
	s := app.NewSession()
	applyFileAction(s, fileAction{Kind: dialogImport, Path: "/no/such/file.step"})
	if n := s.Notice(); !strings.Contains(n, "Import failed") {
		t.Errorf("notice = %q, want it to report the import failure", n)
	}
}
