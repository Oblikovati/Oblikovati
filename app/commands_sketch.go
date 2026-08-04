// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"strings"
)

// The contextual 2D Sketch tab, built from the command registry to mirror the reference UI's
// ribbon (see architecture/mapping/inventor-ribbon-structure.md): panels Create, Modify,
// Pattern, Constrain, Exit. Split out of commands_standard.go (which exceeded the file-size
// limit) so the sketch ribbon lives in one place.

// sketchTabCommands assembles the Sketch tab in panel order.
func sketchTabCommands() []*CommandDefinition {
	cmds := createCommands()
	cmds = append(cmds, sketchModifyCommands()...)
	cmds = append(cmds, sketchPatternCommands()...)
	cmds = append(cmds, sketchFormatCommands()...)
	// Constrain panel (reference order): Dimension, Auto Dimension, then the constraint
	// tools — there is no separate "Dimension" panel in the reference Sketch tab.
	cmds = append(cmds, NewCommand("Sketch.Dimension", "Dimension", "Constrain", func(s *Session) error {
		s.StartTool(NewDimensionTool())
		return nil
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithAlias("D").WithDefaultChord("Shift+D").WithEnable(inSketch).
		WithIcon("dimension").WithButtonStyle(SmallIconButton).
		WithTooltip("Dimension — pick points/a line/a circle/two lines, then click to place it."))
	cmds = append(cmds, NewCommand("Sketch.AutoDimension", "Auto Dimension", "Constrain", func(s *Session) error {
		sk := s.ActiveSketch()
		if sk == nil {
			return errors.New("auto dimension: no active sketch")
		}
		sk.AutoDimension()
		return nil
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).
		WithIcon("auto-dimension").WithButtonStyle(SmallIconButton).
		WithTooltip("Auto Dimension — fully constrain the sketch with dimensions and grounds."))
	cmds = append(cmds, constrainCommands()...)
	// Project Geometry lives in Create as a large button (the canonical ribbon has no
	// "Draw" panel — see architecture/mapping/inventor-ribbon-structure.md). Project Scan Point
	// rides under it as a split-button variant (the two are both "project onto the sketch plane").
	projectScanPoint := NewCommand("Sketch.ProjectScanPoint", "Project Scan Point", "Create", func(s *Session) error {
		_, err := s.CreateSketchPointAtSelectedCloudPoint()
		return err
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(canSketchPointAtCloudPoint).
		WithIcon("sketch-point-scan").WithButtonStyle(SmallIconButton).
		WithTooltip("Project Scan Point — place a sketch point on the selected scan point, projected onto the sketch plane.")
	cmds = append(cmds, NewCommand("Sketch.Project", "Project Geometry", "Create", func(s *Session) error {
		s.StartTool(NewProjectGeometryTool())
		return nil
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).
		WithIcon("project-geometry").WithButtonStyle(LargeIconButton).
		WithTooltip("Project Geometry — pick part edges/vertices to reference onto the sketch plane.").
		WithVariants(projectScanPoint))
	cmds = append(cmds, NewCommand("Sketch.Finish", "Finish Sketch", "Exit", func(s *Session) error {
		return s.FinishSketch()
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).
		WithIcon("finish-sketch").WithButtonStyle(LargeIconButton).
		WithTooltip("Finish Sketch — leave the sketch environment and update the part."))
	// The contextual Sketch tab serves a part AND an assembly sketch — the environment is now
	// content-agnostic (#766) — so its tools appear on both ribbons.
	for _, c := range cmds {
		c.WithRibbons(PartRibbon, AssemblyRibbon)
	}
	return cmds
}

// sketchToolEntry is one tool-launching command (id/label/alias/tooltip + factory).
// variants, when present, become the entry's split-button dropdown (the reference UI's variant
// flyout): sibling tools reachable from the head's arrow, not their own panel buttons.
// large marks the panel's headline tools (Line, Circle, …) that render as large
// captioned buttons; the rest stack as small labeled rows.
type sketchToolEntry struct {
	id, name, alias, tip string
	// chord is a shipped default keyboard shortcut ("Shift+L" for Line, …). Bare a–z/0–9 are
	// reserved for command-window typing (#1751), so a headline tool's shortcut carries Shift.
	chord string
	// icon overrides the lowercased-name asset key for multi-word names
	// (asset filenames are kebab-case).
	icon     string
	start    func() Tool
	variants []sketchToolEntry
	large    bool
}

// newToolCommand builds one tool-launching command (no alias/icon for variants, which only
// appear in a dropdown). The factory is captured per-call so each command starts its own tool.
func newToolCommand(panel string, e sketchToolEntry) *CommandDefinition {
	start := e.start
	return NewCommand(e.id, e.name, panel, func(s *Session) error {
		s.StartTool(start())
		return nil
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).WithTooltip(e.tip)
}

// buildToolCommands turns a table of tool entries into ribbon commands on the given panel,
// attaching each entry's variants as its split-button dropdown.
func buildToolCommands(panel string, entries []sketchToolEntry) []*CommandDefinition {
	cmds := make([]*CommandDefinition, len(entries))
	for i, e := range entries {
		style := SmallIconButton
		if e.large {
			style = LargeIconButton
		}
		key := e.icon
		if key == "" {
			key = strings.ToLower(e.name)
		}
		cmd := newToolCommand(panel, e).WithAlias(e.alias).
			WithIcon(key).WithButtonStyle(style)
		if e.chord != "" {
			cmd.WithDefaultChord(e.chord) // shipped Shift+mnemonic shortcut (#1751 S3)
		}
		if len(e.variants) > 0 {
			variants := make([]*CommandDefinition, len(e.variants))
			for j, v := range e.variants {
				variants[j] = newToolCommand(panel, v)
			}
			cmd.WithVariants(variants...)
		}
		cmds[i] = cmd
	}
	return cmds
}

// sketchModifyCommands are the Sketch tab's Modify panel (reference order: Move, Copy,
// Rotate, Scale, Trim, Extend, Split, Offset; Stretch is a follow-up). Each starts an
// interactive tool the user feeds geometry.
func sketchModifyCommands() []*CommandDefinition {
	return buildToolCommands("Modify", []sketchToolEntry{
		{id: "Sketch.Move", name: "Move", tip: "Move — select geometry, then set the move vector.", start: func() Tool { return NewSketchMoveTool() }},
		{id: "Sketch.Copy", name: "Copy", tip: "Copy — select geometry, then set the copy offset.", start: func() Tool { return NewSketchCopyTool() }},
		{id: "Sketch.Rotate", name: "Rotate", tip: "Rotate — select geometry, then set the center and angle.", start: func() Tool { return NewSketchRotateTool() }},
		{id: "Sketch.Scale", name: "Scale", tip: "Scale — select geometry, then set the center and factor.", start: func() Tool { return NewSketchScaleTool() }},
		{id: "Sketch.Stretch", name: "Stretch", tip: "Stretch — pick vertices to move, then set the vector.", start: func() Tool { return NewSketchStretchTool() }},
		{id: "Sketch.Trim", name: "Trim", alias: "X", tip: "Trim — pick the curve segment to remove up to its crossings.", start: func() Tool { return NewSketchTrimTool() }},
		{id: "Sketch.Extend", name: "Extend", alias: "EX", tip: "Extend — pick near a line end to lengthen it to the next crossing.", start: func() Tool { return NewSketchExtendTool() }},
		{id: "Sketch.Split", name: "Split", alias: "SX", tip: "Split — pick the point to split a curve into two.", start: func() Tool { return NewSketchSplitTool() }},
		{id: "Sketch.Offset", name: "Offset", alias: "O", tip: "Offset — pick a curve to offset by a distance.", start: func() Tool { return NewSketchOffsetTool(0.5) }},
	})
}

// sketchPatternCommands are the Sketch tab's Pattern panel (Rectangular, Circular, Mirror).
func sketchPatternCommands() []*CommandDefinition {
	return buildToolCommands("Pattern", []sketchToolEntry{
		{id: "Sketch.RectangularPattern", name: "Rectangular", tip: "Rectangular Pattern — select geometry, then set directions and counts.", start: func() Tool { return NewSketchRectPatternTool() }},
		{id: "Sketch.CircularPattern", name: "Circular", tip: "Circular Pattern — select geometry, then set center, count and angle.", start: func() Tool { return NewSketchCircPatternTool() }},
		{id: "Sketch.Mirror", name: "Mirror", alias: "MI", tip: "Mirror — pick geometry, then a mirror line.", start: func() Tool { return NewSketchMirrorTool() }},
	})
}

// createCommands are the Sketch tab's Create panel — the geometry tools (plus Fillet,
// which the reference UI places in Create, not Modify).
func createCommands() []*CommandDefinition {
	return buildToolCommands("Create", []sketchToolEntry{
		{id: "Sketch.Line", name: "Line", alias: "L", chord: "Shift+L", large: true, tip: "Line — draw a connected chain of lines; Enter or Escape finishes.", start: func() Tool { return NewLineTool() }},
		{id: "Sketch.Rectangle", name: "Rectangle", alias: "REC", large: true, tip: "Rectangle — draw a two-corner rectangle.", start: func() Tool { return NewRectangleTool() }, variants: []sketchToolEntry{
			{id: "Sketch.Rectangle.ThreePoint", name: "Three Point Rectangle", tip: "Three Point Rectangle — base edge then width.", start: func() Tool { return NewThreePointRectangleTool() }},
			{id: "Sketch.Rectangle.Center", name: "Two Point Center Rectangle", tip: "Two Point Center Rectangle — center then a corner.", start: func() Tool { return NewCenterRectangleTool() }},
		}},
		{id: "Sketch.Circle", name: "Circle", alias: "C", chord: "Shift+C", large: true, tip: "Circle — draw a circle from its center and radius.", start: func() Tool { return NewCircleTool() }, variants: []sketchToolEntry{
			{id: "Sketch.Circle.ThreePoint", name: "Three Point Circle", tip: "Three Point Circle — the circle through three points.", start: func() Tool { return NewThreePointCircleTool() }},
		}},
		{id: "Sketch.Arc", name: "Arc", alias: "A", chord: "Shift+A", large: true, tip: "Arc — draw a three-point arc: start, end, then a point on the arc.", start: func() Tool { return NewArcTool() }, variants: []sketchToolEntry{
			{id: "Sketch.Arc.CenterPoint", name: "Center Point Arc", tip: "Center Point Arc — center, start, then end.", start: func() Tool { return NewCenterPointArcTool() }},
		}},
		{id: "Sketch.Slot", name: "Slot", tip: "Slot — click two centre points for a straight slot.", start: func() Tool { return NewSketchSlotTool(1) }, variants: []sketchToolEntry{
			{id: "Sketch.Slot.CenterArc", name: "Center Point Arc Slot", tip: "Center Point Arc Slot — center, start, then end.", start: func() Tool { return NewCenterPointArcSlotTool(1) }},
			{id: "Sketch.Slot.ThreePointArc", name: "Three Point Arc Slot", tip: "Three Point Arc Slot — start, end, then a point on the arc.", start: func() Tool { return NewThreePointArcSlotTool(1) }},
		}},
		{id: "Sketch.Spline", name: "Spline", alias: "SPL", tip: "Spline — draw an interpolated curve through fit points.", start: func() Tool { return NewSplineTool() }, variants: []sketchToolEntry{
			{id: "Sketch.Spline.ControlVertex", name: "Control Vertex Spline", tip: "Control Vertex Spline — draw a curve from its control polygon.", start: func() Tool { return NewControlVertexSplineTool() }},
		}},
		{id: "Sketch.Ellipse", name: "Ellipse", alias: "EL", tip: "Ellipse — draw an ellipse from center and axes.", start: func() Tool { return NewEllipseTool() }},
		{id: "Sketch.Polygon", name: "Polygon", alias: "POL", tip: "Polygon — draw a regular inscribed polygon.", start: func() Tool { return NewPolygonTool(6) }},
		{id: "Sketch.Fillet", name: "Fillet", alias: "FF", tip: "Fillet — pick two lines to round their corner.", start: func() Tool { return NewSketchFilletTool(0.5) }},
		{id: "Sketch.Chamfer", name: "Chamfer", tip: "Chamfer — pick two lines to bevel their corner.", start: func() Tool { return NewSketchChamferTool(0.5) }},
		{id: "Sketch.Text", name: "Text", tip: "Text — click an anchor, then type the text.", start: func() Tool { return NewSketchTextTool() }},
		{id: "Sketch.Point", name: "Point", alias: "PT", tip: "Point — place sketch points.", start: func() Tool { return NewPointTool() }},
		{id: "Sketch.CreateBlock", name: "Create Block", icon: "create-block", tip: "Create Block — select geometry, then name the reusable block.", start: func() Tool { return NewSketchCreateBlockTool() }, variants: []sketchToolEntry{
			{id: "Sketch.PlaceBlock", name: "Place Block", tip: "Place Block — choose a block, then click its insertion point.", start: func() Tool { return NewSketchPlaceBlockTool("") }},
		}},
	})
}

// constrainCommands are the Sketch tab's Constrain panel — each starts an interactive
// constraint tool the user then feeds geometry (the reference UI's tool-first flow).
func constrainCommands() []*CommandDefinition {
	cmds := make([]*CommandDefinition, len(constraintToolDefs))
	for i, d := range constraintToolDefs {
		newTool := d.new
		cmds[i] = NewCommand(d.id, d.name, "Constrain", func(s *Session) error {
			s.StartTool(newTool())
			return nil
		}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).WithTooltip(d.tooltip).
			WithIcon(strings.ToLower(d.name)).WithButtonStyle(CompactIconButton)
	}
	return cmds
}
