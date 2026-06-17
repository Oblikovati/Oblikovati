// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
)

// ButtonStyle is how a command renders in the ribbon (text, small icon with label,
// large icon, compact icon-only). The type and its values are defined once in the
// Apache-2.0 contract ([types.ButtonStyle]); this alias keeps the app.ButtonStyle /
// app.LargeIconButton spelling local to the implementation (ADR-0018).
type ButtonStyle = types.ButtonStyle

const (
	TextOnlyButton    = types.TextOnlyButton
	SmallIconButton   = types.SmallIconButton
	LargeIconButton   = types.LargeIconButton
	CompactIconButton = types.CompactIconButton
)

// ControlKind is the UI control a command presents as — the ControlDefinition
// kinds. Defined once in the Apache-2.0 contract ([types.ControlKind], M05-F03);
// this alias keeps the app.ButtonControl spelling local to the implementation
// (ADR-0018). The ribbon (ImGui) renders the right widget per kind.
type ControlKind = types.ControlKind

const (
	ButtonControl  = types.ButtonControl
	ToggleControl  = types.ToggleControl
	ComboControl   = types.ComboControl
	SpinnerControl = types.SpinnerControl
	PopupControl   = types.PopupControl
)

// CommandDefinition is a registered command — Inventor's ControlDefinition plus its
// CommandManager entry. Its Run is invoked when the button is clicked or the alias is
// typed; Enabled greys it out. State lives in the [Session] it acts on, not here.
type CommandDefinition struct {
	id               string
	displayName      string
	tab              string // ribbon tab (e.g. "3D Model"); empty ⇒ the default "Tools" tab
	category         string // ribbon panel within the tab (e.g. "Create")
	kind             ControlKind
	alias            string // typed command alias (e.g. "E" → Extrude), Inventor-style
	defaultChord     string // predefined keyboard chord (e.g. "Ctrl+N"); single-letter chords are not allowed (M26)
	tooltip          string
	tooltipTitle     string      // progressive tooltip title (M05-F09)
	tooltipExpanded  string      // progressive tooltip long text, shown after a longer hover
	iconKey          string      // icon asset key (e.g. "extrude"); empty ⇒ no icon
	buttonStyle      ButtonStyle // ribbon render style; zero value ⇒ text-only
	ribbons          []RibbonKey // ribbons this command appears on; empty ⇒ the Part ribbon
	environment      Environment // ribbon environment; BaseEnvironment ⇒ always shown
	enabled          func(*Session) bool
	active           func(*Session) bool // for a ComboControl option: is this the current selection?
	forcedActive     bool                // runtime active flag (commands.setState); see hasForcedActive
	hasForcedActive  bool                // true ⇒ forcedActive overrides the active predicate
	forcedEnabled    bool                // runtime enabled flag (commands.setState); see hasForcedEnabled
	hasForcedEnabled bool                // true ⇒ forcedEnabled overrides the enabled predicate
	run              func(*Session) error
	variants         []*CommandDefinition // split-button dropdown entries (Inventor's variant flyout)
	isVariant        bool                 // true ⇒ reachable only via a head command's dropdown, never its own panel button
	popupItems       []string             // for PopupControl: ids of the registered commands its menu lists (M05-F03)
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

// WithDefaultChord predefines the command's default keyboard shortcut as a full chord
// ("Ctrl+N", "Ctrl+Shift+S", "F7"). Use this — not WithAlias — for shipped shortcuts: a bare
// single-letter is reserved for the keybinding editor (M26), so the default must carry Shift or
// Control (or be a non-letter key). The user can still rebind it.
func (c *CommandDefinition) WithDefaultChord(chord string) *CommandDefinition {
	c.defaultChord = chord
	return c
}

// DefaultChord returns the command's predefined chord string ("" if none).
func (c *CommandDefinition) DefaultChord() string { return c.defaultChord }

// WithKind sets the control kind.
func (c *CommandDefinition) WithKind(k ControlKind) *CommandDefinition { c.kind = k; return c }

// WithTooltip sets the hover tooltip.
func (c *CommandDefinition) WithTooltip(t string) *CommandDefinition { c.tooltip = t; return c }

// WithTooltipDetail sets the progressive tooltip (M05-F09): title heads the hover
// tip; expanded appears after a longer hover — the ProgressiveToolTip equivalent.
func (c *CommandDefinition) WithTooltipDetail(title, expanded string) *CommandDefinition {
	c.tooltipTitle, c.tooltipExpanded = title, expanded
	return c
}

// TooltipTitle / TooltipExpanded are the progressive-tooltip parts ("" when unset).
func (c *CommandDefinition) TooltipTitle() string    { return c.tooltipTitle }
func (c *CommandDefinition) TooltipExpanded() string { return c.tooltipExpanded }

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

// WithRibbons sets the document ribbon(s) the command appears on (e.g. PartRibbon, or several
// for a command shared across document types). Unset ⇒ the Part ribbon, the only modeling
// ribbon today. This is Inventor's "which of the seven ribbons does this control belong to".
func (c *CommandDefinition) WithRibbons(keys ...RibbonKey) *CommandDefinition {
	c.ribbons = keys
	return c
}

// WithEnvironment scopes the command to a ribbon environment. A SketchEnvironment command
// forms the contextual Sketch tab and shows only while a sketch is open; the default
// BaseEnvironment always shows.
func (c *CommandDefinition) WithEnvironment(e Environment) *CommandDefinition {
	c.environment = e
	return c
}

// WithEnable sets the enable predicate (nil ⇒ always enabled).
func (c *CommandDefinition) WithEnable(p func(*Session) bool) *CommandDefinition {
	c.enabled = p
	return c
}

// WithVariants attaches a split-button dropdown to this command — Inventor's variant
// flyout, where one ribbon head (e.g. "Rectangle") fronts a list of related geometry
// tools (Three Point, Two Point Center, …). The head still runs its own action when
// clicked; the variants appear only in its dropdown, never as standalone panel buttons,
// so the canonical panel-head count is unchanged. Variants are registered for id lookup
// (so [Session.Execute] can run them) by [RegisterStandardCommands].
func (c *CommandDefinition) WithVariants(variants ...*CommandDefinition) *CommandDefinition {
	for _, v := range variants {
		v.isVariant = true
	}
	c.variants = variants
	return c
}

// Variants returns this command's split-button dropdown entries (nil if it is a plain
// button). The slice is the live backing array; callers must not mutate it.
func (c *CommandDefinition) Variants() []*CommandDefinition { return c.variants }

// WithPopupItems makes this command a PopupControl listing other REGISTERED commands
// by id — the CommandBarPopUp equivalent (M05-F03, #247). Unlike WithVariants, the
// items are ordinary commands resolved from the registry at ribbon-build time, so an
// add-in can group its existing buttons under one menu without re-declaring them.
// Unknown ids are skipped at build time (the menu shows what exists).
func (c *CommandDefinition) WithPopupItems(ids ...string) *CommandDefinition {
	c.kind = PopupControl
	c.popupItems = ids
	return c
}

// PopupItems returns the command ids a PopupControl lists (nil otherwise).
func (c *CommandDefinition) PopupItems() []string { return c.popupItems }

// WithActive sets the predicate that reports whether this command is the *current* selection
// of its combo group — used to drive the highlighted item of a ribbon selection box (a panel
// of ComboControl commands, e.g. the View tab's Visual Style). Unset ⇒ never the selection.
func (c *CommandDefinition) WithActive(p func(*Session) bool) *CommandDefinition {
	c.active = p
	return c
}

// IsActive reports whether this command renders as active/selected: a runtime flag set via
// SetActiveState (an add-in's commands.setState) wins, else the WithActive predicate.
func (c *CommandDefinition) IsActive(s *Session) bool {
	if c.hasForcedActive {
		return c.forcedActive
	}
	return c.active != nil && c.active(s)
}

// SetActiveState sets the runtime active flag (overriding any WithActive predicate) so an
// add-in can toggle a stateful button's highlighted look via commands.setState.
func (c *CommandDefinition) SetActiveState(active bool) {
	c.forcedActive = active
	c.hasForcedActive = true
}

// SetDisplayName relabels the command at runtime (commands.setState), e.g. a toggle button
// switching between "Presenter" and "Presenting".
func (c *CommandDefinition) SetDisplayName(name string) { c.displayName = name }

// Ribbons returns the ribbons the command appears on, resolving the default (the Part ribbon)
// so a caller always sees the effective placement. The slice is a copy.
func (c *CommandDefinition) Ribbons() []RibbonKey {
	if len(c.ribbons) == 0 {
		return []RibbonKey{PartRibbon}
	}
	return append([]RibbonKey(nil), c.ribbons...)
}

// appearsOnRibbon reports whether the command belongs on the given ribbon. With no explicit
// ribbons it defaults to the Part ribbon (the only modeling ribbon implemented today).
func (c *CommandDefinition) appearsOnRibbon(key RibbonKey) bool {
	if len(c.ribbons) == 0 {
		return key == PartRibbon
	}
	for _, k := range c.ribbons {
		if k == key {
			return true
		}
	}
	return false
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
func (c *CommandDefinition) Environment() Environment { return c.environment }

// IsEnabled reports whether the command may run in the given session (predicate true
// or absent).
func (c *CommandDefinition) IsEnabled(s *Session) bool {
	if c.hasForcedEnabled {
		return c.forcedEnabled
	}
	return c.enabled == nil || c.enabled(s)
}

// SetEnabledState sets a runtime enabled flag (overriding any WithEnabled predicate) so an
// add-in can grey out or restore its commands via commands.setState — e.g. disabling its
// presenter/follow controls until the user joins a session.
func (c *CommandDefinition) SetEnabledState(enabled bool) {
	c.forcedEnabled = enabled
	c.hasForcedEnabled = true
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
