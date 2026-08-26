//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// TestIconCacheEvictsOnScaleChange is the #1237 regression. Icon textures are keyed by pixel
// size, so a UI icon-scale change must retire the previous size's textures. Without that, a
// slider drag leaked one texture per intermediate size per icon and exhausted the 256-set Vulkan
// descriptor pool — icons vanished, and the font atlas could no longer allocate (a GetTexID
// assertion crash). The fix retires the whole set when the scale changes, bounding the cache.
func TestIconCacheEvictsOnScaleChange(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	c := newIconCache(win)

	// Populate a set of real textures at scale 1.0.
	c.beginFrame(1, 1.0)
	for _, name := range []string{"extrude", "revolve", "sweep", "loft", "hole"} {
		if tex, ok := c.texture(name, "", 40); !ok || tex == 0 {
			t.Fatalf("texture(%q) failed to upload", name)
		}
	}
	live := len(c.tex)
	if live == 0 {
		t.Fatal("no textures cached at scale 1.0")
	}

	// A scale change must clear the live set, retiring the old textures rather than orphaning them.
	c.beginFrame(1, 2.0)
	if len(c.tex) != 0 {
		t.Errorf("icon-scale change left %d live textures, want 0 (all retired)", len(c.tex))
	}
	if len(c.retired) < live {
		t.Errorf("icon-scale change retired %d textures, want >= %d", len(c.retired), live)
	}

	// After the in-flight window passes, the retired textures are freed — the cache (and the
	// descriptor pool behind it) does not grow without bound across repeated scale changes.
	for range retireAfterFrames + 2 {
		c.beginFrame(1, 2.0)
	}
	if len(c.retired) != 0 {
		t.Errorf("retired textures not freed after %d frames: %d still pending", retireAfterFrames+2, len(c.retired))
	}
}
