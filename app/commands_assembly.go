// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/model/assembly"
)

// The Assemble ribbon tab (M11/M12), shown for an assembly document (the AssemblyRibbon key,
// switched in by ribbonKeyForDocument). It carries the Component panel (Place, #763), the
// Relationships panel (the M12-F01 constraints — mate/flush/angle/…, #770), the Pattern panel
// (Rectangular/Circular Pattern, Mirror, Copy, #765), and the BOM panel (Bill of Materials,
// #768). The assembly model + wire surface already exist — these commands are the head/app
// wiring that drives them.

// assemblyTabCommands returns the Assemble tab commands in ribbon order, grouped into panels
// by their category (Component / Relationships / Modify / Pattern / BOM).
func assemblyTabCommands() []*CommandDefinition {
	cmds := []*CommandDefinition{
		placeComponentCommand(),
	}
	cmds = append(cmds, relationshipCommands()...)
	cmds = append(cmds, jointCommands()...)
	cmds = append(cmds, representationCommands()...)
	cmds = append(cmds, contactCommands()...)
	return append(cmds, assemblyModelingCommands()...)
}

// jointCommands returns the Joints-panel commands — one per M12-F02 joint kind. Each starts
// an [AssemblyJointTool] that picks the two joint-origin faces and creates the joint. Compact
// icon buttons, like the Relationships panel.
func jointCommands() []*CommandDefinition {
	return []*CommandDefinition{
		jointCommand("Assembly.JointRigid", "Rigid", "joint-rigid", "Rigid joint — fix two components together (0 DOF).",
			func(js *assembly.JointSet, r []assembly.Ref) assembly.Joint { return js.AddRigid(r[0], r[1]) }),
		jointCommand("Assembly.JointRotational", "Rotational", "joint-rotational", "Rotational joint — one rotation about the joint axis (a hinge).",
			func(js *assembly.JointSet, r []assembly.Ref) assembly.Joint { return js.AddRotational(r[0], r[1]) }),
		jointCommand("Assembly.JointSlider", "Slider", "joint-slider", "Slider joint — one translation along the joint axis.",
			func(js *assembly.JointSet, r []assembly.Ref) assembly.Joint { return js.AddSlider(r[0], r[1]) }),
		jointCommand("Assembly.JointCylindrical", "Cylindrical", "joint-cylindrical", "Cylindrical joint — translation along and rotation about the axis (2 DOF).",
			func(js *assembly.JointSet, r []assembly.Ref) assembly.Joint { return js.AddCylindrical(r[0], r[1]) }),
		jointCommand("Assembly.JointPlanar", "Planar", "joint-planar", "Planar joint — two in-plane translations and a rotation (3 DOF).",
			func(js *assembly.JointSet, r []assembly.Ref) assembly.Joint { return js.AddPlanar(r[0], r[1]) }),
		jointCommand("Assembly.JointBall", "Ball", "joint-ball", "Ball joint — three rotations about a common point (3 DOF).",
			func(js *assembly.JointSet, r []assembly.Ref) assembly.Joint { return js.AddBall(r[0], r[1]) }),
		driveCommand(),
	}
}

