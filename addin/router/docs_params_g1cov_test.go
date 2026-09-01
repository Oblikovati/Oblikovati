// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"path/filepath"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// dpcovMethodArgs pairs a wire method with a JSON argument string for the
// table-driven error-path tests below (the M40-G1 typed-router coverage push).
type dpcovMethodArgs struct {
	method string
	args   string
}

// dpcovAllReject asserts every (method,args) pair returns an error.
func dpcovAllReject(t *testing.T, r *Router, s *app.Session, cases []dpcovMethodArgs) {
	t.Helper()
	for _, c := range cases {
		if _, err := r.Handle(s, c.method, []byte(c.args)); err == nil {
			t.Errorf("%s(%s) must fail", c.method, c.args)
		}
	}
}

// --- files.go: file-side reference records (fileReferenceInfo was 0%) ---

func TestDpcovFileReferencesFileSide(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	s.ActiveDocument().SetFileReferenceRecords([]doc.FileReferenceRecord{{
		FullFileName: "/asm/pin.obk", RelativeFileName: "pin.obk",
		LocationType: "ownerDirectory", SaveCounter: 2,
	}})
	var refs wire.ListFileReferencesResult
	call(t, r, s, "files.listReferences", `{"fullFileName":"test.obk"}`, &refs)
	if len(refs.References) != 1 {
		t.Fatalf("references = %+v, want the seeded record", refs.References)
	}
	if got := refs.References[0]; got.FullFileName != "/asm/pin.obk" || got.SaveCounter != 2 || !got.Missing {
		t.Errorf("ref = %+v, want the file-side view of the missing pin", got)
	}
}

func TestDpcovFileReferenceErrorPaths(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"files.listReferences", `{"fullFileName":"ghost.obk"}`},
		{"documents.listFileReferences", `{"document":999999}`},
	})
}

// --- save.go ---

func TestDpcovSaveHandlersRejectUnknownDoc(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"documents.save", `{"document":999999}`},
		{"documents.saveAs", `{"document":999999,"newFullDocumentName":"/x.opd"}`},
		{"documents.saveCopyAs", `{"document":999999,"targetFileName":"/x.opd"}`},
		{"documents.batchSave", `{"operation":"save","items":[{"document":999999}]}`},
		{"documents.open", `{}`},
		{"documents.open", `{"fullDocumentName":"/no/such.opd"}`},
	})
}

func TestDpcovBatchSaveSaveAndSaveAs(t *testing.T) {
	t.Parallel()
	r, s, dir := storedSession(t)
	id := uint64(s.ActiveDocument().ID())
	var res wire.BatchSaveResult
	call(t, r, s, "documents.batchSave", fmt.Sprintf(`{"operation":"save","items":[{"document":%d}]}`, id), &res)
	if res.Saved != 1 {
		t.Fatalf("batch save = %+v, want one saved", res)
	}
	target := filepath.Join(dir, "as.opd")
	call(t, r, s, "documents.batchSave",
		fmt.Sprintf(`{"operation":"saveAs","items":[{"document":%d,"targetFileName":%q}]}`, id, target), &res)
	if res.Saved != 1 {
		t.Fatalf("batch saveAs = %+v, want one saved", res)
	}
}

// --- documents.go ---

func TestDpcovDocumentLifecycleErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"documents.create", `{}`},
		{"documents.activate", `{"id":999999}`},
		{"documents.close", `{"id":999999}`},
	})
}

// --- material.go ---

func TestDpcovMaterialErrorPaths(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"materials.create", mustJSON(t, wire.DuplicateAssetArgs{BaseID: "nope", Name: "X"})},
		{"appearances.create", mustJSON(t, wire.DuplicateAssetArgs{BaseID: "nope", Name: "X"})},
		{"appearances.update", mustJSON(t, wire.UpdateAppearanceArgs{ID: "nope"})},
		{"model.assignMaterial", mustJSON(t, wire.AssignMaterialArgs{BodyKey: "nope", MaterialID: "nope"})},
	})
}

func TestDpcovMaterialUpdateUnknownAndPhysProps(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"materials.update", mustJSON(t, wire.MaterialInfo{ID: "nope"})},
		{"appearances.update", mustJSON(t, wire.UpdateAppearanceArgs{ID: "nope"})},
	})
	rr := New(opregistry.Default())
	if _, err := rr.Handle(app.NewSession(), "model.physicalProperties", []byte(`{}`)); err == nil {
		t.Error("physicalProperties without an active part must fail")
	}
}

// --- keymap.go ---

