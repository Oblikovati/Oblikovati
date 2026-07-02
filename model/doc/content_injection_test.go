// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

// fakeInjectedContent is a named fake content proving factory injection (#1617):
// it can only reach a document through the ContentFactories the workspace was
// constructed with — there is no global registry left to fall back to.
type fakeInjectedContent struct{ marker string }

func (*fakeInjectedContent) DocumentType() DocumentType { return Part }

// TestWorkspaceBuildsContentFromInjectedFactories pins audit B6's seam: a
// workspace constructed with a minimal one-entry factory set builds part
// content through it, and a kind WITHOUT a factory falls back to the
// identity-only stub — injection decides, not import linkage.
func TestWorkspaceBuildsContentFromInjectedFactories(t *testing.T) {
	ws := NewWorkspace(nil, ContentFactories{
		Part: func() Content { return &fakeInjectedContent{marker: "injected"} },
	})
	d, err := ws.Add(Part, "injected.opd", false)
	if err != nil {
		t.Fatalf("Add(Part): %v", err)
	}
	got, ok := d.Content().(*fakeInjectedContent)
	if !ok || got.marker != "injected" {
		t.Fatalf("part content = %T, want the injected fake", d.Content())
	}
	drawing, err := ws.Add(Drawing, "stub.odd", false)
	if err != nil {
		t.Fatalf("Add(Drawing): %v", err)
	}
	if _, stub := drawing.Content().(*DrawingContent); !stub {
		t.Fatalf("drawing content = %T, want the identity-only stub (no factory injected)", drawing.Content())
	}
}
