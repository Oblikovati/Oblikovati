//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// dockablePanel describes one built-in dockable window so every panel opens through ONE shared
// path (Oblikovati#1473): drawDockablePanel owns the BeginClosable/End shell — giving every panel
// a uniform close 'X' — and drawViewMenu lists each registered panel as a show/hide toggle, so a
// window opened from the View menu can always be closed again (the bug: Materials/Preferences had
// no close button and no way back). draw renders only the window BODY; the registry owns the
// chrome.
//
//   - title is BOTH the window caption and its Dear ImGui id, so it must stay stable (the default
//     dock layout in layoutDockedPanels references "Model"/"Command" by exactly this name).
//   - menuLabel is the View-menu text when it must differ from the caption (the docked REPL's
//     window is captioned "Command" but reads as "Command Window" in the menu).
//   - isOpen/setOpen wrap whichever store backs the panel — a head-local bool (Materials,
//     Preferences, the Model browser) or a session predicate (Parameters, Lighting, …) — so this
//     refactor does not have to relocate that state.
//   - after runs after End for the rare panel that spawns trailing child windows (the Parameters
//     value-list / tolerance / group editors are separate ImGui windows, not part of its body).
type dockablePanel struct {
	title     string
	menuLabel string
	width     float32
	height    float32
	isOpen    func(*app.Session) bool
	setOpen   func(*app.Session, bool)
	draw      func(*app.Session)
	after     func(*app.Session)
}

// dockablePanels is the ordered registry of built-in dockable windows. Order is the View-menu
// order. Registered once at init by registerDockablePanel from each panel's file.
var dockablePanels []dockablePanel

// registerDockablePanel adds a built-in dockable window to the registry. Call it from a panel
// file's init so the panel is self-describing and the View menu / render loop pick it up with no
// central edit (the self-registration pattern). Panics on a duplicate title since the title is the
// window id and must be unique.
func registerDockablePanel(p dockablePanel) {
	for _, e := range dockablePanels {
		if e.title == p.title {
			panic("duplicate dockable panel title: " + p.title)
		}
	}
	dockablePanels = append(dockablePanels, p)
}

// drawDockablePanels renders every registered built-in dockable window through the shared path,
// so they all share one creation/close-X behaviour instead of bespoke per-panel Begin calls.
func drawDockablePanels(s *app.Session) {
	for i := range dockablePanels {
		drawDockablePanel(s, &dockablePanels[i])
	}
}

// drawDockablePanel renders one panel when open: a default size on first show, the closable window
// (the 'X' routes back through setOpen so the panel's own visibility store is updated), the body,
// and any trailing child windows.
func drawDockablePanel(s *app.Session, p *dockablePanel) {
	if !p.isOpen(s) {
		return
	}
	if p.width > 0 {
		native.SetNextWindowSizeOnce(p.width, p.height)
	}
	visible, open := native.BeginClosable(p.title)
	if visible {
		p.draw(s)
	}
	native.End()
	if p.after != nil {
		p.after(s)
	}
	if !open {
		p.setOpen(s, false)
	}
}

// drawViewMenu is the main-menu "View" menu (Oblikovati#1473): one show/hide toggle per registered
// dockable window. It replaces the scattered per-panel entries that used to live under Tools/Edit,
// so every panel is reachable — and re-openable after closing — from one predictable place.
func drawViewMenu(s *app.Session) {
	if !native.BeginMenu("View") {
		return
	}
	for i := range dockablePanels {
		p := &dockablePanels[i]
		if native.MenuItem(checkLabel(panelMenuLabel(p), p.isOpen(s))) {
			p.setOpen(s, !p.isOpen(s))
		}
	}
	native.EndMenu()
}

// panelMenuLabel is the View-menu text for a panel: its menuLabel override, else its caption.
func panelMenuLabel(p *dockablePanel) string {
	if p.menuLabel != "" {
		return p.menuLabel
	}
	return p.title
}

// init registers the built-in dockable windows once (so the duplicate-title guard runs). The set
// itself is the data-driven builtinDockablePanels table below.
func init() {
	for _, p := range builtinDockablePanels {
		registerDockablePanel(p)
	}
}

