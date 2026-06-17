// SPDX-License-Identifier: GPL-2.0-only

package cmdline

// sketchVocabulary maps AutoCAD 2D drawing/modify/constraint commands (and the sketch
// lifecycle) onto Oblikovati sketch tools, following acad.pgp aliases where they fit.
func sketchVocabulary() []vocabEntry {
	var v []vocabEntry
	v = append(v, sketchDrawVocabulary()...)
	v = append(v, sketchModifyVocabulary()...)
	v = append(v, sketchConstraintVocabulary()...)
	return v
}

// sketchDrawVocabulary maps the 2D geometry-creation commands.
func sketchDrawVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Sketch.Line", []string{"LINE"}, "Draw a line or connected polyline in the active sketch.", "LINE 0,0 10,0 10,5"},
		{"Sketch.Circle", []string{"CIRCLE"}, "Draw a circle by centre and radius.", "CIRCLE 0,0 5"},
		{"Sketch.Arc", []string{"ARC"}, "Draw an arc through points or by centre.", "ARC 0,0 5,5 10,0"},
		{"Sketch.Rectangle", []string{"RECTANG", "RECTANGLE", "REC"}, "Draw a rectangle by two opposite corners.", "RECTANG 0,0 10,5"},
		{"Sketch.Point", []string{"POINT", "PO"}, "Place a sketch point.", "POINT 5,5"},
		{"Sketch.Polygon", []string{"POLYGON", "POL"}, "Draw a regular polygon by edge count and radius.", "POLYGON 6 0,0 5"},
		{"Sketch.Ellipse", []string{"ELLIPSE", "EL"}, "Draw an ellipse by centre and two axes.", "ELLIPSE 0,0 10 5"},
		{"Sketch.Spline", []string{"SPLINE", "SPL"}, "Draw an interpolated spline through points.", "SPLINE 0,0 5,5 10,0"},
		{"Sketch.Slot", []string{"SLOT"}, "Draw a straight slot by two centres and a width.", "SLOT 0,0 10,0 2"},
		{"Sketch.Text", []string{"TEXT", "MTEXT"}, "Place sketch text.", "TEXT 0,0"},
		{"Sketch.Dimension", []string{"DIMENSION", "DIM"}, "Add a parametric dimension to sketch geometry.", "DIM (select) 25"},
		{"Sketch.AutoDimension", []string{"AUTODIMENSION", "AUTODIM"}, "Auto-dimension the sketch to remove its degrees of freedom.", "AUTODIM (select sketch)"},
		{"Sketch.Centerline", []string{"CENTERLINE"}, "Draw a centerline (excluded from profiles; consumed by mirror/revolve).", "CENTERLINE 0,0 10,0"},
		{"Sketch.Construction", []string{"CONSTRUCTION"}, "Toggle selected sketch geometry to construction.", "CONSTRUCTION (select)"},
		{"Sketch.Project", []string{"PROJECT", "PROJECTGEOMETRY"}, "Project model edges/vertices into the active sketch.", "PROJECT (select edges)"},
		{"Sketch.CreateBlock", []string{"BLOCK"}, "Create a sketch block from selected geometry.", "BLOCK (select)"},
		{"Sketch.PlaceBlock", []string{"INSERT"}, "Insert a sketch block instance.", "INSERT (block) 5,5"},
		{"Sketch.Create2D", []string{"SKETCH2D", "NEWSKETCH"}, "Start a new 2D sketch on a selected plane/face.", "SKETCH2D (select plane)"},
		{"Sketch.Create3D", []string{"SKETCH3D"}, "Start a new 3D sketch.", "SKETCH3D"},
		{"Sketch.Finish", []string{"FINISHSKETCH", "FINISH"}, "Finish the active sketch.", "FINISH"},
	}
}

