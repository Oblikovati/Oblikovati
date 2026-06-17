//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// TestDocumentTabLabelDisambiguatesCollidingNames is the regression for the part↔drawing
// active-document ping-pong: a part "box.opd" and a drawing "box.odd" both display as "box",
// and ImGui keys tabs by their label, so identical labels collided to one tab id and the head
// flipped the active document every frame. The tab label must carry a unique per-document id
// suffix while still showing the bare display name.
func TestDocumentTabLabelDisambiguatesCollidingNames(t *testing.T) {
	s := app.NewSession()
	part, err := s.Workspace().Add(doc.Part, "box.opd", true)
	if err != nil {
		t.Fatalf("add part: %v", err)
	}
	drawing, err := s.Workspace().Add(doc.Drawing, "box.odd", true)
	if err != nil {
		t.Fatalf("add drawing: %v", err)
	}

	if part.DisplayName() != drawing.DisplayName() {
		t.Fatalf("precondition: want colliding display names, got %q vs %q",
			part.DisplayName(), drawing.DisplayName())
	}

	partLabel, drawingLabel := documentTabLabel(part), documentTabLabel(drawing)
	if partLabel == drawingLabel {
		t.Fatalf("tab labels collide (%q) — ImGui would treat both tabs as one and flip the active document",
			partLabel)
	}
	// The visible text (before "###") must still be the bare display name.
	for _, lbl := range []string{partLabel, drawingLabel} {
		if visible, _, found := strings.Cut(lbl, "###"); !found || visible != "box" {
			t.Errorf("label %q: want visible text %q before a \"###\" id suffix", lbl, "box")
		}
	}
}
