// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/pointcloud"
)

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
	// tabCreateModify is the part ribbon's primary modelling tab — sketches, the solid
	// features (Create/Modify) and patterns. Renamed from "3D Model" when the surfacing and
	// mesh tools were split onto their own tab (see tabSurfacesMesh).
	tabCreateModify = "Create & Modify"
	// tabSurfacesMesh groups the surface, freeform, mesh, point-cloud and mold tools, split out
	// of the modelling tab so each tab stays focused.
	tabSurfacesMesh   = "Surfaces & Mesh"
	tab3DSketch       = "3D Sketch"
	panelWorkFeatures = "Work Features"
	// PanelPointCloud is the single consolidated point-cloud panel: the tool buttons and the
	// display controls (mode selector, size/density sliders, intensity ramp) live together in
	// one grid. It is exported so the head renders this panel with its dedicated grid layout
	// without re-declaring the name. panelPointCloudDisplay is the display-mode commands'
	// category — its selector and controls are folded into PanelPointCloud and the standalone
	// Display panel is dropped.
	PanelPointCloud        = "Point Cloud"
	panelPointCloudDisplay = "Point Cloud Display"
)

// RibbonButton is a command rendered as a ribbon control, with its current enabled
// state resolved from the command's predicate against the session. A button with a
// non-empty Variants list renders as a split button (Inventor's variant flyout): the
// command's own action on the button, the variants in a dropdown.
type RibbonButton struct {
	Command  *CommandDefinition
	Enabled  bool
	Active   bool // renders highlighted (accent) — a stateful toggle that is currently on
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
	Name            string
	Buttons         []RibbonButton
	Selector        *RibbonSelector
	Slider          *RibbonSlider
	PointSizeSlider *RibbonSlider
	IntensityRamp   *RibbonColorRamp
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

// RibbonSlider is a panel rendered as one bounded numeric slider. It is for live session settings
// such as point-cloud render density, where a drag edits state directly rather than invoking a
// command.
type RibbonSlider struct {
	ID      string
	Label   string
	Value   float32
	Min     float32
	Max     float32
	Percent bool
	// Disabled greys the slider and blocks dragging — used when no cloud is the target for a
	// point-cloud display control (several scans attached, none selected). Zero value is enabled.
	Disabled bool
	Tooltip  string
}

// RibbonColorControl is a compact color swatch control in a ribbon panel.
type RibbonColorControl struct {
	ID      string
	Label   string
	Value   [4]float32
	Tooltip string
}

// RibbonColorRamp groups low/high color controls for a scalar display ramp.
type RibbonColorRamp struct {
	Low       RibbonColorControl
	High      RibbonColorControl
	Histogram []float32
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
			b.add(RibbonButton{Command: c, Enabled: c.IsEnabled(s), Active: c.IsActive(s), Variants: resolveVariants(c, s)})
		}
	}
	finalizeSelectors(b.tabs, s)
	injectBuiltInRibbonPanels(b.tabs, key, env, s)
	return Ribbon{Key: key, Tabs: b.tabs}
}

func injectBuiltInRibbonPanels(tabs []RibbonTab, key RibbonKey, env Environment, s *Session) {
	if key != PartRibbon || !environmentShows(BaseEnvironment, env) {
		return
	}
	attachPointCloudDisplayControls(tabs, s)
}

func attachPointCloudDisplayControls(tabs []RibbonTab, s *Session) {
	for ti := range tabs {
		if tabs[ti].Name != tabSurfacesMesh {
			continue
		}
		attachPointCloudDisplayControlsToTab(&tabs[ti], s)
		return
	}
}

// attachPointCloudDisplayControlsToTab folds the display controls into the single Point Cloud
// panel: the density and point-size sliders and the intensity ramp are injected onto it, and the
// separate Point Cloud Display panel's mode selector is moved over before that panel is dropped —
// so the tab shows one consolidated point-cloud grid instead of two adjacent panels.
func attachPointCloudDisplayControlsToTab(tab *RibbonTab, s *Session) {
	tools := tab.panel(PanelPointCloud)
	if tools == nil {
		return
	}
	// The size/density controls act on the target cloud: a single attached scan is always the
	// target (no selection needed), but with several attached the user must select one first, so
	// the sliders grey out until they do (#645).
	_, hasTarget := s.targetPointCloud()
	tools.Slider = pointCloudRenderDensitySlider(s, hasTarget)
	tools.PointSizeSlider = pointCloudPointSizeSlider(s, hasTarget)
	tools.IntensityRamp = pointCloudIntensityRampControls(s)
	if display := tab.panel(panelPointCloudDisplay); display != nil {
		tools.Selector = display.Selector
	}
	tab.dropPanel(panelPointCloudDisplay)
}

