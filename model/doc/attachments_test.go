// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"
	"testing"
	"time"

	"oblikovati.org/api/types"
)

// fakeExternalFiles is a named in-memory ExternalFileProbe: foreign files with
// controllable bytes and modification times.
type fakeExternalFiles struct {
	files map[string]fakeExternalFile
}

type fakeExternalFile struct {
	bytes []byte
	mod   time.Time
}

func newFakeExternalFiles() *fakeExternalFiles {
	return &fakeExternalFiles{files: map[string]fakeExternalFile{}}
}

func (f *fakeExternalFiles) put(name string, bytes []byte, mod time.Time) {
	f.files[name] = fakeExternalFile{bytes: bytes, mod: mod}
}

func (f *fakeExternalFiles) StatFile(name string) (time.Time, bool) {
	file, ok := f.files[name]
	return file.mod, ok
}

func (f *fakeExternalFiles) ReadFile(name string) ([]byte, error) {
	file, ok := f.files[name]
	if !ok {
		return nil, fmt.Errorf("fake: no file %q", name)
	}
	return file.bytes, nil
}

// attachmentFixture is a part document in a workspace with a fake foreign
// filesystem holding loads.csv.
func attachmentFixture(t *testing.T) (*Document, *fakeExternalFiles, time.Time) {
	t.Helper()
	ws := NewWorkspace(newFakeStore())
	ext := newFakeExternalFiles()
	ws.SetExternalFileProbe(ext)
	attached := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	ext.put("/data/loads.csv", []byte("f1,f2\n1,2\n"), attached)
	d, err := ws.Add(Part, "bracket.obk", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return d, ext, attached
}

// TestLinkedAttachmentTracksFreshness: a linked attachment records the mod
// time at attach and reports outOfDate / missing as the foreign file changes
// or vanishes (M03-F08).
func TestLinkedAttachmentTracksFreshness(t *testing.T) {
	d, ext, attached := attachmentFixture(t)
	if _, err := d.Attachments().Add("loads", types.AttachmentLinked, "/data/loads.csv"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	a := d.Attachments().Record("loads")
	if !a.LastKnownFileTime().Equal(attached) || a.Status() != types.ReferenceUpToDate {
		t.Fatalf("fresh link = (%v, %v), want upToDate at the attach time", a.LastKnownFileTime(), a.Status())
	}
	if a.ResolvedFileName() != "/data/loads.csv" {
		t.Errorf("resolved = %q, want the linked path", a.ResolvedFileName())
	}

	ext.put("/data/loads.csv", []byte("changed"), attached.Add(time.Hour))
	if a.Status() != types.ReferenceOutOfDate {
		t.Errorf("status after foreign edit = %v, want outOfDate", a.Status())
	}
	delete(ext.files, "/data/loads.csv")
	if a.Status() != types.ReferenceMissing || a.ResolvedFileName() != "" {
		t.Errorf("status after deletion = %v, want missing", a.Status())
	}
}

// TestEmbeddedAttachmentCarriesPayload: an embedded attachment reads the file
// once and stays upToDate even when the origin vanishes.
func TestEmbeddedAttachmentCarriesPayload(t *testing.T) {
	d, ext, _ := attachmentFixture(t)
	if _, err := d.Attachments().Add("loads", types.AttachmentEmbedded, "/data/loads.csv"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	a := d.Attachments().Record("loads")
	if string(a.Payload()) != "f1,f2\n1,2\n" || a.ResourceID() == "" {
		t.Fatalf("payload = %q (resource %q), want the embedded bytes addressable", a.Payload(), a.ResourceID())
	}
	delete(ext.files, "/data/loads.csv")
	if a.Status() != types.ReferenceUpToDate {
		t.Errorf("embedded status = %v, want upToDate regardless of the origin", a.Status())
	}
	if _, err := d.Attachments().Add("ghost", types.AttachmentEmbedded, "/data/ghost.bin"); err == nil {
		t.Error("embedding an unreadable file must fail")
	}
}

// TestAttachmentCollectionRules: unique names, dirty marking, removal.
func TestAttachmentCollectionRules(t *testing.T) {
	d, _, _ := attachmentFixture(t)
	d.ClearDirty()
	if _, err := d.Attachments().Add("loads", types.AttachmentGeneric, "/data/loads.csv"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !d.Dirty() {
		t.Error("attaching must mark the document dirty")
	}
	if _, err := d.Attachments().Add("loads", types.AttachmentLinked, "/data/other.csv"); err == nil {
		t.Error("a duplicate attachment name must fail")
	}
	if _, err := d.Attachments().Add("", types.AttachmentLinked, "/x"); err == nil {
		t.Error("an empty attachment name must fail")
	}
	if got := d.Attachments().Names(); len(got) != 1 || got[0] != "loads" {
		t.Errorf("names = %v, want [loads]", got)
	}
	if !d.Attachments().Remove("loads") || d.Attachments().Remove("loads") {
		t.Error("Remove must report existence exactly once")
	}
	if d.Attachments().Count() != 0 {
		t.Errorf("count after removal = %d, want 0", d.Attachments().Count())
	}
}

// TestAttachmentRecordsRoundTrip: the persistence bridge keeps payloads,
// times and flags across SetAttachmentRecords(AttachmentRecords()).
func TestAttachmentRecordsRoundTrip(t *testing.T) {
	d, _, attached := attachmentFixture(t)
	if _, err := d.Attachments().Add("loads", types.AttachmentLinked, "/data/loads.csv"); err != nil {
		t.Fatalf("Add linked: %v", err)
	}
	if _, err := d.Attachments().Add("table", types.AttachmentEmbedded, "/data/loads.csv"); err != nil {
		t.Fatalf("Add embedded: %v", err)
	}
	d.Attachments().Record("table").SetBrowserVisible(false)

	recs := d.AttachmentRecords()
	d.SetAttachmentRecords(recs)

	linked := d.Attachments().Record("loads")
	if !linked.LastKnownFileTime().Equal(attached) || linked.Kind() != types.AttachmentLinked {
		t.Errorf("linked after round-trip = (%v, %v), want the recorded time kept",
			linked.LastKnownFileTime(), linked.Kind())
	}
	embedded := d.Attachments().Record("table")
	if string(embedded.Payload()) != "f1,f2\n1,2\n" || embedded.BrowserVisible() {
		t.Errorf("embedded after round-trip = (%q, visible=%v), want payload kept and hidden",
			embedded.Payload(), embedded.BrowserVisible())
	}
}
