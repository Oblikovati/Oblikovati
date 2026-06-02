// SPDX-License-Identifier: GPL-2.0-only

package app

// The status bar is a MODEL built from live session state each frame (like the ribbon
// and browser) — Dear ImGui renders it (core/09). It mirrors Inventor's status bar: a
// prompt that guides the user through the active command, plus the current selection
// count. The logic here is pure and testable.

// Prompted is an optional Tool capability: a tool that guides the user with a per-step
// status-bar prompt implements it (Inventor's status-bar prompts). Tools without it get
// a generic prompt derived from their commit-readiness.
type Prompted interface {
	Prompt(s *Session) string
}

// StatusBar is the status-bar model for a frame.
type StatusBar struct {
	Prompt         string // guidance text shown to the user
	ToolName       string // active tool's name, or "" when idle
	ToolActive     bool   // whether a tool is running (so OK/Cancel show)
	CanCommit      bool   // whether OK should be enabled
	SelectionCount int    // size of the current selection set
}

// BuildStatus assembles the status bar from the session: the active tool's prompt and
// commit-readiness, or "Ready" when idle, plus the selection count.
func BuildStatus(s *Session) StatusBar {
	sb := StatusBar{Prompt: "Ready", SelectionCount: s.Selection().Count()}
	ti := s.ActiveTool()
	if ti == nil {
		return sb
	}
	sb.ToolActive = true
	sb.ToolName = ti.Name()
	sb.CanCommit = ti.Tool().CanCommit()
	sb.Prompt = toolPrompt(s, ti.Tool())
	return sb
}

// toolPrompt returns the tool's own per-step prompt if it provides one, else a generic
// prompt based on whether it can commit yet.
func toolPrompt(s *Session, t Tool) string {
	if p, ok := t.(Prompted); ok {
		return p.Prompt(s)
	}
	if t.CanCommit() {
		return "Click OK to finish, or Cancel (Esc)"
	}
	return "Select or specify input for " + t.Name()
}
