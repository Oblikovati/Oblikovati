// SPDX-License-Identifier: GPL-2.0-only

package cmdline

// This file holds the vocabulary for the modelling environments AutoCAD's core has no
// parametric equivalent for — assembly, sheet metal and drawing-from-model — plus the 3D
// sketch, document management and view subsets. AutoCAD-core has no command for these, so
// (except the drawing-view and a few SURF-style words) they use Oblikovati-natural,
// AutoCAD-style verbs.

// assemblyVocabulary maps the assembly environment: constraints, joints, components and
// representations.
func assemblyVocabulary() []vocabEntry {
	var v []vocabEntry
	v = append(v, assemblyConstraintVocabulary()...)
	v = append(v, assemblyJointVocabulary()...)
	v = append(v, assemblyComponentVocabulary()...)
	v = append(v, assemblyManageVocabulary()...)
	return v
}

// assemblyConstraintVocabulary maps the placement constraints (ASM-prefixed where the bare
// word already belongs to a part feature).
func assemblyConstraintVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Assembly.Mate", []string{"MATE"}, "Apply a mate constraint between two components.", "MATE (face1) (face2)"},
		{"Assembly.Flush", []string{"FLUSH"}, "Apply a flush constraint between two faces.", "FLUSH (face1) (face2)"},
		{"Assembly.Angle", []string{"ASMANGLE"}, "Apply an angle constraint between two components.", "ASMANGLE (face1) (face2) 45"},
		{"Assembly.Tangent", []string{"ASMTANGENT"}, "Apply a tangent constraint between two components.", "ASMTANGENT (face) (cyl)"},
		{"Assembly.Insert", []string{"ASMINSERT"}, "Apply an insert constraint (aligns axis and plane).", "ASMINSERT (edge1) (edge2)"},
		{"Assembly.Symmetry", []string{"ASMSYMMETRY"}, "Apply a symmetry constraint about a plane.", "ASMSYMMETRY (c1) (c2) (plane)"},
		{"Assembly.Transitional", []string{"ASMTRANSITIONAL"}, "Apply a transitional (cam) constraint.", "ASMTRANSITIONAL (face) (face)"},
		{"Assembly.Custom", []string{"ASMCUSTOM"}, "Apply a custom assembly constraint.", "ASMCUSTOM (face1) (face2)"},
	}
}

// assemblyJointVocabulary maps the joint family.
func assemblyJointVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Assembly.JointRigid", []string{"JOINTRIGID"}, "Create a rigid joint between components.", "JOINTRIGID (origin1) (origin2)"},
		{"Assembly.JointRotational", []string{"JOINTROTATIONAL"}, "Create a rotational joint.", "JOINTROTATIONAL (origin1) (origin2)"},
		{"Assembly.JointSlider", []string{"JOINTSLIDER"}, "Create a slider joint.", "JOINTSLIDER (origin1) (origin2)"},
		{"Assembly.JointCylindrical", []string{"JOINTCYLINDRICAL"}, "Create a cylindrical joint.", "JOINTCYLINDRICAL (origin1) (origin2)"},
		{"Assembly.JointPlanar", []string{"JOINTPLANAR"}, "Create a planar joint.", "JOINTPLANAR (origin1) (origin2)"},
		{"Assembly.JointBall", []string{"JOINTBALL"}, "Create a ball joint.", "JOINTBALL (origin1) (origin2)"},
	}
}

