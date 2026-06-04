// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"

	"github.com/Oblikovati/oblikovati/renderer"
)

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
	cmds = append(cmds, getStartedCommands()...)
	cmds = append(cmds, modelTabCommands()...)
	cmds = append(cmds, workFeatureCommands()...)
	cmds = append(cmds, manageTabCommands()...)
	cmds = append(cmds, sketchTabCommands()...)
	cmds = append(cmds, viewTabCommands()...)
	return cmds
}

// getStartedCommands are the ZeroDoc ribbon's Get Started tab — shown when no document is
// open. New Part is the primary action; it creates a part, which switches the active ribbon
// to the Part ribbon (RibbonUI_Overview's per-document-type ribbons).
func getStartedCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("GetStarted.NewPart", "New Part", "Launch", func(s *Session) error {
			_, err := s.NewPart()
			return err
		}).WithTab("Get Started").WithRibbons(ZeroDocRibbon).
			WithIcon("new-part").WithButtonStyle(LargeIconButton).
			WithTooltip("New Part — create a part document and open the part environment."),
	}
}

// manageTabCommands are the Manage tab's Parameters panel: open the Parameters dialog
// (Inventor's Manage ▸ Parameters). Enabled whenever a part is active.
func manageTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Manage.Parameters", "Parameters", "Parameters", func(s *Session) error {
			s.OpenParameters()
			return nil
		}).WithTab("Manage").WithEnable(hasActivePart).
			WithIcon("parameters").WithButtonStyle(LargeIconButton).
			WithTooltip("Parameters — add, edit, and organize the model and user parameters that drive the part."),
	}
}

// hasActivePart reports whether the active document is a part (the Parameters dialog
// needs one to read and edit).
func hasActivePart(s *Session) bool {
	_, err := activePart(s)
	return err == nil
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
		NewCommand("Sketch.Create3D", "3D Sketch", "Sketch", func(s *Session) error {
			_, err := s.CreateSketch3D()
			return err
		}).WithTab("3D Model").WithEnable(canCreateSketch3D).
			WithIcon("create-sketch-3d").WithButtonStyle(LargeIconButton).
			WithTooltip("3D Sketch — create a non-planar sketch (sweep/loft path, helix)."),
	}
	cmds = append(cmds, sketch3DToolCommands()...)
	cmds = append(cmds, solidFeatureCommands()...)
	return append(cmds, modifyFeatureCommands()...)
}

// sketch3DToolCommands are the contextual 3D-sketch tools, enabled only while a 3D sketch
// is being edited (M22-F12): the geometry tools plus Finish.
func sketch3DToolCommands() []*CommandDefinition {
	finish := NewCommand("Sketch3D.Finish", "Finish 3D Sketch", "Sketch3D", func(s *Session) error {
		return s.FinishSketch3D()
	}).WithTab("3D Sketch").WithEnable(inSketch3D).WithIcon("finish-sketch").
		WithTooltip("Finish the 3D sketch and return to the model.")
	return append(sketch3DDrawCommands(), finish)
}

// sketch3DDrawCommands are the 3D-sketch geometry-placement tools (line/point/circle/arc/
// helix), each starting its interactive tool.
func sketch3DDrawCommands() []*CommandDefinition {
	return []*CommandDefinition{
		sketch3DToolCommand("Sketch3D.Line", "3D Line", "line",
			"3D Line — place points in model space to build a polyline rail.",
			func() Tool { return NewLine3DTool() }),
		sketch3DToolCommand("Sketch3D.Point", "3D Point", "point",
			"3D Point — place a point in model space.",
			func() Tool { return NewPoint3DTool() }),
		sketch3DToolCommand("Sketch3D.Circle", "3D Circle", "circle",
			"3D Circle — a circle from a center, plane axis and radius.",
			func() Tool { return NewCircle3DTool() }),
		sketch3DToolCommand("Sketch3D.Arc", "3D Arc", "arc",
			"3D Arc — an arc through center, start and end points.",
			func() Tool { return NewArc3DTool() }),
		sketch3DToolCommand("Sketch3D.Helix", "Helical Curve", "helix",
			"Helical Curve — a spring/thread path from radius, pitch and turns.",
			func() Tool { return NewHelix3DTool() }),
		sketch3DToolCommand("Sketch3D.Include", "Include Geometry", "include",
			"Include Geometry — pick part edges/vertices to reference in the 3D sketch.",
			func() Tool { return NewIncludeGeometry3DTool() }),
		sketch3DToolCommand("Sketch3D.SurfaceCurve", "Surface Curve", "surface-curve",
			"Surface Curve — derive an intersection (2 faces) or silhouette (1 face) curve.",
			func() Tool { return NewSurfaceCurve3DTool() }),
	}
}

