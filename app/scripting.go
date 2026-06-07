// SPDX-License-Identifier: GPL-2.0-only

package app

// scripting.go owns the session-side state of the Script Console panel and its ribbon
// command. The runtime (the Lua engine + dispatched caller) lives in the head, which
// reads ScriptConsoleOpen() each frame to decide whether to render the panel — the same
// open-flag pattern the Lighting/Parameters panels use. See ADR-0028 / lua-scripting-plan.

// OpenScriptConsole / CloseScriptConsole / ScriptConsoleOpen drive the Script Console panel.
func (s *Session) OpenScriptConsole()      { s.scriptConsoleOpen = true }
func (s *Session) CloseScriptConsole()     { s.scriptConsoleOpen = false }
func (s *Session) ScriptConsoleOpen() bool { return s.scriptConsoleOpen }

// ToggleScriptConsole flips the panel open/closed (the ribbon button's action).
func (s *Session) ToggleScriptConsole() { s.scriptConsoleOpen = !s.scriptConsoleOpen }

// scriptConsoleCommand is the Manage tab's "Script Console" button (our equivalent of
// Inventor's Manage ▸ iLogic): it toggles the console panel that runs a sandboxed Lua
// script against the live model through the public wire API (ADR-0028). Always enabled —
// a script can create a document, so it is useful even with nothing open.
func scriptConsoleCommand() *CommandDefinition {
	return NewCommand("Manage.ScriptConsole", "Script Console", "Scripts", func(s *Session) error {
		s.ToggleScriptConsole()
		return nil
	}).WithTab("Manage").
		WithIcon("script-console").WithButtonStyle(LargeIconButton).
		WithActive(func(s *Session) bool { return s.ScriptConsoleOpen() }).
		WithTooltip("Script Console — run a sandboxed Lua script that drives the model through the public API.")
}
