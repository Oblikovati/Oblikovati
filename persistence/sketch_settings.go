// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"oblikovati.org/api/types"
	"oblikovati.org/persistence/yamlcodec"
)

// toCodecSketchSettings converts the per-document sketch settings into their on-disk record (#147)
// so they round-trip in the .obk; the constraint-priority enum becomes its frozen integer id.
func toCodecSketchSettings(s types.SketchSettings) *yamlcodec.SketchSettingsRecord {
	return &yamlcodec.SketchSettingsRecord{
		InferConstraints:     s.InferConstraints,
		AutoApplyConstraints: s.AutoApplyConstraints,
		ConstraintPriority:   int32(s.ConstraintPriority),
	}
}

// fromCodecSketchSettings rebuilds the sketch settings from the on-disk record.
func fromCodecSketchSettings(r *yamlcodec.SketchSettingsRecord) types.SketchSettings {
	return types.SketchSettings{
		InferConstraints:     r.InferConstraints,
		AutoApplyConstraints: r.AutoApplyConstraints,
		ConstraintPriority:   types.ConstraintInferencePriority(r.ConstraintPriority),
	}
}
