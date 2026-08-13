// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// The assembly editing-options surface (#1981): read and write the assembly-modeling option set that
// governs placement, adaptivity, update deferral, and section/opacity defaults.

// registerAssemblyOptionsHandlers wires the assembly.options* methods.
func (r *Router) registerAssemblyOptionsHandlers() {
	r.readOnly(wire.MethodAssemblyOptionsGet, assemblyQuery(assemblyOptionsGet))
	r.mutating(wire.MethodAssemblyOptionsSet, "Set Assembly Options", typedAssembly(assemblyOptionsSet))
}

// assemblyOptionsGet returns the active assembly's editing options.
func assemblyOptionsGet(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.AssemblyOptionsResult, error) {
	return wire.AssemblyOptionsResult{Options: assemblyOptionsInfo(asm.Options())}, nil
}

// assemblyOptionsSet replaces the active assembly's editing options and returns them.
func assemblyOptionsSet(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetAssemblyOptionsArgs) (wire.AssemblyOptionsResult, error) {
	asm.SetOptions(assemblyOptionsModel(in.Options))
	s.ActiveDocument().MarkDirty()
	return wire.AssemblyOptionsResult{Options: assemblyOptionsInfo(asm.Options())}, nil
}

// assemblyOptionsInfo maps the model options to their wire shape.
func assemblyOptionsInfo(o compdef.AssemblyOptions) wire.AssemblyOptions {
	return wire.AssemblyOptions{
		PartFeaturesInitiallyAdaptive:            o.PartFeaturesInitiallyAdaptive,
		DeferUpdate:                              o.DeferUpdate,
		OnlyActiveComponentIsOpaque:              o.OnlyActiveComponentIsOpaque,
		PlaceAndGroundFirstComponentAtOrigin:     o.PlaceAndGroundFirstComponentAtOrigin,
		EnableConstraintRedundancyAnalysis:       o.EnableConstraintRedundancyAnalysis,
		DeleteComponentPatternSources:            o.DeleteComponentPatternSources,
		SectionAllParts:                          o.SectionAllParts,
		UseLastOccurrenceOrientationForPlacement: o.UseLastOccurrenceOrientationForPlacement,
		DefaultLevelOfDetail:                     o.DefaultLevelOfDetail,
		DefaultDesignView:                        o.DefaultDesignView,
	}
}

// assemblyOptionsModel maps the wire options back to the model shape.
func assemblyOptionsModel(o wire.AssemblyOptions) compdef.AssemblyOptions {
	return compdef.AssemblyOptions{
		PartFeaturesInitiallyAdaptive:            o.PartFeaturesInitiallyAdaptive,
		DeferUpdate:                              o.DeferUpdate,
		OnlyActiveComponentIsOpaque:              o.OnlyActiveComponentIsOpaque,
		PlaceAndGroundFirstComponentAtOrigin:     o.PlaceAndGroundFirstComponentAtOrigin,
		EnableConstraintRedundancyAnalysis:       o.EnableConstraintRedundancyAnalysis,
		DeleteComponentPatternSources:            o.DeleteComponentPatternSources,
		SectionAllParts:                          o.SectionAllParts,
		UseLastOccurrenceOrientationForPlacement: o.UseLastOccurrenceOrientationForPlacement,
		DefaultLevelOfDetail:                     o.DefaultLevelOfDetail,
		DefaultDesignView:                        o.DefaultDesignView,
	}
}
