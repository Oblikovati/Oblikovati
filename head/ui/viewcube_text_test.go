// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati/model/doc"
)

func TestStrokeFontCoversFaceWords(t *testing.T) {
	for _, w := range []string{"TOP", "BOTTOM", "FRONT", "BACK", "LEFT", "RIGHT"} {
		for _, r := range w {
			if len(glyphStrokes(r)) == 0 {
				t.Errorf("stroke font missing %q (from %q)", r, w)
			}
		}
	}
}

func TestFaceLabelSegmentsLieInPlane(t *testing.T) {
	// The TOP face viewed head-on (top-down): its label must produce strokes, and a
	// face-on view keeps them in the screen plane (non-empty, finite).
	faces := visibleFaces(topDownCamera(), doc.IdentityCubeOrient(), 40)
	var top *cubeFace
	for i := range faces {
		if faces[i].region.Label == "TOP" {
			top = &faces[i]
		}
	}
	if top == nil {
		t.Fatal("TOP face not visible top-down")
	}
	if segs := faceLabelSegments(*top, topDownCamera(), doc.IdentityCubeOrient(), 40); len(segs) == 0 {
		t.Error("TOP label produced no strokes")
	}
}
