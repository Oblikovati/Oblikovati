// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// blockPart builds a part whose evaluated body is the box [min,max], so a source
// assembly placing it has real geometry to derive.
func blockPart(t *testing.T, min, max math.Point3) *compdef.PartComponentDefinition {
	t.Helper()
	block, err := brep.SolidBlock(min, max, "p")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	p := compdef.NewPartComponentDefinition()
	feature.NewBaseFeatures(p.Features()).AddBase(block)
	p.Recompute()
	return p
}

// addAssemblyDoc adds a source assembly document to the workspace, placing one unit-box
// part per given X translation, and keeps the active document unchanged so the derive
// target stays the active part.
func addAssemblyDoc(t *testing.T, s *app.Session, name string, xs ...float64) *doc.Document {
	t.Helper()
	active := s.ActiveDocument()
	d, err := compdef.AddAssembly(s.Workspace(), name, true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	asm := d.Content().(*compdef.AssemblyComponentDefinition)
	for i, x := range xs {
		asm.Place(fmt.Sprintf("box:%d", i+1), blockPart(t, math.P3(0, 0, 0), math.P3(1, 1, 1)), math.Translation4(math.V3(x, 0, 0)))
	}
	if active != nil {
		if err := s.Workspace().SetActiveDocument(active); err != nil {
			t.Fatalf("restore active: %v", err)
		}
	}
	return d
}

// activePartVolume sums the active part's body volumes (the derived result).
func activePartVolume(t *testing.T, s *app.Session) float64 {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	v := 0.0
	for _, b := range def.SurfaceBodies().All() {
		v += ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	}
	return v
}

// TestAssemblyDeriveCreateOverWire derives a one-box assembly into the active part and
// gates the result against the analytic volume (the box, 1.0).
func TestAssemblyDeriveCreateOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	src := addAssemblyDoc(t, s, "src.obk", 0)

	var detail wire.FeatureDetailResult
	call(t, r, s, "assembly.deriveCreate", fmt.Sprintf(`{"source":%d}`, src.ID()), &detail)

	if detail.Feature.Kind != "derivedAssembly" {
		t.Errorf("derived feature kind = %q, want derivedAssembly", detail.Feature.Kind)
	}
	if got := activePartVolume(t, s); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("derived part volume = %g, want 1.0 (the source box)", got)
	}
}

// TestAssemblyShrinkwrapCreateOverWire shrinkwraps a two-box assembly with a whole
// envelope: the result is one box enclosing both unit boxes (X spanning 0..11 → 11).
func TestAssemblyShrinkwrapCreateOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	src := addAssemblyDoc(t, s, "src.obk", 0, 10)

	var detail wire.FeatureDetailResult
	args := fmt.Sprintf(`{"source":%d,"envelopeStyle":%d}`, src.ID(), int32(2)) // EnvelopeWhole
	call(t, r, s, "assembly.shrinkwrapCreate", args, &detail)

	if detail.Feature.Kind != "shrinkwrap" {
		t.Errorf("shrinkwrap feature kind = %q, want shrinkwrap", detail.Feature.Kind)
	}
	if got := activePartVolume(t, s); stdmath.Abs(got-11.0) > 1e-6 {
		t.Errorf("whole-envelope shrinkwrap volume = %g, want 11.0 (box enclosing both)", got)
	}
}

// TestAssemblyDeriveBreakLinkOverWire creates a derive, then breaks its link and checks
// the feature is no longer linked.
func TestAssemblyDeriveBreakLinkOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	src := addAssemblyDoc(t, s, "src.obk", 0)

	var created wire.FeatureDetailResult
	call(t, r, s, "assembly.deriveCreate", fmt.Sprintf(`{"source":%d}`, src.ID()), &created)

	var broken wire.FeatureDetailResult
	call(t, r, s, "assembly.deriveBreakLink", fmt.Sprintf(`{"id":%d}`, created.Feature.ID), &broken)

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	f, ok := def.Features().ByID(feature.ID(created.Feature.ID))
	if !ok {
		t.Fatalf("feature %d not found after break link", created.Feature.ID)
	}
	derived := f.Definition().(*feature.DerivedAssemblyComponent)
	if derived.Linked() {
		t.Error("derived feature still linked after assembly.deriveBreakLink")
	}
}

// TestAssemblyDeriveRejectsNonAssemblySource: deriving from a part document (no
// occurrence tree) is an error naming the offending document.
func TestAssemblyDeriveRejectsNonAssemblySource(t *testing.T) {
	r, s := emptyPartSession(t)
	partID := uint64(s.ActiveDocument().ID()) // the active part itself

	_, err := r.Handle(s, "assembly.deriveCreate", []byte(fmt.Sprintf(`{"source":%d}`, partID)))
	if err == nil {
		t.Fatal("deriveCreate from a part source returned nil error, want a rejection")
	}
}

// TestAssemblyBreakLinkRejectsUnknownFeature: break-link on a missing id is an error.
func TestAssemblyBreakLinkRejectsUnknownFeature(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "assembly.deriveBreakLink", []byte(`{"id":9999}`)); err == nil {
		t.Fatal("deriveBreakLink on an unknown feature returned nil error, want a rejection")
	}
}

// TestAssemblyDeriveStatusAndUpdateOverWire reads a fresh derive's drive state over the
// wire (linked, not out of date, naming its source), exercises update (idempotent for a
// current derive), and checks break-link flips the reported linked state (#751).
func TestAssemblyDeriveStatusAndUpdateOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	src := addAssemblyDoc(t, s, "src.obk", 0)

	var detail wire.FeatureDetailResult
	call(t, r, s, "assembly.deriveCreate", fmt.Sprintf(`{"source":%d}`, src.ID()), &detail)
	id := detail.Feature.ID

	var status wire.DeriveStatusResult
	call(t, r, s, "assembly.deriveStatus", fmt.Sprintf(`{"id":%d}`, id), &status)
	if status.OutOfDate || !status.Linked || status.SourceDocument != "src.obk" {
		t.Fatalf("status = %+v, want linked, current, source src.obk", status)
	}

	var updated wire.DeriveStatusResult
	call(t, r, s, "assembly.deriveUpdate", fmt.Sprintf(`{"id":%d}`, id), &updated)
	if updated.OutOfDate {
		t.Error("updating a current derive should leave it not out of date")
	}

	var broken wire.FeatureDetailResult
	call(t, r, s, "assembly.deriveBreakLink", fmt.Sprintf(`{"id":%d}`, id), &broken)
	call(t, r, s, "assembly.deriveStatus", fmt.Sprintf(`{"id":%d}`, id), &status)
	if status.Linked {
		t.Error("after break-link, status should report Linked=false")
	}

	if _, err := r.Handle(s, "assembly.deriveStatus", []byte(`{"id":99999}`)); err == nil {
		t.Error("deriveStatus of an unknown feature id should fail")
	}
}
