// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/renderer"
)

// RegisterStandardCommands wires the standard ribbon for a session: the Create & Modify
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
	// Work Features registers before the modelling tools so its panel heads the Create & Modify tab.
	cmds = append(cmds, workFeatureCommands()...)
	cmds = append(cmds, modelTabCommands()...)
	cmds = append(cmds, manageTabCommands()...)
	cmds = append(cmds, sketchTabCommands()...)
	cmds = append(cmds, assemblyTabCommands()...)
	cmds = append(cmds, sheetMetalTabCommands()...)
	cmds = append(cmds, newDrawingCommand())
	cmds = append(cmds, drawingTabCommands()...)
	// The View and Inspect tabs are shared by the Part and Assembly ribbons (the navigation,
	// display and measure tools apply to both document types).
	cmds = append(cmds, onRibbons(viewTabCommands(), PartRibbon, AssemblyRibbon)...)
	cmds = append(cmds, onRibbons(analysisCommands(), PartRibbon, AssemblyRibbon)...)
	return cmds
}

// onRibbons scopes every command in the group to the given document ribbons — a bulk
// WithRibbons so a whole tab (e.g. View) can be shared across ribbons in one place.
func onRibbons(cmds []*CommandDefinition, keys ...RibbonKey) []*CommandDefinition {
	for _, c := range cmds {
		c.WithRibbons(keys...)
	}
	return cmds
}

// getStartedCommands are the ZeroDoc ribbon's Get Started tab — shown when no document is
// open. New Part / New Assembly are the launch actions; each creates a document, which switches
// the active ribbon to that type's ribbon (RibbonUI_Overview's per-document-type ribbons).
func getStartedCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("GetStarted.NewPart", "New Part", "Launch", func(s *Session) error {
			_, err := s.NewPart()
			return err
		}).WithTab(tabGetStarted).WithRibbons(ZeroDocRibbon).WithDefaultChord("Ctrl+N").
			WithIcon("new-part").WithButtonStyle(LargeIconButton).
			WithTooltip("New Part — create a part document and open the part environment."),
		NewCommand("GetStarted.NewSheetMetalPart", "New Sheet Metal Part", "Launch", func(s *Session) error {
			_, err := s.NewSheetMetalPart()
			return err
		}).WithTab(tabGetStarted).WithRibbons(ZeroDocRibbon).
			WithIcon("sheet-metal-new").WithButtonStyle(LargeIconButton).
			WithTooltip("New Sheet Metal Part — create a part already in the sheet-metal environment."),
		NewCommand("GetStarted.NewAssembly", "New Assembly", "Launch", func(s *Session) error {
			_, err := s.NewAssembly()
			return err
		}).WithTab(tabGetStarted).WithRibbons(ZeroDocRibbon).
			WithIcon("new-assembly").WithButtonStyle(LargeIconButton).
			WithTooltip("New Assembly — create an assembly document and open the assembly environment, where you place and constrain components."),
		NewCommand("GetStarted.AddInCatalogue", "AddIn Catalogue", "Manage", func(s *Session) error {
			s.RequestAddInCatalogue()
			return nil
		}).WithTab(tabGetStarted).WithRibbons(ZeroDocRibbon).
			WithIcon("addin-catalogue").WithButtonStyle(LargeIconButton).
			WithTooltip("AddIn Catalogue — browse, install and update add-ins for this host."),
		NewCommand("GetStarted.Preferences", "Preferences", "Manage", func(s *Session) error {
			s.RequestPreferences()
			return nil
		}).WithTab(tabGetStarted).WithRibbons(ZeroDocRibbon).
			WithIcon("preferences").WithButtonStyle(LargeIconButton).
			WithTooltip("Preferences — application settings (UI scale, sketch grid, theme, privacy…)."),
	}
}

// manageTabCommands are the Manage tab: the Parameters panel (Inventor's Manage ▸
// Parameters, shown for a part OR an assembly — both hold parameters) and the Scripts panel
// (the Script Console — our equivalent of Manage ▸ iLogic, ADR-0028).
func manageTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Manage.Parameters", "Parameters", "Parameters", func(s *Session) error {
			s.OpenParameters()
			return nil
		}).WithTab("Manage").WithRibbons(PartRibbon, AssemblyRibbon).WithEnable(hasActiveParameterHolder).
			WithIcon("parameters").WithButtonStyle(LargeIconButton).
			WithTooltip("Parameters — add, edit, and organize the model and user parameters that drive the part or assembly."),
		scriptConsoleCommand(),
		deriveAssemblyCommand(),
		shrinkwrapCommand(),
	}
}