// sketchModifyVocabulary maps the 2D editing commands.
func sketchModifyVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Sketch.Fillet", []string{"FILLET"}, "Round a corner between two sketch entities.", "FILLET (two lines) 2"},
		{"Sketch.Chamfer", []string{"CHAMFER", "CHA"}, "Bevel a corner between two sketch entities.", "CHAMFER (two lines) 2"},
		{"Sketch.Trim", []string{"TRIM", "TR"}, "Trim sketch geometry to the nearest intersection.", "TRIM (segment)"},
		{"Sketch.Extend", []string{"EXTEND", "EX"}, "Extend sketch geometry to a boundary.", "EXTEND (segment)"},
		{"Sketch.Offset", []string{"OFFSET"}, "Offset selected sketch geometry by a distance.", "OFFSET 2 (select)"},
		{"Sketch.Mirror", []string{"MIRROR", "MI"}, "Mirror sketch geometry about a centerline.", "MIRROR (select) (line)"},
		{"Sketch.Move", []string{"MOVE"}, "Move sketch geometry by a vector.", "MOVE (select) 0,0 10,0"},
		{"Sketch.Copy", []string{"COPY", "CO", "CP"}, "Copy sketch geometry by a vector.", "COPY (select) 0,0 10,0"},
		{"Sketch.Rotate", []string{"ROTATE", "RO"}, "Rotate sketch geometry about a point.", "ROTATE (select) 0,0 90"},
		{"Sketch.Scale", []string{"SCALE", "SC"}, "Scale sketch geometry about a point.", "SCALE (select) 0,0 2"},
		{"Sketch.Stretch", []string{"STRETCH"}, "Stretch sketch geometry by moving vertices.", "STRETCH (select) 0,0 5,0"},
		{"Sketch.Split", []string{"BREAK", "BR"}, "Split a sketch entity at a point.", "BREAK (entity) (point)"},
		{"Sketch.RectangularPattern", []string{"ARRAYRECT"}, "Pattern sketch geometry in rows and columns.", "ARRAYRECT (select) 3 2"},
		{"Sketch.CircularPattern", []string{"ARRAYPOLAR"}, "Pattern sketch geometry around a centre.", "ARRAYPOLAR (select) 0,0 6"},
	}
}

// sketchConstraintVocabulary maps AutoCAD's parametric geometric constraints (GC*) onto the
// Oblikovati sketch constraints — a clean 1:1 mapping.
func sketchConstraintVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Sketch.Coincident", []string{"GCCOINCIDENT"}, "Constrain two points (or a point and a curve) coincident.", "GCCOINCIDENT (A) (B)"},
		{"Sketch.Collinear", []string{"GCCOLLINEAR"}, "Constrain two lines collinear.", "GCCOLLINEAR (line1) (line2)"},
		{"Sketch.Concentric", []string{"GCCONCENTRIC"}, "Constrain two arcs/circles concentric.", "GCCONCENTRIC (c1) (c2)"},
		{"Sketch.Equal", []string{"GCEQUAL"}, "Constrain two entities to equal size.", "GCEQUAL (a) (b)"},
		{"Sketch.Fix", []string{"GCFIX"}, "Fix a point or curve in place.", "GCFIX (entity)"},
		{"Sketch.Horizontal", []string{"GCHORIZONTAL"}, "Constrain a line horizontal.", "GCHORIZONTAL (line)"},
		{"Sketch.Vertical", []string{"GCVERTICAL"}, "Constrain a line vertical.", "GCVERTICAL (line)"},
		{"Sketch.Parallel", []string{"GCPARALLEL"}, "Constrain two lines parallel.", "GCPARALLEL (line1) (line2)"},
		{"Sketch.Perpendicular", []string{"GCPERPENDICULAR"}, "Constrain two lines perpendicular.", "GCPERPENDICULAR (line1) (line2)"},
		{"Sketch.Tangent", []string{"GCTANGENT"}, "Constrain a curve tangent to another.", "GCTANGENT (curve) (line)"},
		{"Sketch.Symmetric", []string{"GCSYMMETRIC"}, "Constrain two entities symmetric about a line.", "GCSYMMETRIC (a) (b) (line)"},
		{"Sketch.Smooth", []string{"GCSMOOTH"}, "Constrain a spline smooth (curvature-continuous) to a curve.", "GCSMOOTH (spline) (curve)"},
	}
}
