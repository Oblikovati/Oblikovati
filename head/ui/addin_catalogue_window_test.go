//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"
	"testing"

	"oblikovati.org/addincat"
)

func TestCatalogueCardTitle(t *testing.T) {
	base := addincat.Entry{Name: "com.oblikovati.cam", DisplayName: "Oblikovati CAM"}

	available := catalogueCardTitle(addincat.AddInStatus{Entry: base, State: addincat.StateAvailable, LatestVersion: "0.6.0"})
	if !strings.Contains(available, "Oblikovati CAM") || !strings.Contains(available, "0.6.0") {
		t.Errorf("available title = %q", available)
	}

	installed := catalogueCardTitle(addincat.AddInStatus{Entry: base, State: addincat.StateInstalled, InstalledVersion: "0.6.0"})
	if !strings.Contains(installed, "installed 0.6.0") {
		t.Errorf("installed title = %q", installed)
	}

	update := catalogueCardTitle(addincat.AddInStatus{Entry: base, State: addincat.StateUpdateAvailable, InstalledVersion: "0.5.0", LatestVersion: "0.6.0"})
	if !strings.Contains(update, "0.5.0 → 0.6.0") {
		t.Errorf("update title = %q", update)
	}
}

// TestCatalogueCardTitleFallsBackToName uses the add-in id when no display name is set.
func TestCatalogueCardTitleFallsBackToName(t *testing.T) {
	title := catalogueCardTitle(addincat.AddInStatus{
		Entry: addincat.Entry{Name: "com.x.tool"}, State: addincat.StateAvailable, LatestVersion: "1.0.0",
	})
	if !strings.Contains(title, "com.x.tool") {
		t.Errorf("title = %q, want it to fall back to the id", title)
	}
}
