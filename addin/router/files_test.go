// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/model/doc"
)

// TestFileSurfaceOverWire covers files.get / files.listReferences against the
// seeded session (M03-F07, #608).
func TestFileSurfaceOverWire(t *testing.T) {
	r, s := seededSession(t)
	d := s.ActiveDocument()

	var info wire.FileInfo
	call(t, r, s, "files.get", `{"fullFileName":"test.obk"}`, &info)
	if info.InternalName == "" || !info.Loaded || len(info.Documents) != 1 {
		t.Fatalf("files.get = %+v, want the loaded file's identity", info)
	}
	if info.Documents[0] != uint64(d.ID()) {
		t.Errorf("contained documents = %v, want [%d]", info.Documents, d.ID())
	}
	if info.SaveCounter != 0 || info.RevisionID != "" {
		t.Errorf("never-saved file reports %+v, want no revision stamps", info)
	}

	var refs wire.ListFileReferencesResult
	call(t, r, s, "files.listReferences", `{"fullFileName":"test.obk"}`, &refs)
	if len(refs.References) != 0 {
		t.Errorf("references = %+v, want none on the standalone part", refs.References)
	}

	if _, err := r.Handle(s, "files.get", []byte(`{"fullFileName":"ghost.obk"}`)); err == nil {
		t.Error("files.get must fail for a file that is not open")
	}
}

// TestDocumentFileReferenceViewCarriesStatus: persisted records surface over
// documents.listFileReferences with the derived status; an unresolvable record
// reports missing (the broken-reference inspection path, M03-F07).
func TestDocumentFileReferenceViewCarriesStatus(t *testing.T) {
	r, s := seededSession(t)
	d := s.ActiveDocument()
	d.SetFileReferenceRecords([]doc.FileReferenceRecord{{
		FullFileName: "/asm/parts/gone-pin.obk", RelativeFileName: "parts/gone-pin.obk",
		LocationType: "ownerDirectory", SaveCounter: 3,
	}})

	var docRefs wire.ListDocumentFileReferencesResult
	call(t, r, s, "documents.listFileReferences",
		fmt.Sprintf(`{"document":%d}`, d.ID()), &docRefs)
	if len(docRefs.References) != 1 {
		t.Fatalf("references = %+v, want the seeded record", docRefs.References)
	}
	got := docRefs.References[0]
	if got.Status != types.ReferenceMissing || got.DocumentFound {
		t.Errorf("view = %+v, want a missing reference (no store can resolve it)", got)
	}
	if got.DisplayName != "gone-pin" {
		t.Errorf("display name = %q, want gone-pin", got.DisplayName)
	}

	if _, err := r.Handle(s, "files.replaceReference",
		[]byte(`{"fullFileName":"test.obk","requestedName":"/asm/parts/gone-pin.obk","newFileName":"/nope.obk"}`)); err == nil {
		t.Error("repairing toward a nonexistent target must fail")
	}
}
