// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/bom"
)

// TestOptionsDefaults: a new assembly opens with first-component grounding and redundancy analysis on
// (#1981).
func TestOptionsDefaults(t *testing.T) {
	opts := NewAssemblyComponentDefinition().Options()
	if !opts.PlaceAndGroundFirstComponentAtOrigin || !opts.EnableConstraintRedundancyAnalysis {
		t.Errorf("defaults = %+v, want first-component grounding + redundancy analysis on", opts)
	}
	if opts.DeferUpdate || opts.SectionAllParts {
		t.Errorf("defaults = %+v, want deferUpdate/sectionAllParts off", opts)
	}
}

// TestGroundFirstComponentOption: the place-action grounding hook grounds the first component when the
// option is on (default) and only the first; with the option off it grounds nothing. The low-level
// Place itself never grounds (internal placement is not surprised) (#1981).
func TestGroundFirstComponentOption(t *testing.T) {
	on := NewAssemblyComponentDefinition()
	first := on.Place("paint:1", NewVirtualComponent("paint", "P-1", bom.Normal), math.Identity4())
	if first.Grounded() {
		t.Error("the low-level Place must not ground on its own")
	}
	on.GroundFirstComponentIfEnabled(first)
	if !first.Grounded() {
		t.Error("with the option on, the place action should ground the first component")
	}
	second := on.Place("grease:1", NewVirtualComponent("grease", "G-1", bom.Normal), math.Identity4())
	on.GroundFirstComponentIfEnabled(second)
	if second.Grounded() {
		t.Error("only the FIRST component grounds, not the second")
	}

	off := NewAssemblyComponentDefinition()
	off.SetOptions(AssemblyOptions{PlaceAndGroundFirstComponentAtOrigin: false})
	o := off.Place("paint:1", NewVirtualComponent("paint", "P-1", bom.Normal), math.Identity4())
	off.GroundFirstComponentIfEnabled(o)
	if o.Grounded() {
		t.Error("with the option off, the first component should not be grounded")
	}
}

// TestDeferUpdateBatchesRecompute: with DeferUpdate on a recompute is deferred, and clearing the flag
// flushes it (#1981).
func TestDeferUpdateBatchesRecompute(t *testing.T) {
	a := NewAssemblyComponentDefinition()
	a.SetOptions(AssemblyOptions{DeferUpdate: true})
	a.RecomputeFeatures()
	if !a.updatePending {
		t.Fatal("with DeferUpdate on, RecomputeFeatures should defer (updatePending)")
	}
	a.SetOptions(AssemblyOptions{DeferUpdate: false}) // clearing flushes the deferred recompute
	if a.updatePending {
		t.Error("clearing DeferUpdate should flush the deferred recompute (updatePending cleared)")
	}
}
