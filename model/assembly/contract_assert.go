// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/api/contract"

// Compile-time assertions that the constraint engine satisfies the public Apache-2.0
// contract (ADR-0018): the read surfaces an in-proc consumer binds against.
var (
	_ contract.AssemblyConstraints           = (*ConstraintSet)(nil)
	_ contract.AssemblyConstraintsEnumerator = (*OccurrenceConstraints)(nil)
	_ contract.ConstraintLimits              = (*limits)(nil)
	_ contract.MateConstraint                = (*MateConstraint)(nil)
	_ contract.FlushConstraint               = (*FlushConstraint)(nil)
	_ contract.AngleConstraint               = (*AngleConstraint)(nil)
	_ contract.TangentConstraint             = (*TangentConstraint)(nil)
	_ contract.InsertConstraint              = (*InsertConstraint)(nil)
	_ contract.AssemblySymmetryConstraint    = (*AssemblySymmetryConstraint)(nil)
	_ contract.RotateRotateConstraint        = (*RotateRotateConstraint)(nil)
	_ contract.RotateTranslateConstraint     = (*RotateTranslateConstraint)(nil)
	_ contract.TranslateTranslateConstraint  = (*TranslateTranslateConstraint)(nil)
	_ contract.TransitionalConstraint        = (*TransitionalConstraint)(nil)
	_ contract.CustomConstraint              = (*CustomConstraint)(nil)

	// Joints (M12-F02).
	_ contract.AssemblyJoint            = (*assemblyJoint)(nil)
	_ contract.AssemblyJoints           = (*JointSet)(nil)
	_ contract.AssemblyJointsEnumerator = (*OccurrenceJoints)(nil)
	_ contract.JointLimits              = (*jointLimits)(nil)
	_ contract.AssemblyJointProxy       = jointProxy{}
	_ contract.AssemblyJointDefinition  = jointDefinition{}
	_ contract.DSJoint                  = (*dsJoint)(nil)
	_ contract.DSJoints                 = (*DSJointSet)(nil)
	_ contract.DSJointDefinition        = dsJointDefinition{}
	_ contract.DSDegreesOfFreedom       = (*dsDOF)(nil)

	// Drive (M12-F03).
	_ contract.DriveSettings = DriveSettings{}

	// Representations (M12-F04).
	_ contract.DesignViewRepresentation    = (*designViewRep)(nil)
	_ contract.PositionalRepresentation    = (*positionalRep)(nil)
	_ contract.LevelOfDetailRepresentation = (*lodRep)(nil)
	_ contract.ModelState                  = (*modelState)(nil)
	_ contract.RepresentationsManager      = (*Representations)(nil)
)