// assemblyComponentVocabulary maps component placement, copy/mirror/pattern and motion.
func assemblyComponentVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Assembly.Place", []string{"PLACE"}, "Place an existing component into the assembly.", "PLACE (file) 0,0,0"},
		{"Assembly.Copy", []string{"ASMCOPY"}, "Copy an assembly component.", "ASMCOPY (component)"},
		{"Assembly.Mirror", []string{"ASMMIRROR"}, "Mirror assembly components about a plane.", "ASMMIRROR (component) (plane)"},
		{"Assembly.RectangularPattern", []string{"ASMRECTPATTERN"}, "Pattern components in rows and columns.", "ASMRECTPATTERN (component) 3 2"},
		{"Assembly.CircularPattern", []string{"ASMCIRCLEPATTERN"}, "Pattern components around an axis.", "ASMCIRCLEPATTERN (component) (axis) 6"},
		{"Assembly.Drive", []string{"DRIVE"}, "Drive a constraint through a range to animate motion.", "DRIVE (constraint) 0 90"},
		{"Assembly.GripSnap", []string{"GRIPSNAP"}, "Move a component interactively with grip snap.", "GRIPSNAP (component)"},
		{"Assembly.RotateRotate", []string{"ASMROTATEROTATE"}, "Apply a rotation-rotation motion constraint.", "ASMROTATEROTATE (face1) (face2)"},
		{"Assembly.RotateTranslate", []string{"ASMROTATETRANSLATE"}, "Apply a rotation-translation motion constraint.", "ASMROTATETRANSLATE (face1) (face2)"},
		{"Assembly.TranslateTranslate", []string{"ASMTRANSLATETRANSLATE"}, "Apply a translation-translation motion constraint.", "ASMTRANSLATETRANSLATE (face1) (face2)"},
	}
}

// assemblyManageVocabulary maps interference, BOM, contacts, representations and in-place
// feature creation.
func assemblyManageVocabulary() []vocabEntry {
	var v []vocabEntry
	v = append(v, assemblyAnalysisVocabulary()...)
	v = append(v, assemblyFeatureVocabulary()...)
	return v
}

// assemblyAnalysisVocabulary maps interference/BOM/contact and the representation captures.
func assemblyAnalysisVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Assembly.Interference", []string{"INTERFERE"}, "Analyse interference between component sets.", "INTERFERE (set1) (set2)"},
		{"Assembly.BOM", []string{"BOM"}, "Open the Bill of Materials.", "BOM"},
		{"Assembly.ContactSet", []string{"CONTACTSET"}, "Add components to the contact set.", "CONTACTSET (components)"},
		{"Assembly.ContactEnable", []string{"CONTACTSOLVER"}, "Toggle the contact solver on/off.", "CONTACTSOLVER"},
		{"Assembly.NewModelState", []string{"MODELSTATE"}, "Create a new model state.", "MODELSTATE"},
		{"Assembly.CaptureLOD", []string{"CAPTURELOD"}, "Capture a level-of-detail representation.", "CAPTURELOD"},
		{"Assembly.CapturePosition", []string{"CAPTUREPOSITION"}, "Capture a positional representation.", "CAPTUREPOSITION"},
		{"Assembly.CaptureView", []string{"CAPTUREVIEW"}, "Capture a view representation.", "CAPTUREVIEW"},
	}
}

// assemblyFeatureVocabulary maps features authored in the assembly context (ASM-prefixed so
// they stay distinct from the part-level feature words).
func assemblyFeatureVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Assembly.CreateSketch", []string{"ASMSKETCH"}, "Create a sketch in the assembly context.", "ASMSKETCH (plane)"},
		{"Assembly.Extrude", []string{"ASMEXTRUDE"}, "Extrude in the assembly context.", "ASMEXTRUDE (profile) 10"},
		{"Assembly.Revolve", []string{"ASMREVOLVE"}, "Revolve in the assembly context.", "ASMREVOLVE (profile) (axis) 360"},
		{"Assembly.Hole", []string{"ASMHOLE"}, "Place an assembly-context hole.", "ASMHOLE (point) 6"},
		{"Assembly.Fillet", []string{"ASMFILLET"}, "Round edges in the assembly context.", "ASMFILLET (edges) 2"},
		{"Assembly.Chamfer", []string{"ASMCHAMFER"}, "Bevel edges in the assembly context.", "ASMCHAMFER (edges) 2"},
	}
}

// sheetMetalVocabulary maps the sheet-metal environment. AutoCAD-core has no sheet metal, so
// these are Oblikovati-natural words (SM-prefixed where a bare word collides).
func sheetMetalVocabulary() []vocabEntry {
	return append(sheetMetalCreateVocabulary(), sheetMetalModifyVocabulary()...)
}

