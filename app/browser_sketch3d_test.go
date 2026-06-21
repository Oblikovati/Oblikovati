// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// TestBrowserShowsSketch3D is the regression for the invisible-import bug: a non-planar DWG
// lands as a Sketch3D, which the model browser used to omit entirely (it only listed 2D
// sketches), so the import had no browser node. A 3D sketch must now appear as a "sketch3d"
// node carrying its name and a Sketch3DHandle.
func TestBrowserShowsSketch3D(t *testing.T) {
	s := NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "sketch3d-browser.opd", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches3D().Add()
	sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 1, 1))
	def.Recompute()

	root := BuildBrowser(s)
	var node *BrowserNode
	for i := range root.Children {
		if root.Children[i].Kind == "sketch3d" {
			node = &root.Children[i]
			break
		}
	}
	if node == nil {
		t.Fatal("browser has no sketch3d node for the imported 3D sketch")
	}
	if node.Label != sk.Name() {
		t.Errorf("sketch3d node label = %q, want %q", node.Label, sk.Name())
	}
	if _, ok := node.Select.(Sketch3DHandle); !ok {
		t.Errorf("sketch3d node selectable = %T, want Sketch3DHandle", node.Select)
	}
}