func TestDpcovKeymapErrorPaths(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"keymap.setChord", `{"actionId":"Test.X","chord":"Nonsense"}`},
		{"keymap.setAlias", `{"actionId":"Ghost.Action","alias":"ZZ"}`},
	})
}

// --- interests.go ---

func TestDpcovInterestsUnknownDoc(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"documents.listInterests", `{"document":999999}`},
		{"documents.addInterest", `{"document":999999,"interest":{"clientId":"c","name":"n"}}`},
		{"documents.removeInterest", `{"document":999999,"clientId":"c","name":"n"}`},
		{"documents.hasInterest", `{"document":999999,"client":"c"}`},
	})
}

// --- attachments.go ---

func TestDpcovAttachmentsUnknownDoc(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"documents.listAttachments", `{"document":999999}`},
		{"documents.addAttachment", `{"document":999999,"name":"a","kind":3331,"fullFileName":"/x"}`},
		{"documents.removeAttachment", `{"document":999999,"name":"a"}`},
	})
}

// --- help.go + commands.go ---

func TestDpcovHelpAndCommandErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"help.path", `{"source":"unregistered.x"}`},
		{"commands.setState", `{}`},
		{"commands.setState", `{"id":"ghost"}`},
		{"commands.execute", `{}`},
	})
}

// --- messaging.go ---

func TestDpcovMessagingErrorPaths(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"progress.end", `{"id":999999}`},
		{"errors.endSection", `{"section":999999}`},
		{"prompts.show", `{"message":"?"}`},
	})
}

// --- document_properties.go: value-type converters + errors ---

func TestDpcovDocumentPropertyValueTypes(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	id := uint64(s.ActiveDocument().ID())
	for _, v := range []types.Variant{types.IntegerVariant(7), types.DoubleVariant(3.5), types.BoolVariant(true)} {
		args := mustJSON(t, wire.SetPropertyArgs{Document: id, Set: "Custom", Name: "P", Value: v})
		var res wire.PropertyResult
		call(t, r, s, "documents.setProperty", args, &res)
		if res.Property.Value.Type() != v.Type() {
			t.Errorf("round-trip type = %v, want %v", res.Property.Value.Type(), v.Type())
		}
	}
}

func TestDpcovDocumentPropertyErrors(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	id := uint64(s.ActiveDocument().ID())
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"documents.getProperty", fmt.Sprintf(`{"document":%d,"set":"Nope","name":"X"}`, id)},
		{"documents.setProperty", fmt.Sprintf(`{"document":%d,"set":"Custom","name":""}`, id)},
		{"documents.getProperty", `{"document":999999,"set":"x","name":"y"}`},
	})
}

// --- parameter_groups.go ---

func TestDpcovParameterGroupErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"parameters.groups.setDisplayName", `{"internalName":"ghost","displayName":"X"}`},
		{"parameters.groups.delete", `{"internalName":"ghost"}`},
		{"parameters.groups.addMember", `{"internalName":"ghost","parameter":"width"}`},
	})
}

func TestDpcovParameterGroupAddMember(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "parameters.groups.add", `{"internalName":"g1","displayName":"G1"}`, nil)
	var info wire.ParameterGroupInfo
	call(t, r, s, "parameters.groups.addMember", `{"internalName":"g1","parameter":"width"}`, &info)
	if len(info.Members) != 1 || info.Members[0] != "width" {
		t.Errorf("group members = %+v, want [width]", info.Members)
	}
}

// --- parameters_detail.go ---

func TestDpcovParameterDetailErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"parameters.getDetail", `{"name":"ghost"}`},
		{"parameters.drivenBy", `{"name":"ghost"}`},
		{"parameters.delete", `{"name":"ghost"}`},
		{"parameters.setTolerance", `{"name":"ghost"}`},
		{"parameters.setExpressionList", `{"name":"ghost","expressions":["1 cm"]}`},
	})
}

// --- parameter_settings.go ---

func TestDpcovParameterSettingsErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"parameters.setSettings", `{"dimensionDisplayType":"bogus"}`},
		{"parameters.import", `{"xml":"<not-valid"}`},
	})
}

// --- units.go ---

func TestDpcovUnitsErrorPaths(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"units.getPreciseStringFromValue", `{"value":1,"unitsType":"bogus"}`},
		{"units.getDatabaseUnitsFromExpression", `{"expression":"5 bogusunit"}`},
		{"units.getDrivingParameters", `{"expression":"1 +"}`},
	})
	rr := New(opregistry.Default())
	if _, err := rr.Handle(app.NewSession(), "documents.getUnits", []byte(`{}`)); err == nil {
		t.Error("getUnits without an active document must fail")
	}
}

