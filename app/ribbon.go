// SPDX-License-Identifier: GPL-2.0-only

package app

// The ribbon and browser are MODELS built from live state each frame — Dear ImGui
// renders these structs (core/09); the logic here is pure and testable. The ribbon is
// generated from the command registry and mirrors Inventor's two-level layout: tab →
// panel → command. A command's Tab picks the ribbon tab, its Category the panel within
// it, so a new command (or an add-in's command) appears as a button with no UI code edited.
//
// Inventor has one ribbon per document type plus ZeroDoc, switched by the active document
// (RibbonUI_Overview); BuildRibbon selects that ribbon by RibbonKey and includes only the
// commands scoped to it and to the current environment (so the Sketch tab is contextual).

// DefaultTab is where commands with no explicit tab land — Inventor's catch-all "Tools"
// tab, so an under-specified or add-in command is still reachable.
const DefaultTab = "Tools"

// Standard ribbon tab and panel names, mirroring Inventor's layout. Centralized here so
// the command registrations reference one source instead of repeating the display strings.
const (
	tab3DModel        = "3D Model"
	tab3DSketch       = "3D Sketch"
	panelWorkFeatures = "Work Features"
)

// RibbonButton is a command rendered as a ribbon control, with its current enabled
// state resolved from the command's predicate against the session. A button with a
// non-empty Variants list renders as a split button (Inventor's variant flyout): the
// command's own action on the button, the variants in a dropdown.
type RibbonButton struct {
	Command  *CommandDefinition
	Enabled  bool
	Variants []RibbonVariant
}

// RibbonVariant is one entry of a split button's dropdown: the command to run when chosen
// and the label/tooltip to show for it, with its enabled state resolved this frame.
type RibbonVariant struct {
	CommandID string
	Label     string
	Tooltip   string
	Enabled   bool
}

// RibbonPanel groups the buttons of one command category within a tab. When Selector is
// non-nil the panel renders as a selection box (a drop-down) instead of a button grid — used
// for mutually-exclusive choices like the View tab's Visual Style (Inventor's combo control).
type RibbonPanel struct {
	Name     string
	Buttons  []RibbonButton
	Selector *RibbonSelector
}

// RibbonSelectOption is one entry of a [RibbonSelector] drop-down: the command it runs when
// chosen and the label shown for it.
type RibbonSelectOption struct {
	CommandID string
	Label     string
	Tooltip   string
}

// RibbonSelector is a panel rendered as a drop-down list: its options and the index of the
// currently-selected one (resolved from each command's IsActive predicate this frame).
type RibbonSelector struct {
	Options       []RibbonSelectOption
	SelectedIndex int
}

// RibbonTab is one tab of the ribbon, holding the panels whose commands target it.
type RibbonTab struct {
	Name   string
	Panels []RibbonPanel
}

// Ribbon is the full ribbon model for a frame: the document ribbon selected this frame and
// its tabs.
type Ribbon struct {
	Key  RibbonKey
	Tabs []RibbonTab
}

// BuildRibbon generates the ribbon for the active document (ZeroDoc when none is open),
// including only the commands scoped to that ribbon and to the current environment — so the
// part ribbon's contextual Sketch tab appears only while a sketch is open. Tabs, panels, and
// buttons follow command registration order; each button carries its live enabled state.
func BuildRibbon(s *Session) Ribbon {
	key := ribbonKeyForDocument(s.ActiveDocument())
	env := currentEnvironment(s)
	b := ribbonBuilder{tabIndex: map[string]int{}, panelIndex: map[string]map[string]int{}}
	for _, c := range s.commands.All() {
		// Variant commands are flyout-only: they render inside their head's dropdown
		// (resolveVariants below), never as their own panel button.
		if c.isVariant {
			continue
		}
		if c.appearsOnRibbon(key) && environmentShows(c.environment, env) {
			b.add(RibbonButton{Command: c, Enabled: c.IsEnabled(s), Variants: resolveVariants(c, s)})
		}
	}
	finalizeSelectors(b.tabs, s)
	return Ribbon{Key: key, Tabs: b.tabs}
}

// resolveVariants turns a head command's variant definitions into dropdown entries with
// each variant's enabled state resolved against the session this frame.
func resolveVariants(c *CommandDefinition, s *Session) []RibbonVariant {
	if len(c.variants) == 0 {
		return nil
	}
	out := make([]RibbonVariant, len(c.variants))
	for i, v := range c.variants {
		out[i] = RibbonVariant{
			CommandID: v.ID(),
			Label:     v.DisplayName(),
			Tooltip:   v.Tooltip(),
			Enabled:   v.IsEnabled(s),
		}
	}
	return out
}

// finalizeSelectors turns any panel whose commands are ComboControls into a selection box:
// its options are the panel's commands and its SelectedIndex is the one whose IsActive
// predicate holds this frame (default 0). A panel mixes buttons or combos, never both, so the
// first button decides the panel's kind.
func finalizeSelectors(tabs []RibbonTab, s *Session) {
	for ti := range tabs {
		for pi := range tabs[ti].Panels {
			p := &tabs[ti].Panels[pi]
			if len(p.Buttons) == 0 || p.Buttons[0].Command.Kind() != ComboControl {
				continue
			}
			sel := &RibbonSelector{}
			for _, btn := range p.Buttons {
				if btn.Command.IsActive(s) {
					sel.SelectedIndex = len(sel.Options)
				}
				sel.Options = append(sel.Options, RibbonSelectOption{
					CommandID: btn.Command.ID(),
					Label:     btn.Command.DisplayName(),
					Tooltip:   btn.Command.Tooltip(),
				})
			}
			p.Selector = sel
		}
	}
}

// ribbonBuilder accumulates commands into ordered tabs/panels, remembering first-seen
// positions so the layout is stable across frames.
type ribbonBuilder struct {
	tabs       []RibbonTab
	tabIndex   map[string]int
	panelIndex map[string]map[string]int // tab name → panel name → index within the tab
}

func (b *ribbonBuilder) add(btn RibbonButton) {
	tab := btn.Command.Tab()
	if tab == "" {
		tab = DefaultTab
	}
	ti := b.tabAt(tab)
	pi := b.panelAt(tab, ti, btn.Command.Category())
	b.tabs[ti].Panels[pi].Buttons = append(b.tabs[ti].Panels[pi].Buttons, btn)
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
