// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "oblikovati.org/model/occurrence"

// Assembly options (#1981): the assembly-editing option set that governs placement, adaptivity,
// update deferral, and section/opacity defaults — Inventor's AssemblyOptions. Two options change
// behaviour here: PlaceAndGroundFirstComponentAtOrigin grounds the first placed component, and
// DeferUpdate batches recomputes until it is cleared. The rest are stored defaults the head/add-ins
// read and write.

// AssemblyOptions holds an assembly's editing options.
type AssemblyOptions struct {
	PartFeaturesInitiallyAdaptive            bool
	DeferUpdate                              bool
	OnlyActiveComponentIsOpaque              bool
	PlaceAndGroundFirstComponentAtOrigin     bool
	EnableConstraintRedundancyAnalysis       bool
	DeleteComponentPatternSources            bool
	SectionAllParts                          bool
	UseLastOccurrenceOrientationForPlacement bool
	DefaultLevelOfDetail                     string
	DefaultDesignView                        string
}

// defaultAssemblyOptions returns the option defaults a new assembly opens with — first-component
// grounding and redundancy analysis on, everything else off.
func defaultAssemblyOptions() AssemblyOptions {
	return AssemblyOptions{
		PlaceAndGroundFirstComponentAtOrigin: true,
		EnableConstraintRedundancyAnalysis:   true,
	}
}

// Options returns the assembly's editing options (#1981).
func (a *AssemblyComponentDefinition) Options() AssemblyOptions { return a.options }

// SetOptions replaces the assembly's editing options. Clearing DeferUpdate flushes any recompute that
// was deferred while it was set (#1981).
func (a *AssemblyComponentDefinition) SetOptions(opts AssemblyOptions) {
	wasDeferred := a.options.DeferUpdate
	a.options = opts
	if wasDeferred && !opts.DeferUpdate && a.updatePending {
		a.updatePending = false
		a.RecomputeFeatures()
	}
}

// groundFirstComponent grounds a freshly placed occurrence when it is the assembly's first and the
// place-and-ground-first-component option is on (#1981).
func (a *AssemblyComponentDefinition) groundFirstComponent(o *occurrence.Occurrence) {
	if a.options.PlaceAndGroundFirstComponentAtOrigin && a.occurrences.Count() == 1 {
		o.SetGrounded(true)
	}
}
