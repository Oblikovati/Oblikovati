//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"testing"

	"oblikovati.org/head/internal/native"
)

func TestRenderWindowIconsCoversAllSizes(t *testing.T) {
	imgs, err := renderWindowIcons()
	if err != nil {
		t.Fatalf("renderWindowIcons: %v", err)
	}
	if len(imgs) != len(iconSizes) {
		t.Fatalf("rendered %d icons, want %d", len(imgs), len(iconSizes))
	}
	for i, im := range imgs {
		if got := im.Bounds().Dx(); got != iconSizes[i] {
			t.Fatalf("icon %d is %dpx, want %d", i, got, iconSizes[i])
		}
	}
}

// TestRenderWindowIconsAndApplyHandleRenderError drives the error paths without a window:
// a 0px candidate makes appicon.Image fail, so renderWindowIcons returns the error and
// applyWindowIcon takes its early return (and never dereferences the nil window).
func TestRenderWindowIconsAndApplyHandleRenderError(t *testing.T) {
	saved := iconSizes
	iconSizes = []int{0}
	defer func() { iconSizes = saved }()

	if _, err := renderWindowIcons(); err == nil {
		t.Fatal("renderWindowIcons should error on a 0px candidate size")
	}
	applyWindowIcon(nil) // error path returns before touching the window
}

// TestApplyWindowIcon drives the full window-icon path against a real window (skipped
// when no display/Vulkan is available).
func TestApplyWindowIcon(t *testing.T) {
	win, err := native.CreateWindow(320, 240, "obk-appicon-test")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	applyWindowIcon(win)
}