func TestDpcovSetUnitsAnglePreferences(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var got wire.DocumentUnitsInfo
	call(t, r, s, "documents.setUnits", `{"angleUnit":"rad","angleDisplayPrecision":5}`, &got)
	if got.AngleUnit != "rad" || got.AngleDisplayPrecision != 5 {
		t.Errorf("after set = %+v, want rad/5", got)
	}
	if _, err := r.Handle(s, "documents.setUnits", []byte(`{"angleDisplayPrecision":-1}`)); err == nil {
		t.Error("a negative angle precision must fail")
	}
}

// --- work_surfaces.go ---

func TestDpcovWorkSurfaceOutOfRange(t *testing.T) {
	t.Parallel()
	r, s := patchedSurfacePartViaAPI(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"workSurfaces.setVisible", mustJSON(t, wire.SetWorkSurfaceVisibleArgs{Index: 9, Visible: false})},
		{"workSurfaces.rename", mustJSON(t, wire.RenameWorkSurfaceArgs{Index: 9, Name: "X"})},
	})
}

// --- windows.go ---

func TestDpcovWindowsCloseTabUnknown(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	if _, err := r.Handle(s, "windows.closeTab", []byte(`{"document":999999,"force":true}`)); err == nil {
		t.Error("closing an unknown tab must fail")
	}
}

// --- attributes.go ---

func TestDpcovAttributeSetFilter(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	id := uint64(s.ActiveDocument().ID())
	call(t, r, s, "attributes.set", mustJSON(t, wire.SetAttributeArgs{Document: id, Set: "A", Name: "x", Value: types.IntegerVariant(1)}), nil)
	call(t, r, s, "attributes.set", mustJSON(t, wire.SetAttributeArgs{Document: id, Set: "B", Name: "y", Value: types.IntegerVariant(2)}), nil)
	var lst wire.ListAttributesResult
	call(t, r, s, "attributes.list", mustJSON(t, wire.ListAttributesArgs{Document: id, Set: "A"}), &lst)
	if len(lst.Attributes) != 1 || lst.Attributes[0].Set != "A" {
		t.Errorf("filtered list = %+v, want only set A", lst.Attributes)
	}
}

func TestDpcovAttributeErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"attributes.get", `{"document":999999,"set":"A","name":"x"}`},
		{"attributes.delete", `{"document":999999,"set":"A"}`},
		{"attributes.list", `{"document":999999,"allTargets":true}`},
	})
}

// --- point_clouds.go ---

func TestDpcovPointCloudUnknownName(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"pointClouds.setTransform", `{"name":"Ghost"}`},
		{"pointClouds.toModelSpace", `{"name":"Ghost"}`},
		{"pointClouds.fromModelSpace", `{"name":"Ghost"}`},
		{"pointClouds.addCrop", `{"name":"Ghost"}`},
		{"pointClouds.nearestPoint", `{"name":"Ghost"}`},
		{"pointClouds.setCropActive", `{"name":"Ghost"}`},
	})
}

// --- feature_lifecycle.go ---

func TestDpcovFeatureLifecycleUnknownID(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedPartViaAPI(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"features.edit", `{"id":999999,"scalars":[{"index":0,"value":"1 mm"}]}`},
		{"features.rename", `{"id":999999,"name":"X"}`},
		{"features.setSuppressed", `{"id":999999,"suppressed":true}`},
		{"features.reorder", `{"id":999999,"newIndex":0}`},
	})
}

// --- representations.go ---

func TestDpcovRepresentationSettersRequireAssembly(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"designReps.setVisibility", `{}`},
		{"designReps.setAppearance", `{}`},
		{"designReps.addSection", `{}`},
		{"positionalReps.setOverride", `{}`},
		{"positionalReps.setFlexible", `{}`},
		{"lodReps.setSuppressed", `{}`},
	})
}

func TestDpcovRepresentationActivateUnknown(t *testing.T) {
	t.Parallel()
	r, s, _, _ := assemblySessionWithBoxes(t, 0, 5)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"designReps.activate", `{"id":999999}`},
		{"positionalReps.activate", `{"id":999999}`},
		{"lodReps.activate", `{"id":999999}`},
		{"modelStates.activate", `{"id":999999}`},
	})
}

// --- flat_pattern.go ---

func TestDpcovFlatPatternErrorPaths(t *testing.T) {
	t.Parallel()
	r, s := flangedSheet(t)
	dpcovAllReject(t, r, s, []dpcovMethodArgs{
		{"flatPattern.edgesOfType", `{"type":"bogus"}`},
		{"flatPattern.addOrientation", `{"name":"X","alignmentType":"bogus"}`},
	})
}