// sketch3DToolCommand builds a contextual 3D-sketch command that starts the tool from new.
func sketch3DToolCommand(id, name, icon, tip string, newTool func() Tool) *CommandDefinition {
	return NewCommand(id, name, "Sketch3D", func(s *Session) error {
		s.StartTool(newTool())
		return nil
	}).WithTab("3D Sketch").WithEnable(inSketch3D).WithIcon(icon).WithTooltip(tip)
}

// modifyFeatureCommands are the 3D Model tab's "Modify" panel: the material-cutting
// features (hole/chamfer), the local face operations (shell/offset/draft/delete/replace),
// and surface→solid (thicken).
func modifyFeatureCommands() []*CommandDefinition {
	cmds := append(cutFeatureCommands(), localFaceCommands()...)
	return append(cmds, surfaceSolidCommands()...)
}

// surfaceSolidCommands are the Modify features that turn a surface body into a solid.
func surfaceSolidCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Modify.Thicken", "Thicken", "Modify", func(s *Session) error {
			s.StartTool(NewThickenTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("thicken").WithButtonStyle(LargeIconButton).
			WithTooltip("Thicken — turn the active surface body into a solid of a wall thickness."),
	}
}

// cutFeatureCommands are the Modify features that remove material against picked topology.
func cutFeatureCommands() []*CommandDefinition {
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
		NewCommand("Modify.Fillet", "Fillet", "Modify", func(s *Session) error {
			s.StartTool(NewFilletTool())
			return nil
		}).WithTab("3D Model").WithAlias("F").WithEnable(notInSketch).
			WithIcon("fillet").WithButtonStyle(LargeIconButton).
			WithTooltip("Fillet — round selected convex edges with a rolling-ball radius."),
	}
}

// localFaceCommands are the F04 local face operations — the metric edits (shell, offset
// face, draft) plus the topology edits (delete face, replace face).
func localFaceCommands() []*CommandDefinition {
	return append(faceMetricCommands(), faceTopologyCommands()...)
}