// builtinDockablePanels is the ordered table of built-in dockable windows, in View-menu order.
// Centralised (rather than one registration per file) so the menu order is deterministic and
// reviewable in one place; each entry still only points at its panel file's body function.
// "Model"/"Command" keep their exact captions because the default dock layout (layoutDockedPanels)
// docks windows by that name.
var builtinDockablePanels = []dockablePanel{
	{title: "Model", menuLabel: "Model Browser",
		isOpen: headBoolGet(&showBrowser), setOpen: headBoolSet(&showBrowser), draw: drawBrowserBody},
	{title: "Parameters", width: 820, height: 520,
		isOpen:  (*app.Session).ParametersOpen,
		setOpen: sessionSetter((*app.Session).OpenParameters, (*app.Session).CloseParameters),
		draw:    drawParametersBody, after: drawParametersEditors},
	{title: "Materials", width: 420, height: 560,
		isOpen: headBoolGet(&showMaterials), setOpen: headBoolSet(&showMaterials), draw: drawMaterialsBody},
	{title: "Lighting", width: 360, height: 440,
		isOpen:  (*app.Session).LightingPanelOpen,
		setOpen: sessionSetter((*app.Session).OpenLightingPanel, (*app.Session).CloseLightingPanel), draw: drawLightingBody},
	{title: "Color Styles", width: 300, height: 340,
		isOpen:  (*app.Session).ColorStylesPanelOpen,
		setOpen: sessionSetter((*app.Session).OpenColorStylesPanel, (*app.Session).CloseColorStylesPanel), draw: drawColorStylesBody},
	{title: "Display Settings", width: 320, height: 320,
		isOpen:  (*app.Session).DisplaySettingsOpen,
		setOpen: sessionSetter((*app.Session).OpenDisplaySettings, (*app.Session).CloseDisplaySettings), draw: drawDisplaySettingsBody},
	{title: "Document Settings — Units", width: 340, height: 360,
		isOpen:  (*app.Session).UnitsSettingsOpen,
		setOpen: sessionSetter((*app.Session).OpenUnitsSettings, (*app.Session).CloseUnitsSettings), draw: drawUnitsSettingsBody},
	{title: "Named Views", width: 300, height: 320,
		isOpen:  (*app.Session).NamedViewsPanelOpen,
		setOpen: sessionSetter((*app.Session).OpenNamedViewsPanel, (*app.Session).CloseNamedViewsPanel), draw: drawNamedViewsBody},
	{title: "Bill of Materials", width: 720, height: 480,
		isOpen:  (*app.Session).BOMPanelOpen,
		setOpen: sessionSetter((*app.Session).OpenBOM, (*app.Session).CloseBOM), draw: drawBOMBody},
	{title: "History Browser", width: 560, height: 440,
		isOpen:  (*app.Session).HistoryBrowserOpen,
		setOpen: sessionSetter((*app.Session).OpenHistoryBrowser, (*app.Session).CloseHistoryBrowser), draw: drawHistoryBrowserBody},
	{title: "Selection Filter", width: 340, height: 400,
		isOpen:  (*app.Session).SelectionFilterWindowOpen,
		setOpen: sessionSetter((*app.Session).OpenSelectionFilterWindow, (*app.Session).CloseSelectionFilterWindow), draw: drawSelectionFilterBody},
	{title: "Command", menuLabel: "Command Window",
		isOpen: (*app.Session).CommandWindowOpen, setOpen: (*app.Session).SetCommandWindowOpen, draw: drawCommandWindowBody},
	{title: "Preferences", width: 640, height: 480,
		isOpen: headBoolGet(&showPreferences), setOpen: headBoolSet(&showPreferences), draw: drawPreferencesBody},
}

// sessionSetter turns a panel's Open/Close session verbs into the registry's setOpen(v) form.
func sessionSetter(open, close func(*app.Session)) func(*app.Session, bool) {
	return func(s *app.Session, v bool) {
		if v {
			open(s)
		} else {
			close(s)
		}
	}
}

// headBoolGet/headBoolSet adapt a head-local visibility bool (showMaterials, showPreferences,
// showBrowser — UI state that lives in the head, not the session) to the panel's isOpen/setOpen.
func headBoolGet(p *bool) func(*app.Session) bool  { return func(*app.Session) bool { return *p } }
func headBoolSet(p *bool) func(*app.Session, bool) { return func(_ *app.Session, v bool) { *p = v } }
