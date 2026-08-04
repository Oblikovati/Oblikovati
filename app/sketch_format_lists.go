// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/linetype"
)

// The Format panel's three selection lists as ribbon commands (#2015). Each shows the selection's
// current value with a preview — a dash pattern, a colour swatch, a stroke sample — because a
// line type is picked by seeing it, not by reading its name. Entry 0 of every list is Default,
// which clears that field's override.

// FormatListKind names which of the three lists a command drives.
type FormatListKind uint8

const (
	// LineTypeList picks the dash pattern.
	LineTypeList FormatListKind = iota
	// ColorList picks the colour.
	ColorList
	// LineWeightList picks the stroke width.
	LineWeightList
)

// FormatListEntry is one row: its label, and the value it applies. Exactly one of the value
// fields is meaningful, chosen by the owning list's kind.
type FormatListEntry struct {
	Label      string
	LineType   string
	Color      types.Color
	LineWeight float64
}

// formatDefaultLabel is entry 0 of every list — the one that clears the override.
const formatDefaultLabel = "Default"

// FormatListEntries returns the rows of one list, Default first.
func FormatListEntries(kind FormatListKind) []FormatListEntry {
	switch kind {
	case LineTypeList:
		return lineTypeEntries()
	case ColorList:
		return colorEntries()
	default:
		return lineWeightEntries()
	}
}

// lineTypeEntries is Default plus the built-in dash patterns.
func lineTypeEntries() []FormatListEntry {
	out := []FormatListEntry{{Label: formatDefaultLabel}}
	for _, t := range []types.SketchLineType{
		types.SketchLineContinuous, types.SketchLineDashed,
		types.SketchLineHidden, types.SketchLineCenter, types.SketchLinePhantom,
	} {
		out = append(out, FormatListEntry{Label: string(t), LineType: string(t)})
	}
	return out
}

// colorEntries is Default plus a small palette. It is deliberately short: the panel picks a
// colour for DWG interoperability, where a handful of index colours cover almost everything, and
// a full picker belongs in the properties dialog rather than a ribbon dropdown.
func colorEntries() []FormatListEntry {
	out := []FormatListEntry{{Label: formatDefaultLabel}}
	for _, c := range []struct {
		name    string
		r, g, b uint8
	}{
		{"Red", 255, 0, 0}, {"Yellow", 255, 255, 0}, {"Green", 0, 255, 0},
		{"Cyan", 0, 255, 255}, {"Blue", 0, 0, 255}, {"Magenta", 255, 0, 255},
		{"White", 255, 255, 255},
	} {
		out = append(out, FormatListEntry{Label: c.name, Color: types.NewColor(c.r, c.g, c.b)})
	}
	return out
}

// lineWeightEntries is Default plus the standard plotted widths in millimetres.
func lineWeightEntries() []FormatListEntry {
	out := []FormatListEntry{{Label: formatDefaultLabel}}
	for _, w := range []float64{0.13, 0.18, 0.25, 0.35, 0.50, 0.70, 1.00} {
		out = append(out, FormatListEntry{Label: fmt.Sprintf("%.2f mm", w), LineWeight: w})
	}
	return out
}

// FormatListSelection reports which row of a list the current selection sits on, or 0 (Default)
// when the selection has no override or its entities disagree.
func (s *Session) FormatListSelection(kind FormatListKind) int {
	f := s.SelectionFormat()
	for i, e := range FormatListEntries(kind) {
		if formatEntryMatches(kind, e, f.LineType, f.Color, f.LineWeight) {
			return i
		}
	}
	return 0
}

// formatEntryMatches reports whether a row holds the selection's current value for its kind.
func formatEntryMatches(kind FormatListKind, e FormatListEntry, lineType string, color types.Color, weight float64) bool {
	switch kind {
	case LineTypeList:
		return e.LineType == lineType
	case ColorList:
		return e.Color == color
	default:
		return e.LineWeight == weight
	}
}

// ChooseFormatListEntry applies row i of a list to the selection.
func (s *Session) ChooseFormatListEntry(kind FormatListKind, i int) int {
	entries := FormatListEntries(kind)
	if i < 0 || i >= len(entries) {
		return 0
	}
	e := entries[i]
	switch kind {
	case LineTypeList:
		return s.SetSelectionLineType(e.LineType)
	case ColorList:
		return s.SetSelectionColor(e.Color)
	default:
		return s.SetSelectionLineWeight(e.LineWeight)
	}
}

// FormatListPattern is the dash pattern a line-type row previews, or nil for a solid sample.
func FormatListPattern(e FormatListEntry) []float64 {
	if e.LineType == "" {
		return nil
	}
	return linetype.Builtin(types.SketchLineType(e.LineType))
}

// sketchFormatListCommands registers the three lists on the Sketch and 3D Sketch tabs. They apply
// in both environments: a 3D sketch carries the same per-entity overrides.
func sketchFormatListCommands() []*CommandDefinition {
	var cmds []*CommandDefinition
	for _, l := range []struct {
		kind    FormatListKind
		id      string
		name    string
		tooltip string
	}{
		{LineTypeList, "LineType", "Line Type", "Line Type — the dash pattern of the selected sketch geometry."},
		{ColorList, "Color", "Color", "Color — the colour of the selected sketch geometry."},
		{LineWeightList, "LineWeight", "Thickness", "Thickness — the plotted stroke width of the selected sketch geometry."},
	} {
		cmds = append(cmds,
			formatListCommand(l.kind, "Sketch."+l.id, l.name, l.tooltip, "Sketch", SketchEnvironment, inSketch),
			formatListCommand(l.kind, "Sketch3D."+l.id, l.name, l.tooltip, tab3DSketch, Sketch3DEnvironment, inSketch3D),
		)
	}
	return cmds
}

// formatListCommand builds one selection-list command for a tab and environment.
func formatListCommand(kind FormatListKind, id, name, tooltip, tab string, env Environment, enable func(*Session) bool) *CommandDefinition {
	return NewCommand(id, name, "Format", func(s *Session) error { return nil }).
		WithTab(tab).WithEnvironment(env).WithEnable(enable).
		WithButtonStyle(SelectionListButton).WithTooltip(tooltip).
		WithSelectionList(kind)
}