func pointCloudRenderDensitySlider(s *Session, enabled bool) *RibbonSlider {
	return &RibbonSlider{
		ID:       "PointCloud.RenderDensity",
		Label:    "Density",
		Value:    s.PointCloudRenderDensity(),
		Min:      0,
		Max:      100,
		Percent:  true,
		Disabled: !enabled,
		Tooltip:  "Point-cloud render density — draw a stable random percentage of scan points.",
	}
}

func pointCloudPointSizeSlider(s *Session, enabled bool) *RibbonSlider {
	return &RibbonSlider{
		ID:       "PointCloud.PointSize",
		Label:    "Point Size",
		Value:    s.PointCloudPointSize(),
		Min:      1,
		Max:      10,
		Disabled: !enabled,
		Tooltip:  "Point-cloud point size - draw scan points from 1 to 10 pixels wide.",
	}
}

func pointCloudIntensityRampControls(s *Session) *RibbonColorRamp {
	pc, ok := s.targetPointCloud()
	if !ok || pc.DisplayMode() != types.PointCloudDisplayModeIntensity {
		return nil
	}
	low, high := s.PointCloudIntensityRamp()
	return &RibbonColorRamp{
		Histogram: s.cachedIntensityHistogram(pc),
		Low: RibbonColorControl{
			ID:      "PointCloud.IntensityLow",
			Label:   "Low",
			Value:   low,
			Tooltip: "Low-intensity color.",
		},
		High: RibbonColorControl{
			ID:      "PointCloud.IntensityHigh",
			Label:   "High",
			Value:   high,
			Tooltip: "High-intensity color.",
		},
	}
}

const pointCloudIntensityHistogramBins = 32

// intensityHistogramKey identifies the displayed set a memoized histogram was built for. The
// displayed set is fingerprinted by its backing array (pointcloud reallocates displayCache only
// when the set is rebuilt), so first+length distinguishes an unchanged set from a rebuilt one
// without reaching the model's unexported displaySignature. min/max/bins cover the other inputs.
type intensityHistogramKey struct {
	first  *pointcloud.PointSample
	length int
	min    float64
	max    float64
	bins   int
}

// intensityHistogramMemo is one cloud's last-built histogram plus the key it was built for.
type intensityHistogramMemo struct {
	key  intensityHistogramKey
	hist []float32
}

// cachedIntensityHistogram returns pc's normalized intensity histogram, recomputing only when the
// cloud's displayed set changes. The binning loop is O(N) over DisplayedSamples() and the ribbon
// rebuilds this every frame the intensity ramp shows, so a 24M-point scan would otherwise re-bin
// every frame; the memo makes an unchanged set cost O(1) (#645).
func (s *Session) cachedIntensityHistogram(pc *pointcloud.PointCloud) []float32 {
	const bins = pointCloudIntensityHistogramBins
	samples := pc.DisplayedSamples()
	key, ok := intensityHistogramKeyFor(pc, samples, bins)
	if !ok {
		return nil
	}
	if m, hit := s.pcIntensityHistograms[pc]; hit && m.key == key {
		return m.hist
	}
	hist := normalizeHistogram(binIntensities(samples, key.min, key.max, bins))
	if s.pcIntensityHistograms == nil {
		s.pcIntensityHistograms = map[*pointcloud.PointCloud]intensityHistogramMemo{}
	}
	s.pcIntensityHistograms[pc] = intensityHistogramMemo{key: key, hist: hist}
	return hist
}

