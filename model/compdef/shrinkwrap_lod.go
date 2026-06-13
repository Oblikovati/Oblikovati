// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
)

// ShrinkwrapToPart builds a simplified part definition from an assembly source: it runs
// the shrinkwrap recipe over the source and wraps the resulting lightweight bodies as a
// non-parametric base in a fresh part (M11-F06). The part is a placeable
// occurrence.Definition, so it can stand in for the assembly as a substitute LOD.
//
// Example:
//
//	lod, _ := ShrinkwrapToPart(asm, feature.ShrinkwrapDefinition{EnvelopeStyle: feature.EnvelopeWhole})
func ShrinkwrapToPart(source feature.AssemblyBodySource, def feature.ShrinkwrapDefinition) (*PartComponentDefinition, error) {
	bodies, err := feature.BuildShrinkwrap(source, def)
	if err != nil {
		return nil, err
	}
	part := NewPartComponentDefinition()
	feature.NewBaseFeatures(part.Features()).AddBase(bodies...)
	part.Recompute()
	return part, nil
}

// SubstituteWithShrinkwrap simplifies source into a lightweight part and substitutes it
// for the given source occurrences in dst: the sources are suppressed and one
// IsSubstitute occurrence referencing the shrinkwrap LOD is added at transform. This is
// the substitute-representation (level-of-detail) path — a shrinkwrap result registered
// as a substitute of the assembly it simplifies (M11-F06, feeds #348/#355, #361).
func SubstituteWithShrinkwrap(dst *occurrence.Occurrences, sources []*occurrence.Occurrence, name string, source feature.AssemblyBodySource, def feature.ShrinkwrapDefinition, transform math.Matrix4) (*occurrence.Occurrence, error) {
	lod, err := ShrinkwrapToPart(source, def)
	if err != nil {
		return nil, err
	}
	return occurrence.Substitute(dst, sources, name, lod, transform), nil
}
