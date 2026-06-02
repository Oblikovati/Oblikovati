// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

// Each specialization must expose a content object of the right kind, and its
// DocumentType must agree with the content's — type discrimination is correct
// end to end (PBI-034).
func TestSpecializationsExposeMatchingContent(t *testing.T) {
	part := NewPartDocument("p.obk")
	if c := part.ComponentDefinition(); c == nil || c.DocumentType() != Part {
		t.Errorf("PartDocument content = %v, want Part content", c)
	}
	if part.DocumentType() != Part {
		t.Errorf("PartDocument.DocumentType() = %v, want Part", part.DocumentType())
	}

	asm := NewAssemblyDocument("a.obk")
	if c := asm.ComponentDefinition(); c == nil || c.DocumentType() != Assembly {
		t.Errorf("AssemblyDocument content = %v, want Assembly content", c)
	}

	dwg := NewDrawingDocument("d.obk")
	if c := dwg.DrawingContent(); c == nil || c.DocumentType() != Drawing {
		t.Errorf("DrawingDocument content = %v, want Drawing content", c)
	}

	prn := NewPresentationDocument("n.obk")
	if c := prn.PresentationContent(); c == nil || c.DocumentType() != Presentation {
		t.Errorf("PresentationDocument content = %v, want Presentation content", c)
	}
}

// The generic base accessor returns the same content the typed accessor does, so
// the workspace can treat documents uniformly while callers keep a typed view.
func TestBaseContentMatchesTypedAccessor(t *testing.T) {
	part := NewPartDocument("p.obk")
	if Content(part.ComponentDefinition()) != part.Content() {
		t.Error("base Content() and ComponentDefinition() disagree")
	}
	if part.Content().DocumentType() != part.DocumentType() {
		t.Error("content kind disagrees with document kind")
	}
}
