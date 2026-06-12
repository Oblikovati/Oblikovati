// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"
	"time"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// fakeAddinFiles is a named ExternalFileProbe for the wire tests.
type fakeAddinFiles struct {
	files map[string][]byte
	mod   time.Time
}

func (f *fakeAddinFiles) StatFile(name string) (time.Time, bool) {
	_, ok := f.files[name]
	return f.mod, ok
}

func (f *fakeAddinFiles) ReadFile(name string) ([]byte, error) {
	b, ok := f.files[name]
	if !ok {
		return nil, fmt.Errorf("fake: no file %q", name)
	}
	return b, nil
}

// TestAttachmentsOverWire drives documents.addAttachment / listAttachments /
// removeAttachment end to end (M03-F08, #609).
func TestAttachmentsOverWire(t *testing.T) {
	r, s := seededSession(t)
	s.Workspace().SetExternalFileProbe(&fakeAddinFiles{
		files: map[string][]byte{"/data/loads.csv": []byte("f1,f2\n")},
		mod:   time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	})
	id := uint64(s.ActiveDocument().ID())

	var rec wire.AttachmentInfo
	call(t, r, s, "documents.addAttachment",
		fmt.Sprintf(`{"document":%d,"name":"loads","kind":3331,"fullFileName":"/data/loads.csv"}`, id), &rec)
	if rec.Kind != types.AttachmentLinked || rec.Status != types.ReferenceUpToDate || rec.LastKnownFileTime == "" {
		t.Fatalf("addAttachment = %+v, want a fresh linked record", rec)
	}

	var lst wire.ListAttachmentsResult
	call(t, r, s, "documents.listAttachments", fmt.Sprintf(`{"document":%d}`, id), &lst)
	if len(lst.Attachments) != 1 || lst.Attachments[0].Name != "loads" {
		t.Fatalf("attachments = %+v, want [loads]", lst.Attachments)
	}

	if _, err := r.Handle(s, "documents.addAttachment",
		[]byte(fmt.Sprintf(`{"document":%d,"name":"loads","kind":3329,"fullFileName":"/x"}`, id))); err == nil {
		t.Error("a duplicate attachment name must fail over the wire")
	}

	call(t, r, s, "documents.removeAttachment", fmt.Sprintf(`{"document":%d,"name":"loads"}`, id), nil)
	if _, err := r.Handle(s, "documents.removeAttachment",
		[]byte(fmt.Sprintf(`{"document":%d,"name":"loads"}`, id))); err == nil {
		t.Error("removing a missing attachment must fail")
	}
}