// intensityHistogramKeyFor builds the memo key, reporting false when the cloud has no usable
// intensity range or bin count (the histogram is nil then, so there is nothing to cache).
func intensityHistogramKeyFor(pc *pointcloud.PointCloud, samples []pointcloud.PointSample, bins int) (intensityHistogramKey, bool) {
	min, max, ok := pc.IntensityRange()
	if !ok || max <= min || bins <= 0 {
		return intensityHistogramKey{}, false
	}
	var first *pointcloud.PointSample
	if len(samples) > 0 {
		first = &samples[0]
	}
	return intensityHistogramKey{first: first, length: len(samples), min: min, max: max, bins: bins}, true
}

// pointCloudIntensityHistogram computes the normalized histogram without memoization — the direct
// path used by unit tests; the ribbon goes through Session.cachedIntensityHistogram.
func pointCloudIntensityHistogram(pc *pointcloud.PointCloud, bins int) []float32 {
	min, max, ok := pc.IntensityRange()
	if !ok || max <= min || bins <= 0 {
		return nil
	}
	return normalizeHistogram(binIntensities(pc.DisplayedSamples(), min, max, bins))
}

// binIntensities buckets the samples' intensities into bins, returning the counts and the tallest
// bin so normalizeHistogram can scale to it. Samples without intensity are skipped.
func binIntensities(samples []pointcloud.PointSample, min, max float64, bins int) (counts []int, peak int) {
	counts = make([]int, bins)
	for _, sample := range samples {
		if !sample.HasIntensity {
			continue
		}
		i := int(((sample.Intensity - min) / (max - min)) * float64(bins))
		if i < 0 {
			i = 0
		}
		if i >= bins {
			i = bins - 1
		}
		counts[i]++
		if counts[i] > peak {
			peak = counts[i]
		}
	}
	return counts, peak
}

// normalizeHistogram scales bin counts to 0..1 against the tallest bin, returning nil when no
// sample carried intensity (peak 0) so the ramp control shows an empty chart.
func normalizeHistogram(counts []int, peak int) []float32 {
	if peak == 0 {
		return nil
	}
	out := make([]float32, len(counts))
	for i, n := range counts {
		out[i] = float32(n) / float32(peak)
	}
	return out
}

// resolveVariants turns a head command's variant definitions into dropdown entries with
// each variant's enabled state resolved against the session this frame. A PopupControl's
// entries come from the registry instead — its menu lists other registered commands by id
// (M05-F03, #247).
func resolveVariants(c *CommandDefinition, s *Session) []RibbonVariant {
	if c.kind == PopupControl {
		return resolvePopupItems(c, s)
	}
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

// resolvePopupItems looks a popup's item ids up in the registry, skipping unknown ids so
// the menu shows what exists (an add-in may declare items before registering them all).
func resolvePopupItems(c *CommandDefinition, s *Session) []RibbonVariant {
	var out []RibbonVariant
	for _, id := range c.popupItems {
		item, ok := s.commands.ByID(id)
		if !ok {
			continue
		}
		out = append(out, RibbonVariant{
			CommandID: item.ID(),
			Label:     item.DisplayName(),
			Tooltip:   item.Tooltip(),
			Enabled:   item.IsEnabled(s),
		})
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
	// A command may declare several tabs (WithTabs); render its button on each, in order.
	for _, tab := range btn.Command.ribbonTabs() {
		ti := b.tabAt(tab)
		pi := b.panelAt(tab, ti, btn.Command.Category())
		b.tabs[ti].Panels[pi].Buttons = append(b.tabs[ti].Panels[pi].Buttons, btn)
	}
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

// panel returns a pointer to this tab's panel with the given name, or nil — for the built-in
// injection step, which mutates panels (folding the display controls into the Point Cloud panel).
func (t *RibbonTab) panel(name string) *RibbonPanel {
	for i := range t.Panels {
		if t.Panels[i].Name == name {
			return &t.Panels[i]
		}
	}
	return nil
}

// dropPanel removes the named panel from the tab, keeping the order of the rest. Used to retire
// the Point Cloud Display panel once its controls are folded into the Point Cloud panel.
func (t *RibbonTab) dropPanel(name string) {
	kept := t.Panels[:0]
	for _, p := range t.Panels {
		if p.Name != name {
			kept = append(kept, p)
		}
	}
	t.Panels = kept
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
