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
	cmds = append(cmds, workFeatureCommands()...)
	cmds = append(cmds, sketchTabCommands()...)
	cmds = append(cmds, viewTabCommands()...)
	return cmds
}

// workFeatureCommands are the 3D Model tab's Work Features panel: the datum-plane
// constructors. Each button is always live in the part environment (Inventor's Work Plane
// behavior) — click it and, if the right geometry is already selected it builds the datum
// at once, otherwise it starts a guided pick that prompts for the inputs and commits when
// they are gathered. So a click is never inert, whether or not anything was pre-selected.
func workFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("WorkPlane.Offset", "Offset Plane", "Work Features", func(s *Session) error {
			s.StartTool(NewOffsetWorkPlaneTool()) // always opens the distance dialog (Inventor's flow)
			return nil
		}).WithTab("3D Model").WithEnable(canStartWorkPlane).
			WithIcon("work-plane-offset").WithButtonStyle(LargeIconButton).
			WithTooltip("Offset Plane — a work plane parallel to a plane, offset by a distance. Pick a plane, then enter the offset."),
		NewCommand("WorkPlane.Midplane", "Midplane", "Work Features", startWorkPlane(newMidplaneWorkPlaneTool)).
			WithTab("3D Model").WithEnable(canStartWorkPlane).
			WithIcon("work-plane-midplane").WithButtonStyle(SmallIconButton).
			WithTooltip("Midplane — a work plane bisecting two planes. Pick two planes when prompted."),
		NewCommand("WorkPlane.ThreePoints", "Three Points", "Work Features", startWorkPlane(newThreePointWorkPlaneTool)).
			WithTab("3D Model").WithEnable(canStartWorkPlane).
			WithIcon("work-plane-3pt").WithButtonStyle(SmallIconButton).
			WithTooltip("Three Points — a work plane through three points or model vertices. Pick three when prompted."),
		NewCommand("WorkPlane.Tangent", "Tangent to Face", "Work Features", startWorkPlane(newTangentWorkPlaneTool)).
			WithTab("3D Model").WithEnable(canStartWorkPlane).
			WithIcon("work-plane-tangent").WithButtonStyle(SmallIconButton).
			WithTooltip("Tangent to Face — a work plane parallel to a plane and tangent to a cylindrical/spherical face. Pick a plane then a face."),
		NewCommand("WorkPlane.NormalToAxis", "Normal to Axis", "Work Features", startWorkPlane(newNormalToAxisWorkPlaneTool)).
			WithTab("3D Model").WithEnable(canStartWorkPlane).
			WithIcon("work-plane-normal").WithButtonStyle(SmallIconButton).
			WithTooltip("Normal to Axis — a work plane through a point, normal to an axis. Pick an axis then a point."),
	}
}

// modelTabCommands are the 3D Model tab: starting a sketch and the solid features.
func modelTabCommands() []*CommandDefinition {
	cmds := []*CommandDefinition{
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
	}
	cmds = append(cmds, solidFeatureCommands()...)
	return append(cmds, modifyFeatureCommands()...)
}

// modifyFeatureCommands are the 3D Model tab's "Modify" panel — features that cut or
// alter existing material (hole, …).
func modifyFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Modify.Hole", "Hole", "Modify", func(s *Session) error {
			s.StartTool(NewHoleTool())
			return nil
		}).WithTab("3D Model").WithAlias("H").WithEnable(notInSketch).
			WithIcon("hole").WithButtonStyle(LargeIconButton).
			WithTooltip("Hole — drill a cylindrical hole into a planar face of the solid."),
		NewCommand("Modify.Chamfer", "Chamfer", "Modify", func(s *Session) error {
			s.StartTool(NewChamferTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("chamfer").WithButtonStyle(LargeIconButton).
			WithTooltip("Chamfer — bevel selected edges by a setback distance."),
	}
}

// solidFeatureCommands are the 3D Model tab's "Create" panel — the sketched solid
// features, each launching its interactive tool. Split into sketched-profile features
// (extrude/revolve) and swept features (sweep/loft/coil) to keep each builder small.
func solidFeatureCommands() []*CommandDefinition {
	return append(profileSolidCommands(), sweptSolidCommands()...)
}

// profileSolidCommands are the single-profile solid features (extrude, revolve).
func profileSolidCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Create.Extrude", "Extrude", "Create", func(s *Session) error {
			s.StartTool(NewExtrudeTool())
			return nil
		}).WithTab("3D Model").WithAlias("E").WithEnable(notInSketch).
			WithIcon("extrude").WithButtonStyle(LargeIconButton).
			WithTooltip("Extrude — add depth to a sketch profile to create or modify a solid."),
		NewCommand("Create.Revolve", "Revolve", "Create", func(s *Session) error {
			s.StartTool(NewRevolveTool())
			return nil
		}).WithTab("3D Model").WithAlias("R").WithEnable(notInSketch).
			WithIcon("revolve").WithButtonStyle(LargeIconButton).
			WithTooltip("Revolve — spin a sketch profile about an axis to create or modify a solid."),
	}
}

// sweptSolidCommands are the swept/blended solid features (sweep, loft, coil).
func sweptSolidCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Create.Sweep", "Sweep", "Create", func(s *Session) error {
			s.StartTool(NewSweepTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("sweep").WithButtonStyle(LargeIconButton).
			WithTooltip("Sweep — run a sketch profile along a path to create or modify a solid."),
		NewCommand("Create.Loft", "Loft", "Create", func(s *Session) error {
			s.StartTool(NewLoftTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("loft").WithButtonStyle(LargeIconButton).
			WithTooltip("Loft — blend two or more sketch sections into a solid."),
		NewCommand("Create.Coil", "Coil", "Create", func(s *Session) error {
			s.StartTool(NewCoilTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("coil").WithButtonStyle(LargeIconButton).
			WithTooltip("Coil — sweep a sketch profile along a helix to create or modify a solid."),
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