// deriveAssemblyCommand / shrinkwrapCommand are the Manage tab's Simplify panel (#767): merge an
// open assembly into the active part as a base body — full or simplified. Each is enabled only when
// a part is active AND an assembly is open to derive (no source ⇒ nothing to do), and starts its
// tool, which reads the source/options from the generic dialog.
func deriveAssemblyCommand() *CommandDefinition {
	return NewCommand("Manage.Derive", "Derive Assembly", "Simplify", func(s *Session) error {
		s.StartTool(NewDeriveAssemblyTool())
		return nil
	}).WithTab("Manage").WithEnable(canDeriveAssembly).
		WithIcon("derive").WithButtonStyle(LargeIconButton).
		WithTooltip("Derive Assembly — merge an open assembly into this part as one base body, linked to the source.")
}

func shrinkwrapCommand() *CommandDefinition {
	return NewCommand("Manage.Shrinkwrap", "Shrinkwrap", "Simplify", func(s *Session) error {
		s.StartTool(NewShrinkwrapTool())
		return nil
	}).WithTab("Manage").WithEnable(canDeriveAssembly).
		WithIcon("shrinkwrap").WithButtonStyle(LargeIconButton).
		WithTooltip("Shrinkwrap — merge an open assembly into this part as a simplified, lightweight base body.")
}

// canDeriveAssembly reports whether a part is active and at least one assembly is open to derive.
func canDeriveAssembly(s *Session) bool {
	return hasActivePart(s) && len(s.OpenAssemblies()) > 0
}

// hasActivePart reports whether the active document is a part (the Parameters dialog
// needs one to read and edit).
func hasActivePart(s *Session) bool {
	_, err := activePart(s)
	return err == nil
}

// hasActiveParameterHolder reports whether the active document holds parameters — a part OR an
// assembly. The Parameters command uses it so the dialog opens for either (M39-F04, #1560).
func hasActiveParameterHolder(s *Session) bool {
	_, err := s.activeParameterHolder()
	return err == nil
}

// workFeatureCommands are the Create & Modify tab's Work Features panel: the datum-plane
// constructors. Each button is always live in the part environment (Inventor's Work Plane
// behavior) — click it and, if the right geometry is already selected it builds the datum
// at once, otherwise it starts a guided pick that prompts for the inputs and commits when
// they are gathered. So a click is never inert, whether or not anything was pre-selected.
func workFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("WorkPlane.Offset", "Offset Plane", panelWorkFeatures, func(s *Session) error {
			s.StartTool(NewOffsetWorkPlaneTool()) // always opens the distance dialog (Inventor's flow)
			return nil
		}).WithTab(tabCreateModify).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-offset").WithButtonStyle(LargeIconButton).
			WithTooltip("Offset Plane — a work plane parallel to a plane, offset by a distance. Pick a plane, then enter the offset."),
		NewCommand("WorkPlane.Midplane", "Midplane", panelWorkFeatures, startWorkPlane(newMidplaneWorkPlaneTool)).
			WithTab(tabCreateModify).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-midplane").WithButtonStyle(SmallIconButton).
			WithTooltip("Midplane — a work plane bisecting two planes. Pick two planes when prompted."),
		NewCommand("WorkPlane.ThreePoints", "Three Points", panelWorkFeatures, startWorkPlane(newThreePointWorkPlaneTool)).
			WithTab(tabCreateModify).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-3pt").WithButtonStyle(SmallIconButton).
			WithTooltip("Three Points — a work plane through three points or model vertices. Pick three when prompted."),
		NewCommand("WorkPlane.Tangent", "Tangent to Face", panelWorkFeatures, startWorkPlane(newTangentWorkPlaneTool)).
			WithTab(tabCreateModify).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-tangent").WithButtonStyle(SmallIconButton).
			WithTooltip("Tangent to Face — a work plane parallel to a plane and tangent to a cylindrical/spherical face. Pick a plane then a face."),
		NewCommand("WorkPlane.NormalToAxis", "Normal to Axis", panelWorkFeatures, startWorkPlane(newNormalToAxisWorkPlaneTool)).
			WithTab(tabCreateModify).WithEnable(canStartWorkPlane).
			WithIcon("work-plane-normal").WithButtonStyle(SmallIconButton).
			WithTooltip("Normal to Axis — a work plane through a point, normal to an axis. Pick an axis then a point."),
	}
}

