// SPDX-License-Identifier: GPL-2.0-only

package app

// ToolMenuOption is one on/off toggle the active tool offers in the viewport right-click menu —
// Inventor's in-command context options (e.g. the Offset tool's Loop Select and Constrain Offset,
// which the user flips mid-command without a dialog).
type ToolMenuOption struct {
	Label   string
	Checked bool
}

// toolMenuOptioned is the optional [Tool] capability that surfaces such toggles. A tool implements it
// to appear with checkable options in the right-click menu; ToggleMenuOption flips the named one.
type toolMenuOptioned interface {
	MenuOptions() []ToolMenuOption
	ToggleMenuOption(label string)
}

// ActiveToolMenuOptions returns the active tool's right-click toggle options, or nil when no tool is
// active or the active tool offers none. The head renders these as checkable rows at the top of the
// viewport context menu.
func (s *Session) ActiveToolMenuOptions() []ToolMenuOption {
	if t, ok := s.activeToolMenuOptioned(); ok {
		return t.MenuOptions()
	}
	return nil
}

// ToggleActiveToolMenuOption flips the active tool's option named label (a no-op when no tool is
// active, the tool offers no options, or the label is unknown).
func (s *Session) ToggleActiveToolMenuOption(label string) {
	if t, ok := s.activeToolMenuOptioned(); ok {
		t.ToggleMenuOption(label)
	}
}

func (s *Session) activeToolMenuOptioned() (toolMenuOptioned, bool) {
	if s.tool == nil {
		return nil, false
	}
	t, ok := s.tool.tool.(toolMenuOptioned)
	return t, ok
}
