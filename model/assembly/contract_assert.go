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

	// The umbrella read surface every relationship kind embeds (#1619): asserted on the
	// base so a contract-method drift breaks here, not at the first usage-site refactor.
	_ contract.AssemblyConstraint = (*constraintBase)(nil)

	// Representations (M12-F04).
	_ contract.DesignViewRepresentation    = (*designViewRep)(nil)
	_ contract.PositionalRepresentation    = (*positionalRep)(nil)
	_ contract.LevelOfDetailRepresentation = (*lodRep)(nil)
	_ contract.ModelState                  = (*modelState)(nil)
	_ contract.RepresentationsManager      = (*Representations)(nil)

	// Umbrella representation contracts (#1619): each family satisfies the common
	// Representation surface; the model-state collection satisfies ModelStates.
	_ contract.Representation = (*designViewRep)(nil)
	_ contract.Representation = (*positionalRep)(nil)
	_ contract.Representation = (*lodRep)(nil)
	_ contract.ModelStates    = modelStateCollection{}

	_ contract.DesignViewRepresentations    = designViewCollection{}
	_ contract.PositionalRepresentations    = positionalCollection{}
	_ contract.LevelOfDetailRepresentations = lodCollection{}

	// Contact & interference (M12-F05).
	_ contract.ContactSet          = (*contactSet)(nil)
	_ contract.ContactSets         = (*ContactSolver)(nil)
	_ contract.ContactSolver       = (*ContactSolver)(nil)
	_ contract.InterferenceResult  = InterferenceResult{}
	_ contract.InterferenceResults = InterferenceResults{}
)
