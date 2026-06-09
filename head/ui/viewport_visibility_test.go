// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

func TestShouldDrawViewportOnlyWithDocuments(t *testing.T) {
	s := app.NewSession()
	if shouldDrawViewport(s) {
		t.Fatal("shouldDrawViewport(empty session) = true, want false")
	}
	if _, err := compdef.AddPart(s.Workspace(), "p.obk", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if !shouldDrawViewport(s) {
		t.Fatal("shouldDrawViewport(session with document) = false, want true")
	}
}
