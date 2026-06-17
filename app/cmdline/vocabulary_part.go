// SPDX-License-Identifier: GPL-2.0-only

package cmdline

// partVocabulary maps AutoCAD 3D/solid commands onto Oblikovati part features. The 3D edge
// blends use AutoCAD's solid-editing names (FILLETEDGE/CHAMFEREDGE) so they stay distinct
// from the 2D sketch FILLET/CHAMFER.
func partVocabulary() []vocabEntry {
	return append(partCreateVocabulary(), partModifyVocabulary()...)
}

// partCreateVocabulary maps the solid-creation features.
func partCreateVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Create.Extrude", []string{"EXTRUDE", "EXT"}, "Extrude a profile into a solid.", "EXTRUDE (profile) 10"},
		{"Create.Revolve", []string{"REVOLVE", "REV"}, "Revolve a profile about an axis.", "REVOLVE (profile) (axis) 360"},
		{"Create.Sweep", []string{"SWEEP", "SW"}, "Sweep a profile along a path.", "SWEEP (profile) (path)"},
		{"Create.Loft", []string{"LOFT"}, "Loft between two or more profiles.", "LOFT (profile1) (profile2)"},
		{"Create.Coil", []string{"HELIX", "COIL"}, "Create a helical coil from a profile.", "HELIX (profile) (axis)"},
		{"Create.Rib", []string{"RIB"}, "Create a rib/web from an open profile.", "RIB (profile) 2"},
		{"Create.Emboss", []string{"EMBOSS"}, "Emboss or engrave sketch geometry onto a face.", "EMBOSS (profile) 1"},
		{"Create.Decal", []string{"DECAL"}, "Apply a decal image to a face.", "DECAL (face) (image)"},
		{"Create.Grill", []string{"GRILL"}, "Create a grill feature in a plastic part.", "GRILL (boundary)"},
	}
}

// partModifyVocabulary maps the solid-editing features. Boolean/slice/press-pull use the
// real AutoCAD command names (UNION/SUBTRACT/INTERSECT, SLICE, PRESSPULL).
func partModifyVocabulary() []vocabEntry {
	return append(partDressUpVocabulary(), partBodyEditVocabulary()...)
}

// partDressUpVocabulary maps the face/edge dress-up features.
func partDressUpVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Modify.Hole", []string{"HOLE"}, "Place a hole feature.", "HOLE (point) 6"},
		{"Modify.Boss", []string{"BOSS"}, "Create a plastic boss feature.", "BOSS (point)"},
		{"Modify.Fillet", []string{"FILLETEDGE"}, "Round one or more solid edges.", "FILLETEDGE (edges) 2"},
		{"Modify.Chamfer", []string{"CHAMFEREDGE"}, "Bevel one or more solid edges.", "CHAMFEREDGE (edges) 2"},
		{"Modify.Shell", []string{"SHELL"}, "Hollow a solid to a wall thickness.", "SHELL (faces) 2"},
		{"Modify.Thread", []string{"THREAD"}, "Apply a thread to a cylindrical face.", "THREAD (face) M6"},
		{"Modify.Thicken", []string{"THICKEN"}, "Thicken a surface into a solid.", "THICKEN (surface) 2"},
		{"Modify.Draft", []string{"DRAFT", "TAPER"}, "Apply a draft angle to faces.", "DRAFT (faces) (pull) 3"},
		{"Modify.FaceOffset", []string{"OFFSETFACE"}, "Offset a solid face.", "OFFSETFACE (face) 2"},
		{"Modify.DeleteFace", []string{"DELETEFACE"}, "Delete a face from a solid.", "DELETEFACE (face)"},
		{"Modify.ReplaceFace", []string{"REPLACEFACE"}, "Replace a solid face with a surface.", "REPLACEFACE (face) (surface)"},
		{"Modify.MoveFace", []string{"MOVEFACE"}, "Move a solid face.", "MOVEFACE (face) 0,0 0,2"},
	}
}