// modelTabCommands are the Create & Modify tab: starting a sketch and the solid features.
func modelTabCommands() []*CommandDefinition {
	cmds := []*CommandDefinition{
		// The Sketch panel repeats on both modelling tabs (WithTabs) — Inventor shows the sketch
		// starters on each part tab so a sketch is always one click away.
		NewCommand("Sketch.Create2D", "New 2D Sketch", "Sketch", func(s *Session) error {
			// With a host (work plane or planar face) already selected, sketch on it
			// immediately; otherwise start the tool and let the user pick one in the 3D
			// view or the browser.
			if _, ok := s.SelectedSketchHostPlane(); ok {
				_, err := s.CreateSketchOnSelectedPlane()
				return err
			}
			s.StartTool(NewCreateSketchTool())
			return nil
		}).WithTabs(tabCreateModify, tabSurfacesMesh).WithAlias("S").WithEnable(canCreateSketch).
			WithIcon("create-sketch").WithButtonStyle(LargeIconButton).
			WithTooltip("New 2D Sketch — pick a work plane or planar face to sketch on."),
		NewCommand("Sketch.Create3D", "New 3D Sketch", "Sketch", func(s *Session) error {
			_, err := s.CreateSketch3D()
			return err
		}).WithTabs(tabCreateModify, tabSurfacesMesh).WithEnable(canCreateSketch3D).
			WithIcon("create-sketch-3d").WithButtonStyle(LargeIconButton).
			WithTooltip("New 3D Sketch — create a non-planar sketch (sweep/loft path, helix)."),
	}
	cmds = append(cmds, sketch3DToolCommands()...)
	cmds = append(cmds, solidFeatureCommands()...)
	cmds = append(cmds, modifyFeatureCommands()...)
	cmds = append(cmds, patternFeatureCommands()...)
	// Surfaces & Mesh tab order: Surface, Freeform, Mesh, Point Cloud, Mold.
	cmds = append(cmds, surfaceFeatureCommands()...)
	cmds = append(cmds, freeformFeatureCommands()...)
	cmds = append(cmds, meshFeatureCommands()...)
	cmds = append(cmds, pointCloudCommands()...)
	return append(cmds, moldFeatureCommands()...)
}

// meshFeatureCommands are the Surfaces & Mesh tab's Mesh panel: place an STL as mesh reference
// geometry (M10-F04, #700). The command arms the head's file dialog; the import itself is
// Session.ImportMeshFile.
func meshFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Mesh.Place", "Place Mesh", "Mesh", func(s *Session) error {
			s.RequestImportMesh()
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("mesh-place").WithButtonStyle(LargeIconButton).
			WithTooltip("Place Mesh — load an ASCII STL as selectable mesh reference geometry."),
	}
}

// pointCloudCommands are the Surfaces & Mesh tab's Point Cloud panel: attach a laser-scan /
// photogrammetry file as a referenced display object (M17-F06, #645). Like Place Mesh, the
// command arms the head's file dialog; the attach itself is Session.AttachPointCloud.
func pointCloudCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("PointCloud.Import", "Import Point Cloud", "Point Cloud", func(s *Session) error {
			s.RequestImportPointCloud()
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("point-cloud-import").WithButtonStyle(LargeIconButton).
			WithTooltip("Import Point Cloud — attach an ASCII scan (.xyz/.pts) as referenced display data."),
		NewCommand("PointCloud.FitPlane", "Fit Work Plane", "Point Cloud", func(s *Session) error {
			_, err := s.FitSelectedCloudPlane()
			return err
		}).WithTab(tabSurfacesMesh).WithEnable(canFitPointCloudPlane).
			WithIcon("point-cloud-fit-plane").WithButtonStyle(SmallIconButton).
			WithTooltip("Fit Work Plane — least-squares plane through the selected cloud's displayed points (crop first to fit a region)."),
		NewCommand("PointCloud.WorkPoint", "Work Point", "Point Cloud", func(s *Session) error {
			_, err := s.CreateWorkPointAtSelectedCloudPoint()
			return err
		}).WithTab(tabSurfacesMesh).WithEnable(canWorkPointAtCloudPoint).
			WithIcon("point-cloud-work-point").WithButtonStyle(SmallIconButton).
			WithTooltip("Work Point — place a datum point on the selected scan point (snap to a cloud point first)."),
		NewCommand("PointCloud.CropBox", "Crop Box", "Point Cloud", func(s *Session) error {
			return s.StartCropSelectedCloud()
		}).WithTab(tabSurfacesMesh).WithEnable(canCropSelectedCloud).
			WithIcon("point-cloud-crop").WithButtonStyle(SmallIconButton).
			WithTooltip("Crop Box — box a region of the selected cloud in the viewport to crop its display to those points."),
		NewCommand("PointCloud.Move", "Move", "Point Cloud", func(s *Session) error {
			return s.StartMoveSelectedCloud()
		}).WithTab(tabSurfacesMesh).WithEnable(canMoveSelectedCloud).
			WithIcon("point-cloud-move").WithButtonStyle(SmallIconButton).
			WithTooltip("Move — drag the selected cloud in the viewport; datums built on it follow as it moves."),
	}
}

