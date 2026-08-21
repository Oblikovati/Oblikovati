// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestStatusPromptTextFollowsTheActiveCommand pins the status bar's left-hand line to
// Inventor's contract: while a command runs it names the command and what it needs next;
// otherwise it falls back to add-in status text, then to the idle help line.
func TestStatusPromptTextFollowsTheActiveCommand(t *testing.T) {
	cases := []struct {
		name string
		sb   app.StatusBar
		want string
	}{
		{"idle", app.StatusBar{Prompt: "Ready"}, statusIdlePrompt},
		{"idle with add-in status", app.StatusBar{Prompt: "Ready", StatusText: "Importing…"}, "Importing…"},
		{
			"running tool",
			app.StatusBar{ToolActive: true, ToolName: "Extrude", Prompt: "Select a profile"},
			"Extrude: Select a profile",
		},
		{"running tool, no prompt", app.StatusBar{ToolActive: true, ToolName: "Extrude"}, "Extrude"},
		{"running tool, no name", app.StatusBar{ToolActive: true, Prompt: "Select a profile"}, "Select a profile"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := statusPromptText(c.sb); got != c.want {
				t.Errorf("statusPromptText = %q, want %q", got, c.want)
			}
		})
	}
}

// TestStatusRightTextSummarisesEnvironmentAndSelection pins the right-hand summary: empty
// when idle with nothing selected, the sketch badge while editing a sketch, and the
// selection count when anything is picked.
func TestStatusRightTextSummarisesEnvironmentAndSelection(t *testing.T) {
	cases := []struct {
		name string
		sb   app.StatusBar
		want string
	}{
		{"idle", app.StatusBar{}, ""},
		{"selection only", app.StatusBar{SelectionCount: 3}, "3 selected"},
		{"sketch only", app.StatusBar{InSketch: true}, "Sketch"},
		{"both", app.StatusBar{InSketch: true, SelectionCount: 1}, "Sketch" + statusSeparator + "1 selected"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := statusRightText(c.sb); got != c.want {
				t.Errorf("statusRightText = %q, want %q", got, c.want)
			}
		})
	}
}
