// SPDX-License-Identifier: GPL-2.0-only

package app

// Literals shared across the app package — error formats, UI labels/prompts, ribbon
// tab names, transaction labels, and log scopes — defined once so the same wording is
// not duplicated across the many tool and command files.
const (
	errNoAddIn            = "app: no add-in %q"
	errNoMiniToolbar      = "app: no mini-toolbar %q"
	errNoProgressBar      = "app: no progress bar %d"
	errNoBaseView         = "drawing: no base view to dimension — add a base view first"
	errBaseViewNoGeometry = "drawing: base view %q has no geometry to dimension"

	labelBaseView  = "Base View"
	labelWorkPlane = "Work Plane"
	labelWorkPoint = "Work Point"
	tabGetStarted  = "Get Started"

	promptSelectTwoLines        = "Select two lines"
	promptSelectLineOrTwoPoints = "Select a line or two points"
	needTwoLines                = "two lines"

	suffixSuppressed = " (suppressed)"
	logMessageCenter = "message-center"
)
