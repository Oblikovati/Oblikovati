// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

// AddInTaskPanels holds the open modal task panels an add-in asked the host to show. The head
// renders each modally (BeginPopupModal) and calls ResolveTaskPanel on OK/Cancel.
type AddInTaskPanels struct {
	order  []string
	panels map[string]wire.TaskPanelSpec
}

func newAddInTaskPanels() *AddInTaskPanels {
	return &AddInTaskPanels{panels: map[string]wire.TaskPanelSpec{}}
}

// List returns the open task panels in creation order.
func (p *AddInTaskPanels) List() []wire.TaskPanelSpec {
	out := make([]wire.TaskPanelSpec, 0, len(p.order))
	for _, id := range p.order {
		out = append(out, p.panels[id])
	}
	return out
}

// TaskPanels returns the add-in modal-task-panel store.
func (s *Session) TaskPanels() *AddInTaskPanels { return s.taskPanels }

// ShowTaskPanel stores a modal task panel for the head to display; replaces one with the same ID.
// Example: s.ShowTaskPanel(wire.TaskPanelSpec{ID: "mesh.fix", Title: "Fix Mesh"})
func (s *Session) ShowTaskPanel(spec wire.TaskPanelSpec) error {
	if spec.ID == "" || spec.Title == "" {
		return fmt.Errorf("app: task panel needs id and title, got id=%q title=%q", spec.ID, spec.Title)
	}
	if _, exists := s.taskPanels.panels[spec.ID]; !exists {
		s.taskPanels.order = append(s.taskPanels.order, spec.ID)
	}
	s.taskPanels.panels[spec.ID] = spec
	return nil
}

// CloseTaskPanel removes a task panel programmatically (add-in-initiated); emits no event.
// The add-in drives the close, so no TaskPanelClosed event is needed.
func (s *Session) CloseTaskPanel(id string) error {
	if _, ok := s.taskPanels.panels[id]; !ok {
		return fmt.Errorf("app: no task panel %q to close", id)
	}
	s.removeTaskPanel(id)
	return nil
}

// ResolveTaskPanel records the user's OK/Cancel choice: removes the panel and notifies the
// add-in by emitting TaskPanelClosed(After). Called by the head when the user dismisses
// the modal via OK (accepted=true) or Cancel (accepted=false).
func (s *Session) ResolveTaskPanel(id string, accepted bool) error {
	if _, ok := s.taskPanels.panels[id]; !ok {
		return fmt.Errorf("app: no task panel %q to resolve", id)
	}
	s.removeTaskPanel(id)
	event.Emit(s.bus, event.After, TaskPanelClosed{ID: id, Accepted: accepted})
	return nil
}

func (s *Session) removeTaskPanel(id string) {
	delete(s.taskPanels.panels, id)
	for i, oid := range s.taskPanels.order {
		if oid == id {
			s.taskPanels.order = append(s.taskPanels.order[:i], s.taskPanels.order[i+1:]...)
			return
		}
	}
}
