//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/renderer"
)

// TestOnTopMarksItems checks the depth-layering helpers flag items for the depth-disabled lane,
// so sketch entities and dimensions draw over the (depth-tested) grid without z-fighting (#909).
func TestOnTopMarksItems(t *testing.T) {
	items := onTop([]renderer.DrawItem{{Primitive: renderer.Lines}, {Primitive: renderer.Lines}})
	for i, it := range items {
		if !it.OnTop {
			t.Errorf("onTop item %d should be marked OnTop", i)
		}
	}
	if !onTopItem(renderer.DrawItem{}).OnTop {
		t.Error("onTopItem should mark the item OnTop")
	}
}
