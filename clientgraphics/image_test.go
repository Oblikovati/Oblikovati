// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestBuildImageBecomesBillboard checks an image primitive (M16-F05 #641) is extracted as an
// ImageBillboard (with its anchor, path and size), not draw-list geometry.
func TestBuildImageBecomesBillboard(t *testing.T) {
	s := NewStore()
	s.Set(mustDecode(t, wire.SetClientGraphicsArgs{ClientId: "img", Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
		Kind: string(types.GraphicsImage), ImagePath: "/tmp/logo.png", Anchor: []float64{1, 2, 3},
		ImageWidth: 4, ImageHeight: 2,
	}}}}}))
	items, _, images := s.Build(testCamera())
	if len(items) != 0 {
		t.Errorf("an image should produce no draw-list geometry, got %d items", len(items))
	}
	if len(images) != 1 {
		t.Fatalf("want one image billboard, got %d", len(images))
	}
	im := images[0]
	if im.Path != "/tmp/logo.png" || im.Width != 4 || im.Height != 2 {
		t.Errorf("billboard = %+v, want /tmp/logo.png 4x2", im)
	}
	if im.Anchor.X != 1 || im.Anchor.Y != 2 || im.Anchor.Z != 3 {
		t.Errorf("billboard anchor = %v, want (1,2,3)", im.Anchor)
	}
}