// sheetMetalCreateVocabulary maps the wall/flange creation features.
func sheetMetalCreateVocabulary() []vocabEntry {
	return []vocabEntry{
		{"SheetMetal.Face", []string{"SMFACE"}, "Create the base face of a sheet-metal part.", "SMFACE (profile)"},
		{"SheetMetal.Flange", []string{"FLANGE"}, "Add a flange to an edge.", "FLANGE (edge) 20"},
		{"SheetMetal.ContourFlange", []string{"CONTOURFLANGE"}, "Create a contour flange from an open profile.", "CONTOURFLANGE (profile) (edge)"},
		{"SheetMetal.LoftedFlange", []string{"LOFTEDFLANGE"}, "Create a lofted flange between two profiles.", "LOFTEDFLANGE (profile1) (profile2)"},
		{"SheetMetal.ContourRoll", []string{"CONTOURROLL"}, "Roll a contour into a sheet-metal part.", "CONTOURROLL (profile) (axis)"},
		{"SheetMetal.Hem", []string{"HEM"}, "Add a hem to an edge.", "HEM (edge)"},
		{"SheetMetal.Lip", []string{"LIP"}, "Add a lip to an edge.", "LIP (edge)"},
	}
}

// sheetMetalModifyVocabulary maps the bend/corner/cut and flat-pattern features.
func sheetMetalModifyVocabulary() []vocabEntry {
	return []vocabEntry{
		{"SheetMetal.Bend", []string{"SMBEND"}, "Add a bend between two faces.", "SMBEND (edge)"},
		{"SheetMetal.Fold", []string{"FOLD"}, "Fold a face along a sketched line.", "FOLD (line)"},
		{"SheetMetal.Cut", []string{"SMCUT"}, "Cut through sheet-metal faces.", "SMCUT (profile)"},
		{"SheetMetal.Corner", []string{"CORNER"}, "Apply a corner round or chamfer.", "CORNER (corner) 2"},
		{"SheetMetal.CornerSeam", []string{"CORNERSEAM"}, "Create a corner seam between two flanges.", "CORNERSEAM (edge1) (edge2)"},
		{"SheetMetal.CosmeticBend", []string{"COSMETICBEND"}, "Add a cosmetic bend line.", "COSMETICBEND (line)"},
		{"SheetMetal.Rip", []string{"RIP"}, "Rip an edge to open a corner.", "RIP (edge)"},
		{"SheetMetal.Punch", []string{"PUNCH"}, "Apply a punch tool (iFeature).", "PUNCH (point)"},
		{"SheetMetal.Refold", []string{"REFOLD"}, "Refold a flattened model.", "REFOLD"},
		{"SheetMetal.Convert", []string{"SMCONVERT"}, "Convert a solid to sheet metal.", "SMCONVERT (face) 1"},
		{"SheetMetal.CreateFlatPattern", []string{"FLATPATTERN"}, "Create the flat pattern.", "FLATPATTERN"},
		{"SheetMetal.Style", []string{"SMSTYLE"}, "Edit sheet-metal styles and rules.", "SMSTYLE"},
	}
}

// drawingVocabulary maps the drawing-from-model environment. The view words are AutoCAD's
// real VIEW* commands; sheet/standard management is Oblikovati-natural.
func drawingVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Drawing.BaseView", []string{"VIEWBASE"}, "Place a base view from a 3D model.", "VIEWBASE (model) (point)"},
		{"Drawing.ProjectedView", []string{"VIEWPROJ"}, "Place a projected view from a parent view.", "VIEWPROJ (parent) (point)"},
		{"Drawing.SectionView", []string{"VIEWSECTION"}, "Create a section view.", "VIEWSECTION (parent) (line)"},
		{"Drawing.DetailView", []string{"VIEWDETAIL"}, "Create a detail view.", "VIEWDETAIL (parent) (centre) (radius)"},
		{"Drawing.BreakView", []string{"VIEWBREAK"}, "Add a break to a view.", "VIEWBREAK (view) (p1) (p2)"},
		{"Drawing.AuxiliaryView", []string{"VIEWAUX"}, "Create an auxiliary view from an edge.", "VIEWAUX (parent) (edge)"},
		{"Drawing.NewSheet", []string{"NEWSHEET", "LAYOUT"}, "Add a new drawing sheet.", "NEWSHEET"},
		{"Drawing.DeleteSheet", []string{"DELETESHEET"}, "Delete the active drawing sheet.", "DELETESHEET"},
		{"Drawing.DraftingStandard", []string{"DRAFTINGSTANDARD"}, "Set the drafting standard.", "DRAFTINGSTANDARD"},
		{"Drawing.ModelReference", []string{"MODELREFERENCE"}, "Set the referenced model.", "MODELREFERENCE (file)"},
		{"Drawing.ExportDXF", []string{"DXFOUT", "EXPORT"}, "Export the drawing/flat-pattern to DXF.", "DXFOUT"},
	}
}

