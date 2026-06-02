// SPDX-License-Identifier: GPL-2.0-only

package app

import "strings"

// RegisterStandardCommands wires Inventor's standard ribbon for a session: the 3D Model
// tab (Create 2D Sketch, Extrude), the contextual Sketch tab (the full geometry-tool
// Create panel + Finish Sketch), and the View tab (Zoom All, Home). Sketch geometry
// tools are enabled only inside the sketch environment (s.InSketch), so the ribbon acts
// as a sketch editor when a sketch is open and as the part environment otherwise. The
// head and tests both call this, so the wiring is covered headlessly.
func RegisterStandardCommands(s *Session) error {
	for _, c := range standardCommands() {
		if err := s.Commands().Add(c); err != nil {
			return err
		}
	}
	return nil
}

// standardCommands returns every standard command definition in ribbon order.
func standardCommands() []*CommandDefinition {
	var cmds []*CommandDefinition
	cmds = append(cmds, modelTabCommands()...)
	cmds = append(cmds, sketchTabCommands()...)
	cmds = append(cmds, viewTabCommands()...)
	return cmds
}

// modelTabCommands are the 3D Model tab: starting a sketch and the solid features.
func modelTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Sketch.Create2D", "Create 2D Sketch", "Sketch", func(s *Session) error {
			// With a plane already selected, sketch on it immediately; otherwise start
			// the tool and let the user pick a plane in the 3D view or the browser.
			if s.SelectedWorkPlane() != nil {
				_, err := s.CreateSketchOnSelectedPlane()
				return err
			}
			s.StartTool(NewCreateSketchTool())
			return nil
		}).WithTab("3D Model").WithAlias("S").WithEnable(canCreateSketch).
			WithIcon("create-sketch").WithButtonStyle(LargeIconButton).
			WithTooltip("Create 2D Sketch — pick a work plane or planar face to sketch on."),
		NewCommand("Create.Extrude", "Extrude", "Create", func(s *Session) error {
			s.StartTool(NewExtrudeTool())
			return nil
		}).WithTab("3D Model").WithAlias("E").WithEnable(notInSketch).
			WithIcon("extrude").WithButtonStyle(LargeIconButton).
			WithTooltip("Extrude — add depth to a sketch profile to create or modify a solid."),
	}
}

// sketchTabCommands are the contextual Sketch tab: geometry creation, constraints,
// dimension, and Finish Sketch — all gated on being in the sketch environment.
func sketchTabCommands() []*CommandDefinition {
	cmds := createCommands()
	cmds = append(cmds, constrainCommands()...)
	cmds = append(cmds, NewCommand("Sketch.Dimension", "Dimension", "Dimension", func(s *Session) error {
		s.StartTool(newDimensionTool())
		return nil
	}).WithTab("Sketch").WithAlias("D").WithEnable(inSketch).
		WithIcon("dimension").WithButtonStyle(SmallIconButton).
		WithTooltip("Dimension — then pick points/a line/a circle/two lines to dimension."))
	return append(cmds, NewCommand("Sketch.Finish", "Finish Sketch", "Exit", func(s *Session) error {
		return s.FinishSketch()
	}).WithTab("Sketch").WithEnable(inSketch).
		WithIcon("finish-sketch").WithButtonStyle(LargeIconButton).
		WithTooltip("Finish Sketch — leave the sketch environment and update the part."))
}

// createCommands are the Sketch tab's Create panel — the geometry tools.
func createCommands() []*CommandDefinition {
	create := []struct {
		id, name, alias, tip string
		start                func() Tool
	}{
		{"Sketch.Line", "Line", "L", "Line — draw a line between two points.", func() Tool { return NewLineTool() }},
		{"Sketch.Rectangle", "Rectangle", "REC", "Rectangle — draw a two-corner rectangle.", func() Tool { return NewRectangleTool() }},
		{"Sketch.Circle", "Circle", "C", "Circle — draw a circle from its center and radius.", func() Tool { return NewCircleTool() }},
		{"Sketch.Arc", "Arc", "A", "Arc — draw a three-point arc.", func() Tool { return NewArcTool() }},
		{"Sketch.Spline", "Spline", "SPL", "Spline — draw an interpolated curve through fit points.", func() Tool { return NewSplineTool() }},
		{"Sketch.Ellipse", "Ellipse", "EL", "Ellipse — draw an ellipse from center and axes.", func() Tool { return NewEllipseTool() }},
		{"Sketch.Polygon", "Polygon", "POL", "Polygon — draw a regular inscribed polygon.", func() Tool { return NewPolygonTool(6) }},
		{"Sketch.Point", "Point", "PT", "Point — place sketch points.", func() Tool { return NewPointTool() }},
	}
	cmds := make([]*CommandDefinition, len(create))
	for i, c := range create {
		start := c.start
		cmds[i] = NewCommand(c.id, c.name, "Create", func(s *Session) error {
			s.StartTool(start())
			return nil
		}).WithTab("Sketch").WithAlias(c.alias).WithEnable(inSketch).WithTooltip(c.tip).
			WithIcon(strings.ToLower(c.name)).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// constrainCommands are the Sketch tab's Constrain panel — each starts an interactive
// constraint tool that the user then feeds geometry (Inventor's tool-first flow).
func constrainCommands() []*CommandDefinition {
	cmds := make([]*CommandDefinition, len(constraintToolDefs))
	for i, d := range constraintToolDefs {
		newTool := d.new
		cmds[i] = NewCommand(d.id, d.name, "Constrain", func(s *Session) error {
			s.StartTool(newTool())
			return nil
		}).WithTab("Sketch").WithEnable(inSketch).WithTooltip(d.tooltip).
			WithIcon(strings.ToLower(d.name)).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// viewTabCommands are the View tab navigation commands.
func viewTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("View.ZoomAll", "Zoom All", "Navigate", func(s *Session) error {
			s.FitView()
			return nil
		}).WithTab("View").WithIcon("zoom-all").WithButtonStyle(LargeIconButton).
			WithTooltip("Zoom All — fit the entire model in the viewport."),
		NewCommand("View.Home", "Home", "Navigate", func(s *Session) error {
			s.HomeView()
			return nil
		}).WithTab("View").WithIcon("home").WithButtonStyle(LargeIconButton).
			WithTooltip("Home View — the default isometric view, framed to fit."),
	}
}

// Enable predicates shared by the standard commands.
func inSketch(s *Session) bool        { return s.InSketch() }
func notInSketch(s *Session) bool     { return !s.InSketch() }
func canCreateSketch(s *Session) bool { return s.CanCreateSketch() }
