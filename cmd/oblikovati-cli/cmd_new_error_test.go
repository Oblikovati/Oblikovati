// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"io"
	"testing"

	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// TestCreateDocumentWrapsAddFailure covers the "new: %w" wraps: re-adding a document at a path
// already taken fails, and createDocument wraps that error for both the part and non-part paths.
func TestCreateDocumentWrapsAddFailure(t *testing.T) {
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	part, _ := doc.ParseDocumentType("part")
	asm, _ := doc.ParseDocumentType("assembly")

	if _, err := createDocument(ws, part, "dup.opd", false, io.Discard); err != nil {
		t.Fatalf("first part add: %v", err)
	}
	if _, err := createDocument(ws, part, "dup.opd", false, io.Discard); err == nil {
		t.Error("re-adding a part at a taken path should error (part path)")
	}
	if _, err := createDocument(ws, asm, "dup.opd", false, io.Discard); err == nil {
		t.Error("adding an assembly at a taken path should error (non-part path)")
	}
}
