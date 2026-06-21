//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/addincat"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Add-In Catalogue window (Tools ▸ Get Add-Ins…, #1164). A thin view over the session's
// catalogue state: it shows three tabs (available / installed / updatable), a search box that
// re-queries the service, and per-add-in Install/Update/Uninstall actions. All network and
// install work runs on a session goroutine; this view only reads the cached snapshot and
// issues commands, so it never blocks the frame loop.
var addInCatalogueUI struct {
	open   bool
	search [256]byte
}

// OpenAddInCatalogue shows the window and kicks an initial refresh.
func OpenAddInCatalogue(s *app.Session) {
	addInCatalogueUI.open = true
	s.RefreshAddInCatalogue("")
}

// drawAddInCatalogueWindow renders the catalogue window when open.
func drawAddInCatalogueWindow(s *app.Session) {
	if !addInCatalogueUI.open {
		return
	}
	native.SetNextWindowSizeOnce(640, 480)
	if native.Begin("Add-In Catalogue") {
		drawCatalogueHeader(s)
		drawCatalogueTabs(s)
	}
	native.End()
}

// drawCatalogueHeader draws the search box, a refresh button, and any busy/error/notice line.
func drawCatalogueHeader(s *app.Session) {
	if native.InputText("Search##addincat", addInCatalogueUI.search[:]) {
		s.RefreshAddInCatalogue(bufString(addInCatalogueUI.search[:]))
	}
	native.SameLine()
	if native.Button("Refresh") {
		s.RefreshAddInCatalogue(bufString(addInCatalogueUI.search[:]))
	}
	if s.AddInCatalogueBusy() {
		native.Text("Working…")
	}
	if e := s.AddInCatalogueError(); e != "" {
		native.Text("Error: " + e)
	}
	if n := s.AddInCatalogueNotice(); n != "" {
		native.Text(n)
	}
	native.Separator()
}

// drawCatalogueTabs lays out the three state tabs over the current snapshot.
func drawCatalogueTabs(s *app.Session) {
	if !native.BeginTabBar("##addincat-tabs") {
		return
	}
	statuses := s.AddInStatuses()
	drawCatalogueTab(s, "Available", statuses, addincat.StateAvailable)
	drawCatalogueTab(s, "Installed", statuses, addincat.StateInstalled)
	drawCatalogueTab(s, "Updates", statuses, addincat.StateUpdateAvailable)
	native.EndTabBar()
}

// drawCatalogueTab renders the cards whose state matches, in a scrollable child region.
func drawCatalogueTab(s *app.Session, label string, statuses []addincat.AddInStatus, state addincat.State) {
	if !native.BeginTabItem(label) {
		return
	}
	if native.BeginChild("##child-"+label, 0, 0, false) {
		any := false
		for _, st := range statuses {
			if st.State != state {
				continue
			}
			any = true
			drawCatalogueCard(s, st)
		}
		if !any {
			native.Text("Nothing here.")
		}
	}
	native.EndChild()
	native.EndTabItem()
}

// drawCatalogueCard renders one add-in's metadata and its action buttons.
func drawCatalogueCard(s *app.Session, st addincat.AddInStatus) {
	native.PushID(st.Entry.Name)
	native.Text(catalogueCardTitle(st))
	if st.Entry.Description != "" {
		native.Text(st.Entry.Description)
	}
	if st.Entry.License != "" {
		native.Text("License: " + st.Entry.License)
	}
	drawCatalogueCardActions(s, st)
	native.Separator()
	native.PopID()
}

// catalogueCardTitle is the card's heading: the display name plus a version hint that reflects
// the add-in's state (installed version, an update arrow, or the offered version).
func catalogueCardTitle(st addincat.AddInStatus) string {
	name := st.Entry.DisplayName
	if name == "" {
		name = st.Entry.Name
	}
	switch st.State {
	case addincat.StateInstalled:
		return fmt.Sprintf("%s (installed %s)", name, st.InstalledVersion)
	case addincat.StateUpdateAvailable:
		return fmt.Sprintf("%s (%s → %s)", name, st.InstalledVersion, st.LatestVersion)
	default:
		return fmt.Sprintf("%s %s", name, st.LatestVersion)
	}
}

// drawCatalogueCardActions draws the per-state action buttons, disabled while an operation
// is in flight so a second click cannot pile on.
func drawCatalogueCardActions(s *app.Session, st addincat.AddInStatus) {
	native.BeginDisabled(s.AddInCatalogueBusy())
	switch st.State {
	case addincat.StateAvailable:
		if native.Button("Install") {
			s.InstallAddIn(st.Entry.Name)
		}
	case addincat.StateUpdateAvailable:
		if native.Button("Update") {
			s.InstallAddIn(st.Entry.Name)
		}
		native.SameLine()
		if native.Button("Uninstall") {
			s.UninstallAddIn(st.Entry.Name)
		}
	case addincat.StateInstalled:
		if native.Button("Uninstall") {
			s.UninstallAddIn(st.Entry.Name)
		}
	}
	native.EndDisabled()
}
