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
