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

// TestGroundFirstComponentOption: with the option on (default) the first placed component grounds;
// with it off it does not (#1981).
func TestGroundFirstComponentOption(t *testing.T) {
	on := NewAssemblyComponentDefinition()
	first := on.Place("paint:1", NewVirtualComponent("paint", "P-1", bom.Normal), math.Identity4())
	if !first.Grounded() {
		t.Error("with the option on, the first component should be grounded")
	}
	second := on.Place("grease:1", NewVirtualComponent("grease", "G-1", bom.Normal), math.Identity4())
	if second.Grounded() {
		t.Error("only the FIRST component grounds, not the second")
	}

	off := NewAssemblyComponentDefinition()
	off.SetOptions(AssemblyOptions{PlaceAndGroundFirstComponentAtOrigin: false})
	o := off.Place("paint:1", NewVirtualComponent("paint", "P-1", bom.Normal), math.Identity4())
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
