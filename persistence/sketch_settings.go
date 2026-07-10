// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"oblikovati.org/api/types"
	"oblikovati.org/yamlcodec"
)

// toCodecSketchSettings converts the per-document sketch settings into their on-disk record (#147)
// so they round-trip in the .obk; the constraint-priority enum becomes its frozen integer id.
func toCodecSketchSettings(s types.SketchSettings) *yamlcodec.SketchSettingsRecord {
	return &yamlcodec.SketchSettingsRecord{
		InferConstraints:           s.InferConstraints,
		AutoApplyConstraints:       s.AutoApplyConstraints,
		ConstraintPriority:         int32(s.ConstraintPriority),
		XSnapSpacing:               s.XSnapSpacing,
		YSnapSpacing:               s.YSnapSpacing,
		SnapsPerMinorGrid:          s.SnapsPerMinorGrid,
		MinorLinesPerMajorGridLine: s.MinorLinesPerMajorGridLine,

		PersistInferredConstraints:   s.PersistInferredConstraints,
		DisplayConstraintsOnCreation: s.DisplayConstraintsOnCreation,
		EditDimensionsWhenCreated:    s.EditDimensionsWhenCreated,
		OverConstrainedBehavior:      int32(s.OverConstrainedBehavior),

		EnableRelaxMode:                       s.EnableRelaxMode,
		KeepDimensionsWithEquationInRelaxMode: s.KeepDimensionsWithEquationInRelaxMode,
	}
}

// fromCodecSketchSettings rebuilds the sketch settings from the on-disk record.
func fromCodecSketchSettings(r *yamlcodec.SketchSettingsRecord) types.SketchSettings {
	return types.SketchSettings{
		InferConstraints:           r.InferConstraints,
		AutoApplyConstraints:       r.AutoApplyConstraints,
		ConstraintPriority:         types.ConstraintInferencePriority(r.ConstraintPriority),
		XSnapSpacing:               r.XSnapSpacing,
		YSnapSpacing:               r.YSnapSpacing,
		SnapsPerMinorGrid:          r.SnapsPerMinorGrid,
		MinorLinesPerMajorGridLine: r.MinorLinesPerMajorGridLine,

		PersistInferredConstraints:   r.PersistInferredConstraints,
		DisplayConstraintsOnCreation: r.DisplayConstraintsOnCreation,
		EditDimensionsWhenCreated:    r.EditDimensionsWhenCreated,
		OverConstrainedBehavior:      types.OverConstrainedDimensionBehavior(r.OverConstrainedBehavior),

		EnableRelaxMode:                       r.EnableRelaxMode,
		KeepDimensionsWithEquationInRelaxMode: r.KeepDimensionsWithEquationInRelaxMode,
	}
}
