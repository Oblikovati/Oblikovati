//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Web views (M05-F08): each visible web dialog renders as a window — docked when
// its spec asks, floating otherwise. The head has no embedded web engine yet (the
// engine sits behind the app's URLOpener seam), so the window shows the URL with an
// open-in-browser affordance; when an engine lands, this file is the only render
// site to change.

// drawWebViews renders the visible web dialogs.
func drawWebViews(s *app.Session) {
	for _, view := range s.WebViews() {
		if !view.Visible {
			continue
		}
		applyInitialDock(view.Dock)
		dialogSizeOnce(420, 180)
		shown, open := native.BeginClosable(view.Title + "###webview-" + view.ID)
		if shown {
			native.Text(view.URL)
			if native.Button("Open in Browser##" + view.ID) {
				if err := s.OpenURL(view.URL); err != nil {
					fmt.Fprintf(os.Stderr, "web view %q: %v\n", view.ID, err)
				}
			}
		}
		native.End()
		if !open {
			_ = s.CloseWebDialog(view.ID)
		}
	}
}