// sketch3DVocabulary maps the 3D-sketch environment (3D suffix on words that collide with the
// 2D sketch and part features).
func sketch3DVocabulary() []vocabEntry {
	return append(sketch3DDrawVocabulary(), sketch3DEditVocabulary()...)
}

// sketch3DDrawVocabulary maps 3D curve creation.
func sketch3DDrawVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Sketch3D.Line", []string{"LINE3D"}, "Draw a 3D sketch line or polyline.", "LINE3D 0,0,0 10,0,5"},
		{"Sketch3D.Arc", []string{"ARC3D"}, "Draw a 3D sketch arc.", "ARC3D 0,0,0 5,5,0 10,0,0"},
		{"Sketch3D.Circle", []string{"CIRCLE3D"}, "Draw a 3D sketch circle.", "CIRCLE3D 0,0,0 5"},
		{"Sketch3D.Point", []string{"POINT3D"}, "Place a 3D sketch point.", "POINT3D 5,5,5"},
		{"Sketch3D.Spline", []string{"SPLINE3D"}, "Draw a 3D interpolated spline.", "SPLINE3D 0,0,0 5,5,5"},
		{"Sketch3D.SplineFit", []string{"SPLINEFIT3D"}, "Draw a 3D fit-point spline.", "SPLINEFIT3D 0,0,0 5,5,5"},
		{"Sketch3D.ControlPointSpline", []string{"CVSPLINE3D"}, "Draw a 3D control-vertex spline.", "CVSPLINE3D 0,0,0 5,5,5"},
		{"Sketch3D.Helix", []string{"HELIX3D"}, "Draw a 3D helical curve.", "HELIX3D (axis) 10 5"},
		{"Sketch3D.Helical", []string{"HELICAL3D"}, "Draw a 3D helical curve by revolution and pitch.", "HELICAL3D (axis) 2 20"},
		{"Sketch3D.EquationCurve", []string{"EQUATIONCURVE"}, "Draw a curve from parametric equations.", "EQUATIONCURVE"},
		{"Sketch3D.SurfaceCurve", []string{"SURFACECURVE"}, "Draw a curve lying on a surface.", "SURFACECURVE (surface)"},
	}
}

// sketch3DEditVocabulary maps 3D-sketch editing/conditions and the lifecycle.
func sketch3DEditVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Sketch3D.Bend", []string{"BEND3D"}, "Add a bend between two 3D sketch lines.", "BEND3D (line1) (line2) 5"},
		{"Sketch3D.Include", []string{"INCLUDE3D"}, "Include model geometry in the 3D sketch.", "INCLUDE3D (edges)"},
		{"Sketch3D.Dimension", []string{"DIMENSION3D"}, "Dimension 3D sketch geometry.", "DIMENSION3D (select) 25"},
		{"Sketch3D.Tangent", []string{"TANGENT3D"}, "Apply a 3D tangent condition.", "TANGENT3D (curve) (curve)"},
		{"Sketch3D.Smooth", []string{"SMOOTH3D"}, "Apply a 3D smooth condition.", "SMOOTH3D (spline) (curve)"},
		{"Sketch3D.Finish", []string{"FINISH3D"}, "Finish the active 3D sketch.", "FINISH3D"},
	}
}

// manageVocabulary maps the model-management commands (PARAMETERS is AutoCAD's real command).
func manageVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Manage.Parameters", []string{"PARAMETERS"}, "Open the Parameters manager.", "PARAMETERS"},
		{"Manage.Derive", []string{"DERIVE"}, "Derive geometry from another part or assembly.", "DERIVE (file)"},
		{"Manage.Shrinkwrap", []string{"SHRINKWRAP"}, "Create a shrinkwrap substitute.", "SHRINKWRAP"},
		{"Manage.ScriptConsole", []string{"SCRIPT", "RUNSCRIPT"}, "Open the script console / run a script.", "SCRIPT"},
	}
}

