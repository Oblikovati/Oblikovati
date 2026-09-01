// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"
	"testing"

	"oblikovati.org/math"
)

// The mixed per-face dispatch (ADR-0058) declines rather than guesses: a configuration it does not
// model is refused at classification, before any geometry is built, and the caller falls to the
// curved/CSG paths. A boundaryless face — a whole sphere, its own closed shell — carries no boundary
// point to classify the face as a whole, which is exactly such a configuration.

// TestBooleanMixedDeclinesBoundarylessPassFace: a sphere disjoint from a block clears the interaction
// gate (the boxes never meet), so the dispatch reaches the pass-face classification and declines there.
// The decline reaches Boolean's caller as the named sentinel — ops.Boolean routes on it to the
// curved/CSG paths — never as a panic or a wrong body.
func TestBooleanMixedDeclinesBoundarylessPassFace(t *testing.T) {
	t.Parallel()
	block, err := SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	ball, err := SolidSphere(math.P3(30, 30, 30), 2, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	if _, _, err := booleanMixed(Union, block, ball); !errors.Is(err, ErrUnsupportedMixedBoolean) {
		t.Fatalf("booleanMixed with a boundaryless pass face = %v, want ErrUnsupportedMixedBoolean", err)
	}
	if _, err := Boolean(Union, block, ball); !errors.Is(err, ErrUnsupportedMixedBoolean) {
		t.Errorf("Boolean = %v, want the decline classified as ErrUnsupportedMixedBoolean", err)
	}
}
