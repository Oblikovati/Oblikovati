// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/renderer"
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
		// Variants are flyout-only (skipped by BuildRibbon), but must be registered so a
		// dropdown selection can be dispatched by id through Session.Execute.
		for _, v := range c.Variants() {
			if err := s.Commands().Add(v); err != nil {
				return err
			}
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
	cmds = append(cmds, assemblyTabCommands()...)
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

// manageTabCommands are the Manage tab: the Parameters panel (Inventor's Manage ▸
// Parameters, needs an active part) and the Scripts panel (the Script Console — our
// equivalent of Manage ▸ iLogic, ADR-0028).
func manageTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Manage.Parameters", "Parameters", "Parameters", func(s *Session) error {
			s.OpenParameters()
			return nil
		}).WithTab("Manage").WithEnable(hasActivePart).
			WithIcon("parameters").WithButtonStyle(LargeIconButton).
			WithTooltip("Parameters — add, edit, and organize the model and user parameters that drive the part."),
		scriptConsoleCommand(),
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
		NewCommand("WorkPlane.Offset", "Offset Plane", panelWorkFeatures, func(s *Session) error {
			s.StartTool(NewOffsetWorkPlaneTool()) // always opens the distance dialog (Inventor's flow)
			return nil
		}).WithTab(tab3DModel).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-offset").WithButtonStyle(LargeIconButton).
			WithTooltip("Offset Plane — a work plane parallel to a plane, offset by a distance. Pick a plane, then enter the offset."),
		NewCommand("WorkPlane.Midplane", "Midplane", panelWorkFeatures, startWorkPlane(newMidplaneWorkPlaneTool)).
			WithTab(tab3DModel).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-midplane").WithButtonStyle(SmallIconButton).
			WithTooltip("Midplane — a work plane bisecting two planes. Pick two planes when prompted."),
		NewCommand("WorkPlane.ThreePoints", "Three Points", panelWorkFeatures, startWorkPlane(newThreePointWorkPlaneTool)).
			WithTab(tab3DModel).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-3pt").WithButtonStyle(SmallIconButton).
			WithTooltip("Three Points — a work plane through three points or model vertices. Pick three when prompted."),
		NewCommand("WorkPlane.Tangent", "Tangent to Face", panelWorkFeatures, startWorkPlane(newTangentWorkPlaneTool)).
			WithTab(tab3DModel).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-tangent").WithButtonStyle(SmallIconButton).
			WithTooltip("Tangent to Face — a work plane parallel to a plane and tangent to a cylindrical/spherical face. Pick a plane then a face."),
		NewCommand("WorkPlane.NormalToAxis", "Normal to Axis", panelWorkFeatures, startWorkPlane(newNormalToAxisWorkPlaneTool)).
			WithTab(tab3DModel).WithEnable(canStartWorkPlane).
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
		}).WithTab(tab3DModel).WithAlias("S").WithEnable(canCreateSketch).
			WithIcon("create-sketch").WithButtonStyle(LargeIconButton).
			WithTooltip("Create 2D Sketch — pick a work plane or planar face to sketch on."),
		NewCommand("Sketch.Create3D", "3D Sketch", "Sketch", func(s *Session) error {
			_, err := s.CreateSketch3D()
			return err
		}).WithTab(tab3DModel).WithEnable(canCreateSketch3D).
			WithIcon("create-sketch-3d").WithButtonStyle(LargeIconButton).
			WithTooltip("3D Sketch — create a non-planar sketch (sweep/loft path, helix)."),
	}
	cmds = append(cmds, sketch3DToolCommands()...)
	cmds = append(cmds, solidFeatureCommands()...)
	cmds = append(cmds, modifyFeatureCommands()...)
	cmds = append(cmds, patternFeatureCommands()...)
	cmds = append(cmds, surfaceFeatureCommands()...)
	cmds = append(cmds, freeformFeatureCommands()...)
	cmds = append(cmds, moldFeatureCommands()...)
	return append(cmds, meshFeatureCommands()...)
}

// meshFeatureCommands are the 3D Model tab's Mesh panel: place an STL as mesh reference
// geometry (M10-F04, #700). The command arms the head's file dialog; the import itself is
// Session.ImportMeshFile.
func meshFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Mesh.Place", "Place Mesh", "Mesh", func(s *Session) error {
			s.RequestImportMesh()
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("mesh-place").WithButtonStyle(LargeIconButton).
			WithTooltip("Place Mesh — load an ASCII STL as selectable mesh reference geometry."),
	}
}