// partBodyEditVocabulary maps the whole-body edits: booleans, slice, press-pull, body
// transforms, and feature patterns.
func partBodyEditVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Modify.Combine", []string{"COMBINE", "UNION", "SUBTRACT", "INTERSECT"}, "Boolean-combine bodies (union, cut or intersect).", "UNION (body1) (body2)"},
		{"Modify.Split", []string{"SLICE", "SPLITBODY"}, "Split a body with a plane or surface.", "SLICE (body) (plane)"},
		{"Modify.DirectEdit", []string{"DIRECTEDIT", "PRESSPULL"}, "Directly push/pull or edit solid geometry.", "PRESSPULL (face) 5"},
		{"Modify.MoveBodies", []string{"MOVEBODY", "3DMOVE"}, "Move solid bodies by a vector.", "3DMOVE (body) 0,0,0 0,0,10"},
		{"Modify.Mirror", []string{"MIRRORBODY"}, "Mirror a solid body about a plane.", "MIRRORBODY (body) (plane)"},
		{"Modify.RectangularPattern", []string{"RECTPATTERN"}, "Pattern features/bodies in rows and columns.", "RECTPATTERN (feature) 3 2"},
		{"Modify.CircularPattern", []string{"CIRCLEPATTERN"}, "Pattern features/bodies around an axis.", "CIRCLEPATTERN (feature) (axis) 6"},
		{"Modify.SketchDrivenPattern", []string{"SKETCHPATTERN"}, "Pattern features at sketch points.", "SKETCHPATTERN (feature) (sketch)"},
		{"Modify.Hull", []string{"HULL"}, "Create the convex hull of selected bodies.", "HULL (body1) (body2)"},
	}
}

// surfaceVocabulary maps AutoCAD surface commands (SURF*) onto Oblikovati surface features.
func surfaceVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Surface.Patch", []string{"SURFPATCH", "PATCH"}, "Create a surface patch from boundary edges.", "SURFPATCH (edges)"},
		{"Surface.Trim", []string{"SURFTRIM"}, "Trim a surface with a cutting object.", "SURFTRIM (surface) (curve)"},
		{"Surface.Extend", []string{"SURFEXTEND"}, "Extend a surface edge.", "SURFEXTEND (edge) 5"},
		{"Surface.Offset", []string{"SURFOFFSET"}, "Offset a surface by a distance.", "SURFOFFSET (surface) 2"},
		{"Surface.Ruled", []string{"RULESURF", "RULED"}, "Create a ruled surface between two curves.", "RULESURF (curve1) (curve2)"},
		{"Surface.Stitch", []string{"SURFSTITCH", "STITCH"}, "Stitch surfaces into a quilt or solid.", "SURFSTITCH (surfaces)"},
		{"Surface.Sculpt", []string{"SCULPT"}, "Sculpt a solid from bounding surfaces.", "SCULPT (surfaces)"},
		{"Surface.MidSurface", []string{"MIDSURFACE"}, "Create a midsurface between face pairs.", "MIDSURFACE (faces)"},
	}
}

// workplaneVocabulary maps the work-plane (datum) constructors. AutoCAD has no direct
// equivalent, so these use Oblikovati-natural words.
func workplaneVocabulary() []vocabEntry {
	return []vocabEntry{
		{"WorkPlane.Offset", []string{"WORKPLANE", "PLANE"}, "Create a work plane offset from a face/plane.", "WORKPLANE (face) 10"},
		{"WorkPlane.Midplane", []string{"MIDPLANE"}, "Create a work plane midway between two faces.", "MIDPLANE (face1) (face2)"},
		{"WorkPlane.ThreePoints", []string{"PLANE3P"}, "Create a work plane through three points.", "PLANE3P (p1) (p2) (p3)"},
		{"WorkPlane.Tangent", []string{"TANPLANE"}, "Create a work plane tangent to a face.", "TANPLANE (face) (plane)"},
		{"WorkPlane.NormalToAxis", []string{"NORMALPLANE"}, "Create a work plane normal to an axis at a point.", "NORMALPLANE (axis) (point)"},
	}
}

// appVocabulary maps the application-wide editing/file actions (built-in action ids and the
// new-document / close commands).
func appVocabulary() []vocabEntry {
	return []vocabEntry{
		{"file.save", []string{"SAVE", "QSAVE"}, "Save the active document.", "SAVE"},
		{"edit.undo", []string{"UNDO"}, "Undo the last operation.", "UNDO"},
		{"edit.redo", []string{"REDO", "MREDO"}, "Redo the last undone operation.", "REDO"},
		{"tool.cancel", []string{"CANCEL"}, "Cancel the active command.", "CANCEL"},
		{"GetStarted.NewPart", []string{"NEW", "NEWPART"}, "Create a new part document.", "NEW"},
		{"GetStarted.NewAssembly", []string{"NEWASSEMBLY"}, "Create a new assembly document.", "NEWASSEMBLY"},
		{"GetStarted.NewDrawing", []string{"NEWDRAWING"}, "Create a new drawing document.", "NEWDRAWING"},
		{"GetStarted.NewSheetMetalPart", []string{"NEWSHEETMETAL"}, "Create a new sheet-metal part document.", "NEWSHEETMETAL"},
		{"View.Close", []string{"CLOSE"}, "Close the active document.", "CLOSE"},
	}
}
