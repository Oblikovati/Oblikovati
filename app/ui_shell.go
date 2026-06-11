// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

// The UI shell surfaces of M05-F12 (#619): the per-environment radial marking
// menu, add-in context-menu injection, the command search, and the View ▸
// Object-visibility toggles.

// EnvironmentChanged fires (After) when the UI environment switches (base ↔
// sketch); the events relay forwards it as ui.environmentChanged.
type EnvironmentChanged struct{ Environment Environment }

// EventID implements event.Event.
func (EnvironmentChanged) EventID() event.TypeID { return tidEnvironmentChanged }

// defaultMarkingMenus is the out-of-the-box radial layout per environment: the
// modeling staples around the cursor in base, the sketch staples in sketch.
func defaultMarkingMenus() map[Environment]wire.MarkingMenuView {
	return map[Environment]wire.MarkingMenuView{
		BaseEnvironment: {
			Environment: BaseEnvironment,
			Quadrants: []wire.MarkingMenuItem{
				{Quadrant: types.QuadrantNorth, CommandID: "Sketch.Create2D"},
				{Quadrant: types.QuadrantEast, CommandID: "Create.Extrude"},
				{Quadrant: types.QuadrantSouth, CommandID: "Modify.Hole"},
				{Quadrant: types.QuadrantWest, CommandID: "Create.Revolve"},
				{Quadrant: types.QuadrantNorthEast, CommandID: "Modify.Fillet"},
				{Quadrant: types.QuadrantSouthEast, CommandID: "Modify.Chamfer"},
			},
		},
		SketchEnvironment: {
			Environment: SketchEnvironment,
			Quadrants: []wire.MarkingMenuItem{
				{Quadrant: types.QuadrantNorth, CommandID: "Sketch.Line"},
				{Quadrant: types.QuadrantEast, CommandID: "Sketch.Circle"},
				{Quadrant: types.QuadrantSouth, CommandID: "Sketch.Rectangle"},
				{Quadrant: types.QuadrantWest, CommandID: "Sketch.Arc"},
				{Quadrant: types.QuadrantNorthEast, CommandID: "Sketch.Dimension"},
			},
			Overflow: []string{"Sketch.Finish"},
		},
	}
}

// MarkingMenu returns the radial menu for an environment (the default until
// customized).
func (s *Session) MarkingMenu(env Environment) wire.MarkingMenuView {
	if menu, ok := s.markingMenus[env]; ok {
		return menu
	}
	return wire.MarkingMenuView{Environment: env}
}

// SetMarkingMenu replaces one environment's radial menu. Quadrants must be unique
// and in range; unknown command ids are tolerated (skipped at render, like popup
// items), so an add-in can declare before registering.
func (s *Session) SetMarkingMenu(menu wire.MarkingMenuView) error {
	seen := map[types.ScreenQuadrant]bool{}
	for _, item := range menu.Quadrants {
		if item.Quadrant > types.QuadrantNorthWest {
			return fmt.Errorf("app: marking-menu quadrant %d out of range (0..7)", item.Quadrant)
		}
		if seen[item.Quadrant] {
			return fmt.Errorf("app: marking-menu quadrant %v declared twice", item.Quadrant)
		}
		seen[item.Quadrant] = true
	}
	s.markingMenus[menu.Environment] = menu
	return nil
}

// SetContextMenuItems replaces one add-in's injected entries for a browser node
// kind ("" injects into every node's menu).
func (s *Session) SetContextMenuItems(addin, kind string, items []wire.ContextMenuItemSpec) error {
	if addin == "" {
		return fmt.Errorf("app: context-menu injection needs the owning add-in id")
	}
	if s.contextMenus[addin] == nil {
		s.contextMenus[addin] = map[string][]wire.ContextMenuItemSpec{}
	}
	if len(items) == 0 {
		delete(s.contextMenus[addin], kind)
		return nil
	}
	s.contextMenus[addin][kind] = items
	return nil
}

// injectedMenuItems returns the add-in entries for a node kind, as runnable menu
// items (clicking executes the named command).
func (s *Session) injectedMenuItems(kind string) []BrowserMenuItem {
	var out []BrowserMenuItem
	for _, byKind := range s.contextMenus {
		for _, scope := range []string{kind, ""} {
			for _, item := range byKind[scope] {
				out = append(out, runnableMenuItem(item))
			}
		}
	}
	return out
}

// runnableMenuItem adapts an injected spec into a browser menu entry.
func runnableMenuItem(item wire.ContextMenuItemSpec) BrowserMenuItem {
	id := item.CommandID
	return BrowserMenuItem{
		Label:   item.Label,
		Enabled: true,
		Invoke:  func(s *Session) error { return s.Execute(id) },
	}
}

// BrowserMenuFor returns a node's context menu: the built-in entries plus any
// add-in injections for its kind (M05-F12). The head renders this, not BrowserMenu.
func BrowserMenuFor(s *Session, n BrowserNode) []BrowserMenuItem {
	return append(BrowserMenu(n), s.injectedMenuItems(n.Kind)...)
}

// SearchCommands finds registered commands whose id, display name, or alias
// contains the query (case-insensitive) — the search box's backing query.
func (s *Session) SearchCommands(query string) []*CommandDefinition {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nil
	}
	var out []*CommandDefinition
	for _, c := range s.commands.All() {
		if strings.Contains(strings.ToLower(c.ID()), needle) ||
			strings.Contains(strings.ToLower(c.DisplayName()), needle) ||
			strings.Contains(strings.ToLower(c.Alias()), needle) {
			out = append(out, c)
		}
	}
	return out
}

// ObjectVisibility returns the View ▸ Object-visibility toggles.
func (s *Session) ObjectVisibility() wire.ObjectVisibilityView { return s.objectVisibility }

// SetObjectVisibility writes the toggles; hidden geometry also stops being
// pickable (the accessors the overlays and pickers share consult these flags).
func (s *Session) SetObjectVisibility(v wire.ObjectVisibilityView) { s.objectVisibility = v }