// jointCommand builds a Joints-panel command that starts a joint tool of one kind.
func jointCommand(id, name, icon, tooltip string, build jointBuild) *CommandDefinition {
	return NewCommand(id, name, "Joints", func(s *Session) error {
		if _, err := activeAssembly(s); err != nil {
			return err
		}
		s.StartTool(NewAssemblyJointTool(name, build))
		return nil
	}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
		WithIcon(icon).WithButtonStyle(CompactIconButton).WithTooltip(tooltip)
}

// assemblyModelingCommands returns the sketch/modify/pattern/BOM commands of the Assemble tab.
func assemblyModelingCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Assembly.CreateSketch", "Create 2D Sketch", "Sketch", func(s *Session) error {
			if _, ok := s.SelectedSketchHostPlane(); ok {
				_, err := s.CreateSketchOnSelectedPlane()
				return err
			}
			s.StartTool(NewCreateSketchTool())
			return nil
		}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(canCreateSketch).
			WithIcon("create-sketch").WithButtonStyle(LargeIconButton).
			WithTooltip("Create 2D Sketch — author a sketch in assembly space (extrude/revolve machine the components)."),
		assemblyToolCommand("Assembly.Extrude", "Extrude", "Modify", "extrude",
			"Extrude — run a sketch profile across the components to add or cut material.",
			func() Tool { return NewAssemblyExtrudeTool() }),
		assemblyToolCommand("Assembly.Revolve", "Revolve", "Modify", "revolve",
			"Revolve — spin a sketch profile about an axis across the components.",
			func() Tool { return NewAssemblyRevolveTool() }),
		assemblyToolCommand("Assembly.Hole", "Hole", "Modify", "hole",
			"Hole — drill a parametric bore (centre, axis, diameter, depth) through the components.",
			func() Tool { return NewAssemblyHoleTool() }),
		assemblyToolCommand("Assembly.RectangularPattern", "Rectangular Pattern", "Pattern", "rectangular-pattern",
			"Rectangular Pattern — replicate the selected component on a grid (counts and spacing).",
			func() Tool { return NewAssemblyRectPatternTool() }),
		assemblyToolCommand("Assembly.CircularPattern", "Circular Pattern", "Pattern", "circular-pattern",
			"Circular Pattern — replicate the selected component around the Z axis (count and angle).",
			func() Tool { return NewAssemblyCircPatternTool() }),
		assemblyToolCommand("Assembly.Mirror", "Mirror", "Pattern", "mirror",
			"Mirror — add a mirror of the selected components across a plane.",
			func() Tool { return NewAssemblyMirrorTool() }),
		assemblyToolCommand("Assembly.Chamfer", "Chamfer", "Modify", "chamfer",
			"Chamfer — bevel a picked component edge on every instance of that component.",
			func() Tool { return NewAssemblyChamferTool() }),
		assemblyToolCommand("Assembly.Fillet", "Fillet", "Modify", "fillet",
			"Fillet — round a picked component edge on every instance of that component.",
			func() Tool { return NewAssemblyFilletTool() }),
		NewCommand("Assembly.Copy", "Copy", "Pattern", func(s *Session) error { return s.CopyComponents() }).
			WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
			WithIcon("copy").WithButtonStyle(SmallIconButton).
			WithTooltip("Copy — add an independent copy of each selected component."),
		NewCommand("Assembly.BOM", "Bill of Materials", "BOM", func(s *Session) error {
			if _, err := activeAssembly(s); err != nil {
				return err
			}
			s.OpenBOM()
			return nil
		}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
			WithIcon("bom").WithButtonStyle(LargeIconButton).
			WithTooltip("Bill of Materials — list the assembly's components (structured or parts-only) and export to CSV."),
	}
}

