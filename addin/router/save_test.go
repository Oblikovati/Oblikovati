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
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// storedSession is a router session whose workspace persists real .opd files
// under a temp dir — the open/save lifecycle needs a live store.
func storedSession(t *testing.T) (*Router, *app.Session, string) {
	t.Helper()
	dir := t.TempDir()
	s := app.NewSessionWithStore(persistence.NewPackageStore())
	d, err := s.Workspace().Add(doc.Part, filepath.Join(dir, "bracket.opd"), true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	d.SetContent(compdef.NewPartComponentDefinition())
	return New(opregistry.Default()), s, dir
}

// TestSaveLifecycleOverWire drives documents.save / saveAs / open end to end
// against real packages (#138, M03-F09).
func TestSaveLifecycleOverWire(t *testing.T) {
	t.Parallel()
	r, s, dir := storedSession(t)
	id := uint64(s.ActiveDocument().ID())

	var saved wire.SaveDocumentResult
	call(t, r, s, "documents.save", fmt.Sprintf(`{"document":%d}`, id), &saved)
	if saved.FullDocumentName != filepath.Join(dir, "bracket.opd") {
		t.Fatalf("save = %+v, want the document's path", saved)
	}

	moved := filepath.Join(dir, "bracket-v2.opd")
	call(t, r, s, "documents.saveAs",
		fmt.Sprintf(`{"document":%d,"newFullDocumentName":%q}`, id, moved), &saved)
	if saved.FullDocumentName != moved || s.ActiveDocument().FullFileName() != moved {
		t.Fatalf("saveAs = %+v, want the document retargeted to %s", saved, moved)
	}

	// Close, then reopen over the wire.
	call(t, r, s, "documents.closeAll", `{"force":true}`, nil)
	var info wire.DocumentInfo
	call(t, r, s, "documents.open",
		fmt.Sprintf(`{"fullDocumentName":%q,"visible":true}`, moved), &info)
	if info.Name != "bracket-v2" || !info.Active {
		t.Fatalf("open = %+v, want the reopened active document", info)
	}
}

// TestSaveCopyAsAndBatchOverWire: a copy lands at the target without
// retargeting; the batch reports per-file outcomes (M03-F09).
func TestSaveCopyAsAndBatchOverWire(t *testing.T) {
	t.Parallel()
	r, s, dir := storedSession(t)
	id := uint64(s.ActiveDocument().ID())
	source := s.ActiveDocument().FullFileName()

	target := filepath.Join(dir, "export", "bracket-rev3.opd")
	if _, err := r.Handle(s, "documents.saveCopyAs",
		[]byte(fmt.Sprintf(`{"document":%d,"targetFileName":%q}`, id, target))); err == nil {
		t.Fatal("copying into a nonexistent directory must surface the store error")
	}

	target = filepath.Join(dir, "bracket-rev3.opd")
	var saved wire.SaveDocumentResult
	call(t, r, s, "documents.saveCopyAs",
		fmt.Sprintf(`{"document":%d,"targetFileName":%q,"metadata":{"displayName":"Bracket rev3"}}`, id, target), &saved)
	if saved.FullDocumentName != target || s.ActiveDocument().FullFileName() != source {
		t.Fatalf("saveCopyAs = %+v, want the copy at the target and the source untouched", saved)
	}

	var batch wire.BatchSaveResult
	call(t, r, s, "documents.batchSave", fmt.Sprintf(
		`{"operation":"saveCopyAs","items":[{"document":%d,"targetFileName":%q},{"document":%d,"targetFileName":%q}]}`,
		id, filepath.Join(dir, "a.opd"), id, source), &batch)
	if batch.Saved != 1 || len(batch.Results) != 2 {
		t.Fatalf("batch = %+v, want one success and one carried failure", batch)
	}
	if batch.Results[1].OK || batch.Results[1].Error == "" {
		t.Errorf("results[1] = %+v, want the copy-onto-source failure carried", batch.Results[1])
	}

	if _, err := r.Handle(s, "documents.batchSave",
		[]byte(`{"operation":"archive","items":[]}`)); err == nil {
		t.Error("an unknown batch operation must fail")
	}
}

// TestSaveOptionGroupOverWire: the save group reads and writes through
// options.getGroup/setGroup, rejecting unimplemented thumbnail modes.
func TestSaveOptionGroupOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)

	var groups wire.ListOptionGroupsResult
	call(t, r, s, "options.listGroups", "{}", &groups)
	if len(groups.Groups) != 5 || groups.Groups[4] != "save" {
		t.Fatalf("groups = %v, want save listed", groups.Groups)
	}

	call(t, r, s, "options.setGroup",
		`{"group":"save","save":{"thumbnail":79875,"saveDependents":true,"oldVersionsToKeep":2}}`, nil)
	var view wire.OptionGroupView
	call(t, r, s, "options.getGroup", `{"group":"save"}`, &view)
	if view.Save == nil || view.Save.Thumbnail != types.ThumbnailActiveWindowOnSave ||
		!view.Save.SaveDependents || view.Save.OldVersionsToKeep != 2 {
		t.Fatalf("save view = %+v, want the written policy back", view.Save)
	}

	if _, err := r.Handle(s, "options.setGroup",
		[]byte(`{"group":"save","save":{"thumbnail":79877}}`)); err == nil {
		t.Error("an unimplemented thumbnail mode must be rejected over the wire")
	}
}