// viewVocabulary maps the navigation, orientation and visual-style operations. Per-preset
// display toggles (lighting/environment/colour-scheme presets, shadows, layouts) are GUI
// settings, not modelling operations, so they are intentionally not command words.
func viewVocabulary() []vocabEntry {
	return append(viewNavigateVocabulary(), viewStyleVocabulary()...)
}

// viewNavigateVocabulary maps zoom/home/named-views and the orthographic orientations.
func viewNavigateVocabulary() []vocabEntry {
	return []vocabEntry{
		{"View.ZoomAll", []string{"ZOOM", "ZOOMALL", "ZE"}, "Zoom to fit all geometry.", "ZOOM"},
		{"View.Home", []string{"HOME", "HOMEVIEW"}, "Go to the home (isometric) view.", "HOME"},
		{"View.NamedViews", []string{"NAMEDVIEWS", "VIEWMGR"}, "Open the named-views manager.", "NAMEDVIEWS"},
		{"View.ViewCube", []string{"VIEWCUBE"}, "Toggle the ViewCube.", "VIEWCUBE"},
		{"View.Orient.Front", []string{"FRONT"}, "Orient to the front view.", "FRONT"},
		{"View.Orient.Back", []string{"BACK"}, "Orient to the back view.", "BACK"},
		{"View.Orient.Top", []string{"TOPVIEW"}, "Orient to the top view.", "TOPVIEW"},
		{"View.Orient.Bottom", []string{"BOTTOMVIEW"}, "Orient to the bottom view.", "BOTTOMVIEW"},
		{"View.Orient.Left", []string{"LEFTVIEW"}, "Orient to the left view.", "LEFTVIEW"},
		{"View.Orient.Right", []string{"RIGHTVIEW"}, "Orient to the right view.", "RIGHTVIEW"},
		{"View.Orient.Iso", []string{"ISO"}, "Orient to the isometric view.", "ISO"},
	}
}

// viewStyleVocabulary maps the visual-style operations (AutoCAD's VSCURRENT styles).
func viewStyleVocabulary() []vocabEntry {
	return []vocabEntry{
		{"View.Wireframe", []string{"WIREFRAME"}, "Set the wireframe visual style.", "WIREFRAME"},
		{"View.Shaded", []string{"SHADED"}, "Set the shaded visual style.", "SHADED"},
		{"View.ShadedWithEdges", []string{"SHADEDEDGES"}, "Set the shaded-with-edges visual style.", "SHADEDEDGES"},
		{"View.Realistic", []string{"REALISTIC"}, "Set the realistic visual style.", "REALISTIC"},
		{"View.Monochrome", []string{"MONOCHROME"}, "Set the monochrome visual style.", "MONOCHROME"},
		{"View.Illustration", []string{"ILLUSTRATION"}, "Set the illustration visual style.", "ILLUSTRATION"},
		{"View.Watercolor", []string{"WATERCOLOR"}, "Set the watercolour visual style.", "WATERCOLOR"},
		{"View.TechnicalIllustration", []string{"TECHNICAL"}, "Set the technical-illustration visual style.", "TECHNICAL"},
	}
}

// moldFreeformMeshVocabulary maps the mold, freeform (T-spline) and mesh operations. These
// have no AutoCAD-core equivalent, so they use Oblikovati-natural words.
func moldFreeformMeshVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Mold.CoreCavity", []string{"CORECAVITY"}, "Generate core and cavity from a part.", "CORECAVITY (part)"},
		{"Freeform.Box", []string{"FREEFORMBOX"}, "Create a freeform (T-spline) box.", "FREEFORMBOX 0,0,0 10,10,10"},
		{"Freeform.Plane", []string{"FREEFORMPLANE"}, "Create a freeform (T-spline) plane.", "FREEFORMPLANE (plane) 10 10"},
		{"Freeform.QuadBall", []string{"QUADBALL"}, "Create a freeform (T-spline) quadball.", "QUADBALL 0,0,0 5"},
		{"Mesh.Place", []string{"MESHPLACE", "IMPORTMESH"}, "Import or place a mesh body.", "MESHPLACE (file)"},
	}
}