// moldFeatureCommands are the 3D Model tab's Mold panel: the core/cavity tooling split
// (M10-F04, #701).
func moldFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Mold.CoreCavity", "Core/Cavity", "Mold", func(s *Session) error {
			s.StartTool(NewCoreCavityTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("core-cavity").WithButtonStyle(LargeIconButton).
			WithTooltip("Core/Cavity — split the tooling block at a parting plane into core and cavity solids."),
	}
}

// freeformFeatureCommands are the 3D Model tab's Freeform panel: the sub-D primitives
// (M10-F03, #698). Cage editing on the placed feature is the freeform.* wire surface.
func freeformFeatureCommands() []*CommandDefinition {
	prims := []struct {
		id, name, icon, tip string
		start               func() Tool
	}{
		{"Freeform.Box", "Box", "freeform-box", "Freeform Box — place a sub-D box cage and smooth it by subdivision level.", func() Tool { return NewFreeformBoxTool() }},
		{"Freeform.Plane", "Plane", "freeform-plane", "Freeform Plane — place an open sub-D plane cage (a surface body).", func() Tool { return NewFreeformPlaneTool() }},
		{"Freeform.QuadBall", "Quad Ball", "freeform-quadball", "Freeform Quad Ball — place a closed sphere-like sub-D cage.", func() Tool { return NewFreeformQuadBallTool() }},
	}
	cmds := make([]*CommandDefinition, len(prims))
	for i, d := range prims {
		start := d.start
		cmds[i] = NewCommand(d.id, d.name, "Freeform", func(s *Session) error {
			s.StartTool(start())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).WithTooltip(d.tip).
			WithIcon(d.icon).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// surfaceFeatureCommands are the 3D Model tab's Surface panel (canonical: Patch, Stitch, Sculpt,
// Extend, Trim, Rule Fillet). Patch fills a closed sketch region with a surface.
func surfaceFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Surface.Patch", "Patch", "Surface", func(s *Session) error {
			s.StartTool(NewPatchTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("patch").WithButtonStyle(LargeIconButton).
			WithTooltip("Patch — fill a closed sketch boundary with a surface."),
		NewCommand("Surface.Trim", "Trim", "Surface", func(s *Session) error {
			s.StartTool(NewSurfaceTrimTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("surface-trim").WithButtonStyle(LargeIconButton).
			WithTooltip("Trim — cut a surface with a work plane and keep one side."),
		NewCommand("Surface.Stitch", "Stitch", "Surface", func(s *Session) error {
			s.StartTool(NewStitchTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("stitch").WithButtonStyle(LargeIconButton).
			WithTooltip("Stitch — weld surface bodies into one quilt (a closed quilt becomes a solid)."),
		NewCommand("Surface.Sculpt", "Sculpt", "Surface", func(s *Session) error {
			s.StartTool(NewSculptTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("sculpt").WithButtonStyle(LargeIconButton).
			WithTooltip("Sculpt — fill the volume bounded by surfaces into a solid."),
		NewCommand("Surface.Extend", "Extend", "Surface", func(s *Session) error {
			s.StartTool(NewExtendTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("surface-extend").WithButtonStyle(LargeIconButton).
			WithTooltip("Extend — grow a surface outward along a boundary edge."),
		NewCommand("Surface.Ruled", "Ruled Surface", "Surface", func(s *Session) error {
			s.StartTool(NewRuledSurfaceTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("ruled-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Ruled Surface — sweep a closed profile's edges by straight rulings into a band."),
		NewCommand("Surface.Offset", "Offset Surface", "Surface", func(s *Session) error {
			s.StartTool(NewSurfaceOffsetTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("surface-offset").WithButtonStyle(LargeIconButton).
			WithTooltip("Offset Surface — copy the running surface along its normal by a distance."),
		NewCommand("Surface.MidSurface", "Mid-Surface", "Surface", func(s *Session) error {
			s.StartTool(NewMidSurfaceTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("mid-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Mid-Surface — extract mid-plane patches from the solid's thin walls (for FEA)."),
	}
}

// patternFeatureCommands are the 3D Model tab's Pattern panel: replicate selected features
// as real placed copies (canonical ribbon: Rectangular, Circular, Sketch Driven, Mirror).
// Each starts an interactive tool fed the source features.
func patternFeatureCommands() []*CommandDefinition {
	pats := []struct {
		id, name, icon, tip string
		start               func() Tool
	}{
		{"Modify.RectangularPattern", "Rectangular", "rectangular-pattern", "Rectangular Pattern — select features, set counts and spacing.", func() Tool { return NewFeatureRectPatternTool() }},
		{"Modify.CircularPattern", "Circular", "circular-pattern", "Circular Pattern — select features, set count and angle.", func() Tool { return NewFeatureCircPatternTool() }},
		{"Modify.Mirror", "Mirror", "mirror", "Mirror — select features, set the mirror-plane normal.", func() Tool { return NewFeatureMirrorTool() }},
		{"Modify.SketchDrivenPattern", "Sketch Driven", "sketch-driven-pattern", "Sketch-Driven Pattern — select features, then the sketch whose points place the copies.", func() Tool { return NewFeatureSketchDrivenPatternTool() }},
	}
	cmds := make([]*CommandDefinition, len(pats))
	for i, p := range pats {
		start := p.start
		cmds[i] = NewCommand(p.id, p.name, "Pattern", func(s *Session) error {
			s.StartTool(start())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).WithTooltip(p.tip).
			WithIcon(p.icon).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// sketch3DToolCommands are the contextual 3D-sketch tools, enabled only while a 3D sketch
// is being edited (M22-F12): the geometry tools, the Constrain panel, plus Finish.
func sketch3DToolCommands() []*CommandDefinition {
	finish := NewCommand("Sketch3D.Finish", "Finish 3D Sketch", "Sketch3D", func(s *Session) error {
		return s.FinishSketch3D()
	}).WithTab(tab3DSketch).WithEnable(inSketch3D).WithIcon("finish-sketch").
		WithButtonStyle(LargeIconButton).
		WithTooltip("Finish the 3D sketch and return to the model.")
	cmds := append(sketch3DDrawCommands(), sketch3DConstrainCommands()...)
	return append(cmds, finish)
}

// sketch3DConstrainCommands are the 3D Sketch tab's Constrain panel (issues #142 and
// #144): Dimension first (Inventor's panel order), then the constraint tools — each
// starts an interactive tool the user feeds 3D geometry, the same tool-first flow as
// the 2D Constrain panel.
func sketch3DConstrainCommands() []*CommandDefinition {
	cmds := []*CommandDefinition{NewCommand("Sketch3D.Dimension", "Dimension", "Constrain", func(s *Session) error {
		s.StartTool(newDimension3DTool())
		return nil
	}).WithTab(tab3DSketch).WithEnable(inSketch3D).WithIcon("dimension").WithButtonStyle(SmallIconButton).
		WithTooltip("Dimension — pick a spline, a circle, a line, or two points to dimension.")}
	for _, d := range sketch3DConstraintToolDefs {
		newTool := d.new
		cmds = append(cmds, NewCommand(d.id, d.name, "Constrain", func(s *Session) error {
			s.StartTool(newTool())
			return nil
		}).WithTab(tab3DSketch).WithEnable(inSketch3D).WithTooltip(d.tooltip).
			WithIcon(d.icon).WithButtonStyle(SmallIconButton))
	}
	return cmds
}

// sketch3DDrawCommands are the 3D-sketch geometry-placement tools (line/point/circle/arc/
// splines/equation curve/helix), each starting its interactive tool.
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
		sketch3DToolCommand("Sketch3D.Spline", "3D Spline", "spline",
			"3D Spline — a smooth curve interpolating clicked points.",
			func() Tool { return NewSpline3DTool() }),
		sketch3DToolCommand("Sketch3D.ControlPointSpline", "Control Point Spline", "spline-control",
			"Control Point Spline — a smooth curve shaped by its clicked control polygon.",
			func() Tool { return NewControlPointSpline3DTool() }),
		sketch3DToolCommand("Sketch3D.EquationCurve", "Equation Curve", "equation-curve",
			"Equation Curve — a parametric curve from x(t), y(t), z(t) over a t range.",
			func() Tool { return NewEquationCurve3DTool() }),
		sketch3DToolCommand("Sketch3D.Helix", "Helical Curve", "helix",
			"Helical Curve — a spring/thread path from radius, pitch and turns.",
			func() Tool { return NewHelix3DTool() }),
		sketch3DToolCommand("Sketch3D.Bend", "Bend", "fillet",
			"Bend — set a radius, then pick two connected lines to round their corner with a tangent arc.",
			func() Tool { return NewBend3DTool() }),
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
	}).WithTab(tab3DSketch).WithEnable(inSketch3D).WithIcon(icon).
		WithButtonStyle(SmallIconButton).WithTooltip(tip)
}

// modifyFeatureCommands are the 3D Model tab's "Modify" panel: the material-cutting
// features (hole/chamfer), the local face operations (shell/offset/draft/delete/replace),
// and surface→solid (thicken).
func modifyFeatureCommands() []*CommandDefinition {
	cmds := append(cutFeatureCommands(), localFaceCommands()...)
	cmds = append(cmds, surfaceSolidCommands()...)
	return append(cmds, directEditCommands()...)
}

// directEditCommands are the Modify features that combine or relocate existing geometry:
// Combine (boolean of two bodies), Move Face (direct edit), Move (relocate a body). They
// were model-complete (M09/M20) but had no ribbon tool.
func directEditCommands() []*CommandDefinition {
	defs := []struct {
		id, name, icon, tip string
		start               func() Tool
	}{
		{"Modify.Combine", "Combine", "combine", "Combine — boolean two bodies (Join/Cut/Intersect).", func() Tool { return NewCombineTool() }},
		{"Modify.Split", "Split", "split", "Split — divide the part by a work plane into two bodies, or trim one side away.", func() Tool { return NewSplitTool() }},
		{"Modify.MoveFace", "Move Face", "move-face", "Move Face — translate picked faces, retopologizing the solid.", func() Tool { return NewMoveFaceTool() }},
		{"Modify.MoveBodies", "Move Bodies", "move-bodies", "Move Bodies — relocate a body by a vector.", func() Tool { return NewMoveBodyTool() }},
		{"Modify.DirectEdit", "Direct Edit", "direct-edit", "Direct Edit — move, push/pull, rotate, delete or scale picked geometry (#332).", func() Tool { return NewDirectEditTool() }},
		{"Modify.Hull", "Hull", "hull", "Hull — wrap the part's solids into one convex solid.", func() Tool { return NewHullTool() }},
	}
	cmds := make([]*CommandDefinition, len(defs))
	for i, d := range defs {
		start := d.start
		cmds[i] = NewCommand(d.id, d.name, "Modify", func(s *Session) error {
			s.StartTool(start())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).WithTooltip(d.tip).
			WithIcon(d.icon).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// surfaceSolidCommands are the Modify features that turn a surface body into a solid.
func surfaceSolidCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Modify.Thicken", "Thicken", "Modify", func(s *Session) error {
			s.StartTool(NewThickenTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
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
		}).WithTab(tab3DModel).WithAlias("H").WithEnable(notInSketch).
			WithIcon("hole").WithButtonStyle(LargeIconButton).
			WithTooltip("Hole — drill a cylindrical hole into a planar face of the solid."),
		NewCommand("Modify.Boss", "Boss", "Modify", func(s *Session) error {
			s.StartTool(NewBossTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("boss").WithButtonStyle(LargeIconButton).
			WithTooltip("Boss — raise a cylindrical stud on a planar face (the join-side mirror of Hole)."),
		NewCommand("Modify.Chamfer", "Chamfer", "Modify", func(s *Session) error {
			s.StartTool(NewChamferTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("chamfer").WithButtonStyle(LargeIconButton).
			WithTooltip("Chamfer — bevel selected edges by a setback distance."),
		NewCommand("Modify.Fillet", "Fillet", "Modify", func(s *Session) error {
			s.StartTool(NewFilletTool())
			return nil
		}).WithTab(tab3DModel).WithAlias("F").WithEnable(notInSketch).
			WithIcon("fillet").WithButtonStyle(LargeIconButton).
			WithTooltip("Fillet — round selected convex edges with a rolling-ball radius."),
		NewCommand("Modify.Thread", "Thread", "Modify", func(s *Session) error {
			s.StartTool(NewThreadTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("thread").WithButtonStyle(LargeIconButton).
			WithTooltip("Thread — apply a cosmetic or modeled-cut thread to a cylindrical face (ISO/ANSI/JIS)."),
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
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("shell").WithButtonStyle(LargeIconButton).
			WithTooltip("Shell — hollow the solid to a wall thickness, removing the selected faces."),
		NewCommand("Modify.FaceOffset", "Offset Face", "Modify", func(s *Session) error {
			s.StartTool(NewFaceOffsetTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("face-offset").WithButtonStyle(LargeIconButton).
			WithTooltip("Offset Face — move selected faces along their normal, retrimming neighbours."),
		NewCommand("Modify.Draft", "Draft", "Modify", func(s *Session) error {
			s.StartTool(NewDraftTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
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
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("delete-face").WithButtonStyle(LargeIconButton).
			WithTooltip("Delete Face — remove selected faces and heal the openings."),
		NewCommand("Modify.ReplaceFace", "Replace Face", "Modify", func(s *Session) error {
			s.StartTool(NewReplaceFaceTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
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
		}).WithTab(tab3DModel).WithAlias("E").WithEnable(notInSketch).
			WithIcon("extrude").WithButtonStyle(LargeIconButton).
			WithTooltip("Extrude — add depth to a sketch profile to create or modify a solid."),
		NewCommand("Create.Revolve", "Revolve", "Create", func(s *Session) error {
			s.StartTool(NewRevolveTool())
			return nil
		}).WithTab(tab3DModel).WithAlias("R").WithEnable(notInSketch).
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
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("sweep").WithButtonStyle(LargeIconButton).
			WithTooltip("Sweep — run a sketch profile along a path to create or modify a solid."),
		NewCommand("Create.Loft", "Loft", "Create", func(s *Session) error {
			s.StartTool(NewLoftTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("loft").WithButtonStyle(LargeIconButton).
			WithTooltip("Loft — blend two or more sketch sections into a solid."),
		NewCommand("Create.Coil", "Coil", "Create", func(s *Session) error {
			s.StartTool(NewCoilTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("coil").WithButtonStyle(LargeIconButton).
			WithTooltip("Coil — sweep a sketch profile along a helix to create or modify a solid."),
		NewCommand("Create.Rib", "Rib", "Create", func(s *Session) error {
			s.StartTool(NewRibTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("rib").WithButtonStyle(LargeIconButton).
			WithTooltip("Rib — thicken an open sketch profile into a reinforcing wall joined to the part."),
		NewCommand("Create.Emboss", "Emboss", "Create", func(s *Session) error {
			s.StartTool(NewEmbossTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(notInSketch).
			WithIcon("emboss").WithButtonStyle(LargeIconButton).
			WithTooltip("Emboss — raise or engrave a closed sketch profile on the part."),
		NewCommand("Create.Decal", "Decal", "Create", func(s *Session) error {
			s.StartTool(NewDecalTool())
			return nil
		}).WithTab(tab3DModel).WithEnable(hasActivePart).
			WithIcon("decal").WithButtonStyle(LargeIconButton).
			WithTooltip("Decal — project an image onto a face (cosmetic)."),
	}
}

// viewTabCommands are the View tab commands: navigation, the Visual Style presets, and the
// lighting/environment/shadow controls (M16/F03).
func viewTabCommands() []*CommandDefinition {
	cmds := append(viewNavigateCommands(), visualStyleCommands()...)
	cmds = append(cmds, lightingViewCommands()...)
	return append(cmds, windowsViewCommands()...)
}

// windowsViewCommands is the View tab's "Windows" panel (Inventor's window-tiling home):
// add/close a view of the active document and choose how its views tile the viewport.
// A document owns a collection of views, each with its own camera (Document → Views →
// Camera); these drive the same Views collection the API and .obk persistence use.
func windowsViewCommands() []*CommandDefinition {
	layout := func(id, name, icon string, l types.ViewLayout) *CommandDefinition {
		return NewCommand(id, name, "Windows", func(s *Session) error {
			return s.SetViewLayout(l)
		}).WithTab("View").WithEnable(hasActivePart).WithIcon(icon).WithButtonStyle(SmallIconButton).
			WithTooltip(name + " — tile the active document's views this way.")
	}
	return []*CommandDefinition{
		NewCommand("View.New", "New View", "Windows", func(s *Session) error {
			return s.NewView()
		}).WithTab("View").WithEnable(hasActivePart).WithIcon("view-new").WithButtonStyle(LargeIconButton).
			WithTooltip("New View — add another view of this document, with its own camera."),
		NewCommand("View.Close", "Close View", "Windows", func(s *Session) error {
			return s.CloseActiveView()
		}).WithTab("View").WithEnable(hasActivePart).WithIcon("view-close").WithButtonStyle(SmallIconButton).
			WithTooltip("Close View — remove the active view (a document keeps at least one)."),
		NewCommand("View.ViewCube", "ViewCube", "Windows", func(s *Session) error {
			s.SetShowViewCube(!s.ShowViewCube())
			return nil
		}).WithTab("View").WithEnable(hasActivePart).WithIcon("view-cube").WithButtonStyle(SmallIconButton).
			WithActive(func(s *Session) bool { return s.ShowViewCube() }).
			WithTooltip("ViewCube — show or hide the navigation cube in each viewport."),
		layout("View.LayoutSingle", "Single View", "layout-single", types.LayoutSingle),
		layout("View.LayoutTwoH", "Two Views (Side by Side)", "layout-two-h", types.LayoutTwoH),
		layout("View.LayoutTwoV", "Two Views (Stacked)", "layout-two-v", types.LayoutTwoV),
		layout("View.LayoutThree", "Three Views", "layout-three", types.LayoutThree),
		layout("View.LayoutFour", "Four Views", "layout-four", types.LayoutFour),
	}
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
