// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"

	"oblikovati.org/app"
)

// Pure presentation helpers for the status bar (Inventor-parity 2026-08-17), split out
// with no cgo build tag so they are unit-tested without the native layer (mirrors
// command_window_text.go). The band itself is drawn in chrome_statusbar.go.

// statusIdlePrompt is what the bar reads with nothing running — Inventor's idle status
// line (C-inventor-ui-reference §7: "with no command active it shows 'Press F1 for more
// help' at far bottom-left").
const statusIdlePrompt = "Press F1 for more help"

// statusSeparator joins the right-hand summary's parts.
const statusSeparator = "   ·   "

// statusPromptText is the bar's left-hand line: the running tool's name and its per-step
// prompt (Inventor's status bar "indicates the next action the active command requires"),
// else any add-in status text, else the idle line.
func statusPromptText(sb app.StatusBar) string {
	if sb.ToolActive {
		switch {
		case sb.ToolName != "" && sb.Prompt != "":
			return sb.ToolName + ": " + sb.Prompt
		case sb.Prompt != "":
			return sb.Prompt
		case sb.ToolName != "":
			return sb.ToolName
		}
	}
	if sb.StatusText != "" {
		return sb.StatusText
	}
	return statusIdlePrompt
}

// statusRightText is the bar's right-hand summary: the current environment badge while a
// sketch is being edited (Inventor swaps in a sketch status bar there) and the selection
// count when anything is selected. "" when neither applies, so an idle bar is just the
// prompt.
func statusRightText(sb app.StatusBar) string {
	var parts []string
	if sb.InSketch {
		parts = append(parts, "Sketch")
	}
	if sb.SelectionCount > 0 {
		parts = append(parts, selectionText(sb.SelectionCount))
	}
	return strings.Join(parts, statusSeparator)
}
