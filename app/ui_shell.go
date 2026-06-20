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

// RepeatMenuEntry returns the right-click "Repeat <command>" entry shown when idle: its display
// label and the command id to re-invoke. ok is false when a tool/command is active (the menu then
// offers in-command actions, not Repeat) or there is no prior command (#915 C5).
func (s *Session) RepeatMenuEntry() (label, commandID string, ok bool) {
	if s.ActiveTool() != nil {
		return "", "", false
	}
	id, has := s.LastCommandID()
	if !has {
		return "", "", false
	}
	name := id
	if c, found := s.commands.ByID(id); found {
		name = c.DisplayName()
	}
	return "Repeat " + name, id, true
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

// Add-in UI environments (M05-F16, #667): an add-in registers its own contextual
// environment (values ≥ 2; the built-in base/sketch stay 0/1) and activates it;
// commands registered with the value form its tabs, exactly like the sketch
// environment. The active add-in environment overrides the sketch resolution.

// RegisterEnvironment declares an add-in environment.
func (s *Session) RegisterEnvironment(env Environment, name string) error {
	if env <= SketchEnvironment {
		return fmt.Errorf("app: environment %d is reserved (0 base, 1 sketch); add-in environments start at 2", env)
	}
	if name == "" {
		return fmt.Errorf("app: environment %d needs a display name", env)
	}
	s.addinEnvironments[env] = name
	return nil
}

// AddInEnvironments returns the registered add-in environments.
func (s *Session) AddInEnvironments() map[Environment]string {
	out := make(map[Environment]string, len(s.addinEnvironments))
	for env, name := range s.addinEnvironments {
		out[env] = name
	}
	return out
}

// ActivateEnvironment enters a registered add-in environment; BaseEnvironment
// leaves it. The switch reaches observers as EnvironmentChanged.
func (s *Session) ActivateEnvironment(env Environment) error {
	if env == BaseEnvironment {
		if s.activeAddInEnv != BaseEnvironment {
			s.activeAddInEnv = BaseEnvironment
			event.Emit(s.bus, event.After, EnvironmentChanged{Environment: CurrentEnvironment(s)})
		}
		return nil
	}
	if _, ok := s.addinEnvironments[env]; !ok {
		return fmt.Errorf("app: environment %d is not registered", env)
	}
	if s.activeAddInEnv != env {
		s.activeAddInEnv = env
		event.Emit(s.bus, event.After, EnvironmentChanged{Environment: env})
	}
	return nil
}

// ActiveAddInEnvironment returns the active add-in environment (base when none).
func (s *Session) ActiveAddInEnvironment() Environment { return s.activeAddInEnv }