// faceMetricCommands move faces by a distance/angle (shell, offset face, draft).
func faceMetricCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Modify.Shell", "Shell", "Modify", func(s *Session) error {
			s.StartTool(NewShellTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("shell").WithButtonStyle(LargeIconButton).
			WithTooltip("Shell — hollow the solid to a wall thickness, removing the selected faces."),
		NewCommand("Modify.FaceOffset", "Offset Face", "Modify", func(s *Session) error {
			s.StartTool(NewFaceOffsetTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("face-offset").WithButtonStyle(LargeIconButton).
			WithTooltip("Offset Face — move selected faces along their normal, retrimming neighbours."),
		NewCommand("Modify.Draft", "Draft", "Modify", func(s *Session) error {
			s.StartTool(NewDraftTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("draft").WithButtonStyle(LargeIconButton).
			WithTooltip("Draft — taper selected faces by an angle about the pull direction."),
	}
}

// faceTopologyCommands change which faces exist (delete face + heal, replace face).
func faceTopologyCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Modify.DeleteFace", "Delete Face", "Modify", func(s *Session) error {
			s.StartTool(NewDeleteFaceTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("delete-face").WithButtonStyle(LargeIconButton).
			WithTooltip("Delete Face — remove selected faces and heal the openings."),
		NewCommand("Modify.ReplaceFace", "Replace Face", "Modify", func(s *Session) error {
			s.StartTool(NewReplaceFaceTool())
			return nil
		}).WithTab("3D Model").WithEnable(notInSketch).
			WithIcon("replace-face").WithButtonStyle(LargeIconButton).
			WithTooltip("Replace Face — move selected faces onto a target face's plane."),
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
	cmds = append(cmds, sketchModifyCommands()...)
	cmds = append(cmds, constrainCommands()...)
	cmds = append(cmds, NewCommand("Sketch.Project", "Project Geometry", "Draw", func(s *Session) error {
		s.StartTool(NewProjectGeometryTool())
		return nil
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).
		WithIcon("project-geometry").WithButtonStyle(SmallIconButton).
		WithTooltip("Project Geometry — pick part edges/vertices to reference onto the sketch plane."))
	cmds = append(cmds, NewCommand("Sketch.Dimension", "Dimension", "Dimension", func(s *Session) error {
		s.StartTool(newDimensionTool())
		return nil
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithAlias("D").WithEnable(inSketch).
		WithIcon("dimension").WithButtonStyle(SmallIconButton).
		WithTooltip("Dimension — then pick points/a line/a circle/two lines to dimension."))
	return append(cmds, NewCommand("Sketch.Finish", "Finish Sketch", "Exit", func(s *Session) error {
		return s.FinishSketch()
	}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).
		WithIcon("finish-sketch").WithButtonStyle(LargeIconButton).
		WithTooltip("Finish Sketch — leave the sketch environment and update the part."))
}

// sketchModifyCommands are the Sketch tab's Modify panel — the operations that edit
// existing sketch geometry (offset, mirror, sketch fillet). Each starts an interactive
// tool that the user feeds geometry, mirroring the constraint tools' flow.
func sketchModifyCommands() []*CommandDefinition {
	mods := []struct {
		id, name, alias, tip string
		start                func() Tool
	}{
		{"Sketch.Offset", "Offset", "O", "Offset — pick a curve to offset by a distance.", func() Tool { return NewSketchOffsetTool(0.5) }},
		{"Sketch.Mirror", "Mirror", "MI", "Mirror — pick geometry, then a mirror line.", func() Tool { return NewSketchMirrorTool() }},
		{"Sketch.Fillet", "Sketch Fillet", "FF", "Sketch Fillet — pick two lines to round their corner.", func() Tool { return NewSketchFilletTool(0.5) }},
	}
	cmds := make([]*CommandDefinition, len(mods))
	for i, m := range mods {
		start := m.start
		cmds[i] = NewCommand(m.id, m.name, "Modify", func(s *Session) error {
			s.StartTool(start())
			return nil
		}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithAlias(m.alias).WithEnable(inSketch).
			WithTooltip(m.tip).WithIcon(strings.ToLower(m.name)).WithButtonStyle(SmallIconButton)
	}
	return cmds
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
		}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithAlias(c.alias).WithEnable(inSketch).WithTooltip(c.tip).
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
		}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).WithTooltip(d.tooltip).
			WithIcon(strings.ToLower(d.name)).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// viewTabCommands are the View tab commands: navigation, the Visual Style presets, and the
// lighting/environment/shadow controls (M16/F03).
func viewTabCommands() []*CommandDefinition {
	cmds := append(viewNavigateCommands(), visualStyleCommands()...)
	return append(cmds, lightingViewCommands()...)
}

// viewNavigateCommands are the View tab's Navigate panel.
func viewNavigateCommands() []*CommandDefinition {
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

// visualStyleSpec maps a renderer VisualStyle to its View-tab command id and tooltip. The ids
// are stable (tests + aliases depend on "View.Shaded" etc.); the styles and labels come from
// the renderer's gallery so the ribbon mirrors Inventor's full DisplayModeEnum set.
type visualStyleSpec struct {
	style   renderer.VisualStyle
	id      string
	tooltip string
}

var visualStyleSpecs = []visualStyleSpec{
	{renderer.Realistic, "View.Realistic", "Realistic — physically based (PBR) shading."},
	{renderer.Shaded, "View.Shaded", "Shaded — lit faces, no edges."},
	{renderer.ShadedWithEdges, "View.ShadedWithEdges", "Shaded with Edges — lit faces with the edge wireframe."},
	{renderer.ShadedWithHiddenEdges, "View.ShadedWithHiddenEdges", "Shaded with Hidden Edges — visible edges solid, occluded edges dashed."},
	{renderer.Wireframe, "View.Wireframe", "Wireframe — every edge, no shaded faces."},
	{renderer.WireframeVisibleOnly, "View.WireframeVisibleOnly", "Wireframe with Visible Edges Only — occluded edges removed."},
	{renderer.WireframeWithHiddenEdges, "View.WireframeWithHiddenEdges", "Wireframe with Hidden Edges — visible edges solid, occluded edges dashed."},
	{renderer.Monochrome, "View.Monochrome", "Monochrome — desaturated, posterized illustration with outlines."},
	{renderer.Watercolor, "View.Watercolor", "Watercolor — soft pigment washes on paper."},
	{renderer.Illustration, "View.Illustration", "Illustration — flat/cel shading with crisp outlines."},
	{renderer.TechnicalIllustration, "View.TechnicalIllustration", "Technical Illustration — Gooch cool-warm shading with emphasized edges."},
}

// visualStyleCommands are the View tab's Visual Style panel: every Inventor display mode as a
// mutually-exclusive option of one selection box (ComboControl). Selecting one sets the
// session style; the active one drives the box's current selection.
func visualStyleCommands() []*CommandDefinition {
	cmds := make([]*CommandDefinition, 0, len(visualStyleSpecs))
	for _, sp := range visualStyleSpecs {
		cmds = append(cmds, NewCommand(sp.id, sp.style.String(), "Visual Style", func(s *Session) error {
			s.SetVisualStyle(sp.style)
			return nil
		}).WithTab("View").WithKind(ComboControl).WithTooltip(sp.tooltip).
			WithActive(func(s *Session) bool { return s.VisualStyle() == sp.style }))
	}
	return cmds
}

// Enable predicates shared by the standard commands.
func inSketch(s *Session) bool          { return s.InSketch() }
func notInSketch(s *Session) bool       { return !s.InSketch() }
func canCreateSketch(s *Session) bool   { return s.CanCreateSketch() }
func canCreateSketch3D(s *Session) bool { return s.CanCreateSketch3D() }
func inSketch3D(s *Session) bool        { return s.InSketch3D() }