// assemblyToolCommand builds an Assemble-tab command that starts a replication tool on the active
// assembly. The tool seeds its sources from the current selection and reads the rest from the
// generic tool-param dialog (#765).
func assemblyToolCommand(id, name, panel, icon, tooltip string, newTool func() Tool) *CommandDefinition {
	return NewCommand(id, name, panel, func(s *Session) error {
		if _, err := activeAssembly(s); err != nil {
			return err
		}
		s.StartTool(newTool())
		return nil
	}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
		WithIcon(icon).WithButtonStyle(SmallIconButton).WithTooltip(tooltip)
}

// relationshipCommands returns the Relationships-panel commands — one per M12-F01 constraint
// kind (#770). Each starts an [AssemblyConstraintTool] that picks the component faces the
// relationship relates and creates it. They are compact icon buttons, like the sketch
// Constrain panel.
func relationshipCommands() []*CommandDefinition {
	return []*CommandDefinition{
		gripSnapCommand(),
		constraintCommand("Assembly.Mate", "Mate", "mate", "Mate — make two component faces coincident (opposed normals).", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddMate(r[0], r[1], 0, types.MateSolutionOpposed)
			}),
		constraintCommand("Assembly.Flush", "Flush", "flush", "Flush — make two component faces coplanar (aligned normals).", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddFlush(r[0], r[1], 0)
			}),
		constraintCommand("Assembly.Angle", "Angle", "angle", "Angle — hold an angle between two component faces.", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddAngle(r[0], r[1], math.Pi/2, types.AngleSolutionUndirected)
			}),
		constraintCommand("Assembly.Tangent", "Tangent", "tangent", "Tangent — keep a face tangent to a cylindrical face.", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddTangent(r[0], r[1], false)
			}),
		constraintCommand("Assembly.Insert", "Insert", "insert", "Insert — collinear axes plus a plane mate (a bolt into a hole).", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddInsert(r[0], r[1], 0, false)
			}),
		constraintCommand("Assembly.Symmetry", "Symmetry", "symmetry", "Symmetry — position two component faces symmetrically about a third.", 3,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddSymmetry(r[0], r[1], r[2])
			}),
		constraintCommand("Assembly.RotateRotate", "Rotate-Rotate", "rotate-rotate", "Rotate-Rotate — couple two rotations by a gear ratio.", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddRotateRotate(r[0], r[1], 1)
			}),
		constraintCommand("Assembly.RotateTranslate", "Rotate-Translate", "rotate-translate", "Rotate-Translate — rack and pinion.", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddRotateTranslate(r[0], r[1], 1)
			}),
		constraintCommand("Assembly.TranslateTranslate", "Translate-Translate", "translate-translate", "Translate-Translate — couple two translations by a ratio.", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddTranslateTranslate(r[0], r[1], 1)
			}),
		constraintCommand("Assembly.Transitional", "Transitional", "transitional", "Transitional — keep a face in sliding contact with a transition face.", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddTransitional(r[0], r[1])
			}),
		constraintCommand("Assembly.Custom", "Custom", "custom", "Custom — register an add-in-solved relationship.", 2,
			func(set *assembly.ConstraintSet, r []assembly.Ref) assembly.Constraint {
				return set.AddCustom(r[0], r[1], "custom", nil)
			}),
	}
}

// gripSnapCommand builds the Grip Snap command: pick a face on the component to move + a target face,
// and the constraint is inferred (or chosen) and the part snaps into place (#794).
func gripSnapCommand() *CommandDefinition {
	return NewCommand("Assembly.GripSnap", "Grip Snap", "Relationships", func(s *Session) error {
		if _, err := activeAssembly(s); err != nil {
			return err
		}
		s.StartTool(NewGripSnapTool())
		return nil
	}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
		WithIcon("grip-snap").WithButtonStyle(CompactIconButton).
		WithTooltip("Grip Snap — pick a face to move and a target face; the snap constraint is inferred and the part snaps into place.")
}

// constraintCommand builds a Relationships-panel command that starts a constraint tool of one
// kind on the active assembly.
func constraintCommand(id, name, icon, tooltip string, need int, build constraintBuild) *CommandDefinition {
	return NewCommand(id, name, "Relationships", func(s *Session) error {
		if _, err := activeAssembly(s); err != nil {
			return err
		}
		s.StartTool(NewAssemblyConstraintTool(name, need, build))
		return nil
	}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
		WithIcon(icon).WithButtonStyle(CompactIconButton).WithTooltip(tooltip)
}

// placeComponentCommand builds the Place command: it starts the modal Place Component tool on
// the active assembly. The component file is chosen by the head's file dialog, which feeds the
// running tool through SetPlaceComponentDocument; each ground-plane click then drops an
// instance (#763).
func placeComponentCommand() *CommandDefinition {
	return NewCommand("Assembly.Place", "Place", "Component", runPlaceComponent).
		WithTab("Assemble").
		WithRibbons(AssemblyRibbon).
		WithEnable(hasActiveAssembly).
		WithIcon("place").WithButtonStyle(LargeIconButton).
		WithTooltip("Place a component into the assembly.")
}

// runPlaceComponent starts the Place Component tool on the active assembly. The tool then waits
// for a chosen component (the head's file dialog) and for ground-plane clicks to drop instances.
func runPlaceComponent(s *Session) error {
	if _, err := activeAssembly(s); err != nil {
		return err
	}
	s.StartTool(NewPlaceComponentTool())
	return nil
}
