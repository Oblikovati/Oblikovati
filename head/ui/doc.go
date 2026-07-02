// SPDX-License-Identifier: GPL-2.0-only

// Package ui composes the Inventor-style chrome — menu bar, ribbon, model browser,
// and the viewport panel — from the live application Session each frame. There is no
// retained widget tree: every frame reads app.BuildRibbon / app.BuildBrowser and the
// current tool/selection, and Dear ImGui draws that (ADR-0004/0009). All layout lives
// here in Go; the native package only exposes ImGui verbs.
//
// The entry point is DrawChrome: called between Window.BeginFrame and
// Window.EndFrame, it renders one frame of chrome and returns the id of the
// command the user activated (or ""); the caller executes it, so drawing stays
// free of side effects on the model.
//
// The chrome is split by region across sibling files in this package:
//   - chrome.go         — DrawChrome orchestration, keyboard, shared part accessors
//   - chrome_menubar.go — the top menu bar and New Part
//   - chrome_ribbon.go  — the ribbon (tabs, panels, command buttons)
//   - chrome_doctabs.go — the document tab strip and active-document follow
//   - chrome_viewport.go— the Vulkan viewport panel, picking and overlays
//   - chrome_statusbar.go — the status bar
package ui
