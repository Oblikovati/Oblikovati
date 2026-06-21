// SPDX-License-Identifier: GPL-2.0-only

package app

// selectionPrioritySpec pairs a priority with its ribbon label and tooltip.
type selectionPrioritySpec struct {
	priority       SelectionPriority
	id, label, tip string
}

var selectionPrioritySpecs = []selectionPrioritySpec{
	{PriorityGeneral, "Select.Priority.General", "General", "Select any geometry (the default)."},
	{PriorityPart, "Select.Priority.Part", "Part Priority", "A click selects the whole body / component."},
	{PriorityFace, "Select.Priority.Face", "Face Priority", "A click selects faces."},
	{PriorityEdge, "Select.Priority.Edge", "Edge Priority", "A click selects edges."},
}

// OpenSelectionFilterWindow / CloseSelectionFilterWindow / ToggleSelectionFilterWindow drive the
// Selection Filter & Priority window's visibility (#1222); SelectionFilterWindowOpen reports it.
func (s *Session) OpenSelectionFilterWindow()  { s.selectionFilterWindowOpen = true }
func (s *Session) CloseSelectionFilterWindow() { s.selectionFilterWindowOpen = false }
func (s *Session) ToggleSelectionFilterWindow() {
	s.selectionFilterWindowOpen = !s.selectionFilterWindowOpen
}
func (s *Session) SelectionFilterWindowOpen() bool { return s.selectionFilterWindowOpen }

// selectionFilterCommand is the View tab's Select panel button that opens the Selection Filter &
// Priority window — the full per-kind enable + drag-to-prioritise editor behind the quick combo
// presets (#1222).
func selectionFilterCommand() *CommandDefinition {
	return NewCommand("Select.Filter", "Selection Filter", "Select", func(s *Session) error {
		s.ToggleSelectionFilterWindow()
		return nil
	}).WithTab("View").WithKind(ButtonControl).
		WithTooltip("Choose which entity types are selectable and their pick priority.").
		WithActive(func(s *Session) bool { return s.SelectionFilterWindowOpen() })
}

// selectionPriorityCommands are the View tab's Select panel: a mutually-exclusive combo (like the
// Visual Style box) that sets the no-tool selection priority, biasing both click and box-select
// picking (#912 / Inventor's Select dropdown).
func selectionPriorityCommands() []*CommandDefinition {
	cmds := make([]*CommandDefinition, 0, len(selectionPrioritySpecs))
	for _, sp := range selectionPrioritySpecs {
		cmds = append(cmds, NewCommand(sp.id, sp.label, "Select", func(s *Session) error {
			s.SetSelectionPriority(sp.priority)
			return nil
		}).WithTab("View").WithKind(ComboControl).WithTooltip(sp.tip).
			WithActive(func(s *Session) bool { return s.SelectionPriority() == sp.priority }))
	}
	return cmds
}