// moldFeatureCommands are the Surfaces & Mesh tab's Mold panel: the core/cavity tooling split
// (M10-F04, #701).
func moldFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Mold.CoreCavity", "Core/Cavity", "Mold", func(s *Session) error {
			s.StartFeatureTool(NewCoreCavityTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("core-cavity").WithButtonStyle(LargeIconButton).
			WithTooltip("Core/Cavity — split the tooling block at a parting plane into core and cavity solids."),
	}
}

// freeformFeatureCommands are the Surfaces & Mesh tab's Freeform panel: the sub-D primitives
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
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).WithTooltip(d.tip).
			WithIcon(d.icon).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// surfaceFeatureCommands are the Surfaces & Mesh tab's Surface panel (canonical: Patch, Stitch, Sculpt,
// Extend, Trim, Rule Fillet). Patch fills a closed sketch region with a surface.
func surfaceFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Surface.Patch", "Patch", "Surface", func(s *Session) error {
			s.StartFeatureTool(NewPatchTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("patch").WithButtonStyle(LargeIconButton).
			WithTooltip("Patch — fill a closed sketch boundary with a surface."),
		NewCommand("Surface.Trim", "Trim", "Surface", func(s *Session) error {
			s.StartFeatureTool(NewSurfaceTrimTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("surface-trim").WithButtonStyle(LargeIconButton).
			WithTooltip("Trim — cut a surface with a work plane and keep one side."),
		NewCommand("Surface.Stitch", "Stitch", "Surface", func(s *Session) error {
			s.StartFeatureTool(NewStitchTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("stitch").WithButtonStyle(LargeIconButton).
			WithTooltip("Stitch — weld surface bodies into one quilt (a closed quilt becomes a solid)."),
		NewCommand("Surface.Sculpt", "Sculpt", "Surface", func(s *Session) error {
			s.StartFeatureTool(NewSculptTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("sculpt").WithButtonStyle(LargeIconButton).
			WithTooltip("Sculpt — fill the volume bounded by surfaces into a solid."),
		NewCommand("Surface.Extend", "Extend", "Surface", func(s *Session) error {
			s.StartFeatureTool(NewExtendTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("surface-extend").WithButtonStyle(LargeIconButton).
			WithTooltip("Extend — grow a surface outward along a boundary edge."),
		NewCommand("Surface.RuleFillet", "Rule Fillet", "Surface", func(s *Session) error {
			s.StartTool(NewRuleFilletTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("fillet").WithButtonStyle(LargeIconButton).
			WithTooltip("Rule Fillet — round a whole class of edges (all rounds, all fillets, or all edges) at one radius."),
		NewCommand("Surface.Ruled", "Ruled Surface", "Surface", func(s *Session) error {
			s.StartTool(NewRuledSurfaceTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("ruled-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Ruled Surface — sweep a closed profile's edges by straight rulings into a band."),
		NewCommand("Surface.Offset", "Offset Surface", "Surface", func(s *Session) error {
			s.StartTool(NewSurfaceOffsetTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("surface-offset").WithButtonStyle(LargeIconButton).
			WithTooltip("Offset Surface — copy the running surface along its normal by a distance."),
		NewCommand("Surface.MidSurface", "Mid-Surface", "Surface", func(s *Session) error {
			s.StartTool(NewMidSurfaceTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("mid-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Mid-Surface — extract mid-plane patches from the solid's thin walls (for FEA)."),
		NewCommand("Surface.Rebuild", "Rebuild Surface", "Surface", func(s *Session) error {
			s.StartTool(NewSurfaceRebuildTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("rebuild-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Rebuild Surface — refit freeform faces to clean Class-A NURBS (fewer, even control points)."),
		NewCommand("Surface.NurbsPlane", "NURBS Plane", "Surface", func(s *Session) error {
			s.StartTool(NewNurbsPlaneTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("nurbs-plane").WithButtonStyle(LargeIconButton).
			WithTooltip("NURBS Plane — create a flat Class-A NURBS surface patch to start styling from."),
		NewCommand("Surface.EditCV", "Edit Control Points", "Surface", func(s *Session) error {
			s.StartTool(NewControlPointEditTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("control-point-edit").WithButtonStyle(LargeIconButton).
			WithTooltip("Edit Control Points — drag a NURBS surface's control net to shape it (region, falloff, symmetry)."),
		NewCommand("Surface.Match", "Match Surface", "Surface", func(s *Session) error {
			s.StartTool(NewMatchTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("match-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Match Surface — rebuild the surface against its neighbour to G0/G1/G2/G3 continuity."),
		NewCommand("Surface.ExtendNurbs", "Extend Surface", "Surface", func(s *Session) error {
			s.StartTool(NewExtendSurfaceTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("extend-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Extend Surface — lengthen a NURBS surface past an edge with tangent (G1) or curvature (G2) continuation."),
		NewCommand("Surface.Untrim", "Untrim", "Surface", untrimSurface).
			WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("untrim-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Untrim — recover a trimmed NURBS face's full underlying surface."),
		NewCommand("Surface.Fill", "Fill Surface", "Surface", func(s *Session) error {
			s.StartTool(NewFillSurfaceTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("fill-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Fill Surface — close a four-sided opening with a single clean NURBS at G0/G1/G2 continuity."),
		NewCommand("Surface.Bridge", "Bridge Surface", "Surface", func(s *Session) error {
			s.StartTool(NewBridgeSurfaceTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("bridge-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Bridge Surface — connect two surfaces with a clean NURBS transition at G0/G1/G2 continuity per side."),
		NewCommand("Surface.Network", "Network Surface", "Surface", func(s *Session) error {
			s.StartTool(NewNetworkTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("network-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Network Surface — interpolate a grid of intersecting U and V curves with a single NURBS (Gordon surface)."),
		NewCommand("Surface.Fair", "Fair Surface", "Surface", func(s *Session) error {
			s.StartTool(NewFairSurfaceTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("fair-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Fair Surface — smooth curvature wrinkles out of a surface while holding its boundary continuity (G0/G1/G2)."),
		NewCommand("Surface.FitToCloud", "Fit Surface", "Surface", func(s *Session) error {
			s.StartTool(NewFitSurfaceTool())
			return nil
		}).WithTab(tabSurfacesMesh).WithEnable(hasActivePart).
			WithIcon("fit-surface").WithButtonStyle(LargeIconButton).
			WithTooltip("Fit Surface — fit a clean Class-A NURBS to a scanned point-cloud region (degree + U/V spans), reporting the deviation to the scan."),
	}
}

// patternFeatureCommands are the Create & Modify tab's Pattern panel: replicate selected features
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
		}).WithTab(tabCreateModify).WithEnable(hasActivePart).WithTooltip(p.tip).
			WithIcon(p.icon).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// sketch3DToolCommands are the contextual 3D-sketch tools, enabled only while a 3D sketch
// is being edited (M22-F12): the geometry tools, the Constrain panel, plus Finish.
func sketch3DToolCommands() []*CommandDefinition {
	finish := NewCommand("Sketch3D.Finish", "Finish 3D Sketch", "Exit", func(s *Session) error {
		return s.FinishSketch3D()
	}).WithTab(tab3DSketch).WithEnvironment(Sketch3DEnvironment).WithEnable(inSketch3D).WithIcon("finish-sketch").
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
	}).WithTab(tab3DSketch).WithEnvironment(Sketch3DEnvironment).WithEnable(inSketch3D).WithIcon("dimension").WithButtonStyle(SmallIconButton).
		WithTooltip("Dimension — pick a spline, a circle, a line, or two points to dimension.")}
	for _, d := range sketch3DConstraintToolDefs {
		newTool := d.new
		cmds = append(cmds, NewCommand(d.id, d.name, "Constrain", func(s *Session) error {
			s.StartTool(newTool())
			return nil
		}).WithTab(tab3DSketch).WithEnvironment(Sketch3DEnvironment).WithEnable(inSketch3D).WithTooltip(d.tooltip).
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
	}).WithTab(tab3DSketch).WithEnvironment(Sketch3DEnvironment).WithEnable(inSketch3D).WithIcon(icon).
		WithButtonStyle(SmallIconButton).WithTooltip(tip)
}

// modifyFeatureCommands are the Create & Modify tab's "Modify" panel: the material-cutting
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
		}).WithTab(tabCreateModify).WithEnable(hasActivePart).WithTooltip(d.tip).
			WithIcon(d.icon).WithButtonStyle(SmallIconButton)
	}
	return cmds
}

// surfaceSolidCommands are the Modify features that turn a surface body into a solid.
func surfaceSolidCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Modify.Thicken", "Thicken", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewThickenTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("thicken").WithButtonStyle(LargeIconButton).
			WithTooltip("Thicken — turn the active surface body into a solid of a wall thickness."),
	}
}

// cutFeatureCommands are the Modify features that remove material against picked topology.
func cutFeatureCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Modify.Hole", "Hole", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewHoleTool())
			return nil
		}).WithTab(tabCreateModify).WithAlias("H").WithEnable(notInSketch).
			WithIcon("hole").WithButtonStyle(LargeIconButton).
			WithTooltip("Hole — drill a cylindrical hole into a planar face of the solid."),
		NewCommand("Modify.Boss", "Boss", "Modify", func(s *Session) error {
			s.StartTool(NewBossTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("boss").WithButtonStyle(LargeIconButton).
			WithTooltip("Boss — raise a cylindrical stud on a planar face (the join-side mirror of Hole)."),
		NewCommand("Modify.Lip", "Lip", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewLipTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("lip").WithButtonStyle(LargeIconButton).
			WithTooltip("Lip — run a raised lip (or recessed groove) bead along picked edges of the part."),
		NewCommand("Modify.Chamfer", "Chamfer", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewChamferTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("chamfer").WithButtonStyle(LargeIconButton).
			WithTooltip("Chamfer — bevel selected edges by a setback distance."),
		filletCommand(),
		NewCommand("Modify.Thread", "Thread", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewThreadTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("thread").WithButtonStyle(LargeIconButton).
			WithTooltip("Thread — apply a cosmetic or modeled-cut thread to a cylindrical face (ISO/ANSI/JIS)."),
	}
}

// filletCommand is the Fillet split button: the primary rounds picked edges (Edge Fillet), and
// the dropdown adds Face Fillet — rounding the edges shared between two face sets (#694), the way
// Inventor groups the fillet variants under one command. Only the primary is returned;
// RegisterStandardCommands registers the variant for id dispatch by walking primary.Variants().
func filletCommand() *CommandDefinition {
	faceFillet := NewCommand("Modify.FaceFillet", "Face Fillet", "Modify", func(s *Session) error {
		s.StartFeatureTool(NewFaceFilletTool())
		return nil
	}).WithTab(tabCreateModify).WithEnable(notInSketch).
		WithIcon("fillet").WithButtonStyle(LargeIconButton).
		WithTooltip("Face Fillet — round the edges shared between two sets of faces.")
	fullRound := NewCommand("Modify.FullRoundFillet", "Full Round Fillet", "Modify", func(s *Session) error {
		s.StartFeatureTool(NewFullRoundFilletTool())
		return nil
	}).WithTab(tabCreateModify).WithEnable(notInSketch).
		WithIcon("fillet").WithButtonStyle(LargeIconButton).
		WithTooltip("Full Round Fillet — replace a face between two parallel sides with a half-round.")
	return NewCommand("Modify.Fillet", "Fillet", "Modify", func(s *Session) error {
		s.StartFeatureTool(NewFilletTool())
		return nil
	}).WithTab(tabCreateModify).WithAlias("F").WithEnable(notInSketch).
		WithIcon("fillet").WithButtonStyle(LargeIconButton).
		WithTooltip("Fillet — round selected convex edges with a rolling-ball radius.").
		WithVariants(faceFillet, fullRound)
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
			s.StartFeatureTool(NewShellTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("shell").WithButtonStyle(LargeIconButton).
			WithTooltip("Shell — hollow the solid to a wall thickness, removing the selected faces."),
		NewCommand("Modify.FaceOffset", "Offset Face", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewFaceOffsetTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("face-offset").WithButtonStyle(LargeIconButton).
			WithTooltip("Offset Face — move selected faces along their normal, retrimming neighbours."),
		NewCommand("Modify.Draft", "Draft", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewDraftTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("draft").WithButtonStyle(LargeIconButton).
			WithTooltip("Draft — taper selected faces by an angle about the pull direction."),
	}
}

// faceTopologyCommands change which faces exist (delete face + heal, replace face).
func faceTopologyCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Modify.DeleteFace", "Delete Face", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewDeleteFaceTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("delete-face").WithButtonStyle(LargeIconButton).
			WithTooltip("Delete Face — remove selected faces and heal the openings."),
		NewCommand("Modify.ReplaceFace", "Replace Face", "Modify", func(s *Session) error {
			s.StartFeatureTool(NewReplaceFaceTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("replace-face").WithButtonStyle(LargeIconButton).
			WithTooltip("Replace Face — move selected faces onto a target face's plane."),
	}
}

// solidFeatureCommands are the Create & Modify tab's "Create" panel — the sketched solid
// features, each launching its interactive tool. Split into sketched-profile features
// (extrude/revolve) and swept features (sweep/loft/coil) to keep each builder small.
func solidFeatureCommands() []*CommandDefinition {
	return append(profileSolidCommands(), sweptSolidCommands()...)
}

// profileSolidCommands are the single-profile solid features (extrude, revolve).
func profileSolidCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Create.Extrude", "Extrude", "Create", func(s *Session) error {
			s.StartFeatureTool(NewExtrudeTool())
			return nil
		}).WithTab(tabCreateModify).WithAlias("E").WithEnable(notInSketch).
			WithIcon("extrude").WithButtonStyle(LargeIconButton).
			WithTooltip("Extrude — add depth to a sketch profile to create or modify a solid."),
		NewCommand("Create.Revolve", "Revolve", "Create", func(s *Session) error {
			s.StartFeatureTool(NewRevolveTool())
			return nil
		}).WithTab(tabCreateModify).WithAlias("R").WithEnable(notInSketch).
			WithIcon("revolve").WithButtonStyle(LargeIconButton).
			WithTooltip("Revolve — spin a sketch profile about an axis to create or modify a solid."),
	}
}

// sweptSolidCommands are the swept/blended solid features (sweep, loft, coil).
func sweptSolidCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Create.Sweep", "Sweep", "Create", func(s *Session) error {
			s.StartFeatureTool(NewSweepTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("sweep").WithButtonStyle(LargeIconButton).
			WithTooltip("Sweep — run a sketch profile along a path to create or modify a solid."),
		NewCommand("Create.Loft", "Loft", "Create", func(s *Session) error {
			s.StartFeatureTool(NewLoftTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("loft").WithButtonStyle(LargeIconButton).
			WithTooltip("Loft — blend two or more sketch sections into a solid."),
		NewCommand("Create.Coil", "Coil", "Create", func(s *Session) error {
			s.StartFeatureTool(NewCoilTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("coil").WithButtonStyle(LargeIconButton).
			WithTooltip("Coil — sweep a sketch profile along a helix to create or modify a solid."),
		NewCommand("Create.Rib", "Rib", "Create", func(s *Session) error {
			s.StartFeatureTool(NewRibTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("rib").WithButtonStyle(LargeIconButton).
			WithTooltip("Rib — thicken an open sketch profile into a reinforcing wall joined to the part."),
		NewCommand("Create.Emboss", "Emboss", "Create", func(s *Session) error {
			s.StartFeatureTool(NewEmbossTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("emboss").WithButtonStyle(LargeIconButton).
			WithTooltip("Emboss — raise or engrave a closed sketch profile on the part."),
		NewCommand("Create.Grill", "Grill", "Create", func(s *Session) error {
			s.StartFeatureTool(NewGrillTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("grill").WithButtonStyle(LargeIconButton).
			WithTooltip("Grill — cut a ventilation grill: a vent bridged by the boundary profile's rib/spar/island structure."),
		NewCommand("Create.Rest", "Rest", "Create", func(s *Session) error {
			s.StartFeatureTool(NewRestTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(notInSketch).
			WithIcon("rest").WithButtonStyle(LargeIconButton).
			WithTooltip("Rest — raise a pad (or recess a pocket) over a closed sketch region on the part."),
		NewCommand("Create.SnapFit", "Snap Fit", "Create", func(s *Session) error {
			s.StartFeatureTool(NewSnapFitTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(hasActivePart).
			WithIcon("snap-fit").WithButtonStyle(LargeIconButton).
			WithTooltip("Snap Fit — add a cantilever snap-fit hook sized by its beam and catch dimensions."),
		NewCommand("Create.Decal", "Decal", "Create", func(s *Session) error {
			s.StartTool(NewDecalTool())
			return nil
		}).WithTab(tabCreateModify).WithEnable(hasActivePart).
			WithIcon("decal").WithButtonStyle(LargeIconButton).
			WithTooltip("Decal — project an image onto a face (cosmetic)."),
	}
}

// viewTabCommands are the View tab commands: navigation, the Visual Style presets, and the
// lighting/environment/shadow controls (M16/F03).
func viewTabCommands() []*CommandDefinition {
	cmds := append(viewNavigateCommands(), orientViewCommands()...)
	cmds = append(cmds, displayViewCommands()...)
	cmds = append(cmds, colorStylesCommand())
	cmds = append(cmds, selectionPriorityCommands()...)
	cmds = append(cmds, selectionFilterCommand())
	cmds = append(cmds, visualStyleCommands()...)
	cmds = append(cmds, colorSchemeCommands()...)
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
		}).WithTab("View").WithEnable(hasActivePart).WithDefaultChord("Ctrl+W").WithIcon("view-close").WithButtonStyle(SmallIconButton).
			WithTooltip("Close View — remove the active view (a document keeps at least one)."),
		NewCommand("View.ViewCube", "ViewCube", "Windows", func(s *Session) error {
			s.SetShowViewCube(!s.ShowViewCube())
			return nil
		}).WithTab("View").WithEnable(hasActivePart).WithIcon("view-cube").WithButtonStyle(SmallIconButton).
			WithActive(func(s *Session) bool { return s.ShowViewCube() }).
			WithTooltip("ViewCube — show or hide the navigation cube in each viewport."),
		NewCommand("View.NavBar", "Navigation Bar", "Windows", func(s *Session) error {
			s.SetShowNavBar(!s.ShowNavBar())
			return nil
		}).WithTab("View").WithEnable(hasActivePart).WithIcon("nav-bar").WithButtonStyle(SmallIconButton).
			WithActive(func(s *Session) bool { return s.ShowNavBar() }).
			WithTooltip("Navigation Bar — show or hide the floating navigation-tool strip in each viewport."),
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
		NewCommand("View.LookAt", "Look At", "Navigate", func(s *Session) error {
			s.LookAtSelection()
			return nil
		}).WithTab("View").WithIcon("look-at").WithButtonStyle(LargeIconButton).WithEnable(canLookAt).
			WithTooltip("Look At — reorient the view normal to the selected planar face or work plane."),
		NewCommand("View.ZoomWindow", "Zoom Window", "Navigate", func(s *Session) error {
			s.ArmZoomWindow()
			return nil
		}).WithTab("View").WithIcon("zoom-window").WithButtonStyle(LargeIconButton).WithEnable(hasActivePart).
			WithActive(func(s *Session) bool { return s.ZoomWindowArmed() }).
			WithTooltip("Zoom Window — drag a rectangle in the viewport; the view zooms to fit it."),
		NewCommand("View.ConstrainedOrbit", "Constrained Orbit", "Navigate", func(s *Session) error {
			s.ToggleConstrainedOrbit()
			return nil
		}).WithTab("View").WithIcon("orbit-constrained").WithButtonStyle(LargeIconButton).WithEnable(hasActivePart).
			WithActive(func(s *Session) bool { return s.ConstrainedOrbitActive() }).
			WithTooltip("Constrained Orbit — left-drag turntables about the vertical axis (horizontal = turn, vertical = tilt)."),
		NewCommand("View.SteeringWheel", "SteeringWheels", "Navigate", func(s *Session) error {
			s.ToggleSteeringWheel()
			return nil
		}).WithTab("View").WithIcon("steering-wheel").WithButtonStyle(LargeIconButton).WithEnable(hasActivePart).
			WithActive(func(s *Session) bool { return s.SteeringWheelActive() }).
			WithTooltip("SteeringWheels — a radial menu of navigation tools that follows the cursor."),
	}
}

// canLookAt enables Look At only when a planar face or work plane is selected.
func canLookAt(s *Session) bool { return s.CanLookAt() }

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
