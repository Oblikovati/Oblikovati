// SPDX-License-Identifier: GPL-2.0-only

package app

// The ribbon and browser are MODELS built from live state each frame — Dear ImGui
// renders these structs (core/09); the logic here is pure and testable. The ribbon is
// generated from the command registry and mirrors Inventor's two-level layout: tab →
// panel → command. A command's Tab picks the ribbon tab, its Category the panel within
// it, so a new command (or an add-in's command) appears as a button with no UI code edited.

// DefaultTab is where commands with no explicit tab land — Inventor's catch-all "Tools"
// tab, so an under-specified or add-in command is still reachable.
const DefaultTab = "Tools"

// RibbonButton is a command rendered as a ribbon control, with its current enabled
// state resolved from the command's predicate against the session.
type RibbonButton struct {
	Command *CommandDefinition
	Enabled bool
}

// RibbonPanel groups the buttons of one command category within a tab.
type RibbonPanel struct {
	Name    string
	Buttons []RibbonButton
}

// RibbonTab is one tab of the ribbon, holding the panels whose commands target it.
type RibbonTab struct {
	Name   string
	Panels []RibbonPanel
}

// Ribbon is the full ribbon model for a frame.
type Ribbon struct {
	Tabs []RibbonTab
}

// BuildRibbon generates the ribbon from the session's command registry, grouping each
// command under its tab then its panel. Tabs, panels, and buttons all follow command
// registration order; each button carries its live enabled state.
func BuildRibbon(s *Session) Ribbon {
	b := ribbonBuilder{tabIndex: map[string]int{}, panelIndex: map[string]map[string]int{}}
	for _, c := range s.commands.All() {
		b.add(c, c.IsEnabled(s))
	}
	return Ribbon{Tabs: b.tabs}
}

// ribbonBuilder accumulates commands into ordered tabs/panels, remembering first-seen
// positions so the layout is stable across frames.
type ribbonBuilder struct {
	tabs       []RibbonTab
	tabIndex   map[string]int
	panelIndex map[string]map[string]int // tab name → panel name → index within the tab
}

func (b *ribbonBuilder) add(c *CommandDefinition, enabled bool) {
	tab := c.Tab()
	if tab == "" {
		tab = DefaultTab
	}
	ti := b.tabAt(tab)
	pi := b.panelAt(tab, ti, c.Category())
	b.tabs[ti].Panels[pi].Buttons = append(
		b.tabs[ti].Panels[pi].Buttons, RibbonButton{Command: c, Enabled: enabled},
	)
}

func (b *ribbonBuilder) tabAt(name string) int {
	if i, ok := b.tabIndex[name]; ok {
		return i
	}
	i := len(b.tabs)
	b.tabIndex[name] = i
	b.panelIndex[name] = map[string]int{}
	b.tabs = append(b.tabs, RibbonTab{Name: name})
	return i
}

func (b *ribbonBuilder) panelAt(tab string, ti int, panel string) int {
	if i, ok := b.panelIndex[tab][panel]; ok {
		return i
	}
	i := len(b.tabs[ti].Panels)
	b.panelIndex[tab][panel] = i
	b.tabs[ti].Panels = append(b.tabs[ti].Panels, RibbonPanel{Name: panel})
	return i
}

// Tab returns the tab with the given name, or false.
func (r Ribbon) Tab(name string) (RibbonTab, bool) {
	for _, t := range r.Tabs {
		if t.Name == name {
			return t, true
		}
	}
	return RibbonTab{}, false
}

// Panel returns the first panel with the given name across all tabs, or false. Panel
// names are unique in practice, so this is a convenient cross-tab lookup.
func (r Ribbon) Panel(name string) (RibbonPanel, bool) {
	for _, t := range r.Tabs {
		if p, ok := t.Panel(name); ok {
			return p, true
		}
	}
	return RibbonPanel{}, false
}

// Panel returns this tab's panel with the given name, or false.
func (t RibbonTab) Panel(name string) (RibbonPanel, bool) {
	for _, p := range t.Panels {
		if p.Name == name {
			return p, true
		}
	}
	return RibbonPanel{}, false
}
