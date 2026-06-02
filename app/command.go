// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"github.com/Oblikovati/api/types"
)

// ButtonStyle is how a command renders in the ribbon (text, small icon, large icon).
// The type and its values are defined once in the Apache-2.0 contract
// ([types.ButtonStyle]); this alias keeps the app.ButtonStyle / app.LargeIconButton
// spelling local to the implementation (ADR-0018).
type ButtonStyle = types.ButtonStyle

const (
	TextOnlyButton  = types.TextOnlyButton
	SmallIconButton = types.SmallIconButton
	LargeIconButton = types.LargeIconButton
)

// ControlKind is the UI control a command presents as — Inventor's ControlDefinition
// kinds. The ribbon (ImGui) renders the right widget per kind; the logic is identical.
type ControlKind uint8

const (
	// ButtonControl: a one-shot command button (the default).
	ButtonControl ControlKind = iota
	// ToggleControl: a checkbox/toggle button.
	ToggleControl
	// ComboControl: a drop-down list.
	ComboControl
	// SpinnerControl: a numeric spinner.
	SpinnerControl
)

// CommandDefinition is a registered command — Inventor's ControlDefinition plus its
// CommandManager entry. Its Run is invoked when the button is clicked or the alias is
// typed; Enabled greys it out. State lives in the [Session] it acts on, not here.
type CommandDefinition struct {
	id          string
	displayName string
	tab         string // ribbon tab (e.g. "3D Model"); empty ⇒ the default "Tools" tab
	category    string // ribbon panel within the tab (e.g. "Create")
	kind        ControlKind
	alias       string // typed command alias (e.g. "E" → Extrude), Inventor-style
	tooltip     string
	iconKey     string      // icon asset key (e.g. "extrude"); empty ⇒ no icon
	buttonStyle ButtonStyle // ribbon render style; zero value ⇒ text-only
	enabled     func(*Session) bool
	run         func(*Session) error
}

// NewCommand starts a command definition with the required fields. Use the With*
// methods to set the alias, kind, enable predicate and tooltip.
func NewCommand(id, displayName, category string, run func(*Session) error) *CommandDefinition {
	return &CommandDefinition{id: id, displayName: displayName, category: category, run: run}
}

// WithTab sets the ribbon tab the command appears on (its Category is the panel within
// that tab). Inventor groups commands two levels deep — tab → panel — so e.g. Extrude
// lives on the "3D Model" tab, "Create" panel.
func (c *CommandDefinition) WithTab(tab string) *CommandDefinition { c.tab = tab; return c }

// WithAlias sets the typed command alias.
func (c *CommandDefinition) WithAlias(alias string) *CommandDefinition { c.alias = alias; return c }

// WithKind sets the control kind.
func (c *CommandDefinition) WithKind(k ControlKind) *CommandDefinition { c.kind = k; return c }

// WithTooltip sets the hover tooltip.
func (c *CommandDefinition) WithTooltip(t string) *CommandDefinition { c.tooltip = t; return c }

// WithIcon sets the command's icon asset key (resolved by the head to an SVG glyph).
// Pair it with WithButtonStyle to make the ribbon show the icon; a key with the
// text-only style is ignored by the renderer.
func (c *CommandDefinition) WithIcon(key string) *CommandDefinition { c.iconKey = key; return c }

// WithButtonStyle sets how the command renders in the ribbon (text/small-icon/
// large-icon). Inventor's large buttons head a panel; small ones fill dense grids.
func (c *CommandDefinition) WithButtonStyle(s ButtonStyle) *CommandDefinition {
	c.buttonStyle = s
	return c
}

// WithEnable sets the enable predicate (nil ⇒ always enabled).
func (c *CommandDefinition) WithEnable(p func(*Session) bool) *CommandDefinition {
	c.enabled = p
	return c
}

// ID/DisplayName/Tab/Category/Kind/Alias/Tooltip are the command's metadata.
func (c *CommandDefinition) ID() string               { return c.id }
func (c *CommandDefinition) DisplayName() string      { return c.displayName }
func (c *CommandDefinition) Tab() string              { return c.tab }
func (c *CommandDefinition) Category() string         { return c.category }
func (c *CommandDefinition) Kind() ControlKind        { return c.kind }
func (c *CommandDefinition) Alias() string            { return c.alias }
func (c *CommandDefinition) Tooltip() string          { return c.tooltip }
func (c *CommandDefinition) Icon() string             { return c.iconKey }
func (c *CommandDefinition) ButtonStyle() ButtonStyle { return c.buttonStyle }

// IsEnabled reports whether the command may run in the given session (predicate true
// or absent).
func (c *CommandDefinition) IsEnabled(s *Session) bool {
	return c.enabled == nil || c.enabled(s)
}

// CommandManager is the registry of command definitions — Inventor's CommandManager
// / ControlDefinitions. The ribbon is generated from it (category → panel, command →
// button), so adding a command makes a button appear with no UI code edited.
type CommandManager struct {
	defs       []*CommandDefinition
	byID       map[string]*CommandDefinition
	byAlias    map[string]*CommandDefinition
	categories []string
}

// NewCommandManager returns an empty command registry.
func NewCommandManager() *CommandManager {
	return &CommandManager{byID: map[string]*CommandDefinition{}, byAlias: map[string]*CommandDefinition{}}
}

// Add registers a command, erroring on a duplicate id or alias.
func (m *CommandManager) Add(c *CommandDefinition) error {
	if _, dup := m.byID[c.id]; dup {
		return fmt.Errorf("app: command id %q already registered", c.id)
	}
	if c.alias != "" {
		if _, dup := m.byAlias[c.alias]; dup {
			return fmt.Errorf("app: command alias %q already registered", c.alias)
		}
		m.byAlias[c.alias] = c
	}
	if _, seen := m.categoryIndex(c.category); !seen {
		m.categories = append(m.categories, c.category)
	}
	m.defs = append(m.defs, c)
	m.byID[c.id] = c
	return nil
}

// ByID and ByAlias look up a command.
func (m *CommandManager) ByID(id string) (*CommandDefinition, bool) {
	c, ok := m.byID[id]
	return c, ok
}

func (m *CommandManager) ByAlias(a string) (*CommandDefinition, bool) {
	c, ok := m.byAlias[a]
	return c, ok
}

// All returns every command in registration order.
func (m *CommandManager) All() []*CommandDefinition {
	out := make([]*CommandDefinition, len(m.defs))
	copy(out, m.defs)
	return out
}

// Categories returns the distinct ribbon categories in first-seen order.
func (m *CommandManager) Categories() []string {
	out := make([]string, len(m.categories))
	copy(out, m.categories)
	return out
}

// ByCategory returns the commands in a category, in registration order.
func (m *CommandManager) ByCategory(category string) []*CommandDefinition {
	var out []*CommandDefinition
	for _, c := range m.defs {
		if c.category == category {
			out = append(out, c)
		}
	}
	return out
}

func (m *CommandManager) categoryIndex(category string) (int, bool) {
	for i, c := range m.categories {
		if c == category {
			return i, true
		}
	}
	return -1, false
}
