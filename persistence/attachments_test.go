// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"oblikovati.org/api/types"
	"oblikovati.org/model/doc"
)

// fakeForeignFiles is a named ExternalFileProbe over in-memory foreign files.
type fakeForeignFiles struct {
	files map[string][]byte
	mod   time.Time
}

func (f *fakeForeignFiles) StatFile(name string) (time.Time, bool) {
	_, ok := f.files[name]
	return f.mod, ok
}

func (f *fakeForeignFiles) ReadFile(name string) ([]byte, error) {
	b, ok := f.files[name]
	if !ok {
		return nil, fmt.Errorf("fake: no file %q", name)
	}
	return b, nil
}

// TestAttachmentsRoundTripThroughPackage: linked and embedded attachments
// persist in the .obk (payload base64) and reload intact in a fresh
// workspace (M03-F08).
func TestAttachmentsRoundTripThroughPackage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bracket.obk")
	store := NewPackageStore()
	ext := &fakeForeignFiles{
		files: map[string][]byte{"/data/loads.csv": []byte("f1,f2\n1,2\n")},
		mod:   time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	ws := doc.NewWorkspace(store)
	ws.SetExternalFileProbe(ext)
	d, err := ws.Add(doc.Part, path, true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := d.Attachments().Add("loads", types.AttachmentLinked, "/data/loads.csv"); err != nil {
		t.Fatalf("Add linked: %v", err)
	}
	if _, err := d.Attachments().Add("table", types.AttachmentEmbedded, "/data/loads.csv"); err != nil {
		t.Fatalf("Add embedded: %v", err)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ws2 := doc.NewWorkspace(store)
	ws2.SetExternalFileProbe(ext)
	reopened, err := ws2.Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := reopened.Attachments().Names(); len(got) != 2 || got[0] != "loads" || got[1] != "table" {
		t.Fatalf("reloaded names = %v, want [loads table] in order", got)
	}
	linked := reopened.Attachments().Record("loads")
	if linked.Kind() != types.AttachmentLinked || !linked.LastKnownFileTime().Equal(ext.mod) {
		t.Errorf("linked = (%v, %v), want the recorded link", linked.Kind(), linked.LastKnownFileTime())
	}
	if linked.Status() != types.ReferenceUpToDate {
		t.Errorf("linked status = %v, want upToDate against the unchanged foreign file", linked.Status())
	}
	embedded := reopened.Attachments().Record("table")
	if string(embedded.Payload()) != "f1,f2\n1,2\n" || embedded.ResourceID() == "" {
		t.Errorf("embedded payload = %q (resource %q), want the bytes carried in the .obk",
			embedded.Payload(), embedded.ResourceID())
	}
}

// TestInterestsRoundTripThroughPackage: the registry persists in a readable
// 'interests' section and reloads with the document (M03-F10).
func TestInterestsRoundTripThroughPackage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bracket.obk")
	store := NewPackageStore()
	ws := doc.NewWorkspace(store)
	d, err := ws.Add(doc.Part, path, true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := d.Interests().Add(types.DocumentInterestRecord{
		ClientID: "com.x.toolpaths", Name: "toolpath-recipes", DataVersion: 3, ClientData: "k",
	}); err != nil {
		t.Fatalf("Add interest: %v", err)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := doc.NewWorkspace(store).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got := reopened.Interests().Records()
	if len(got) != 1 || got[0].DataVersion != 3 || got[0].InterestType != types.Interested {
		t.Fatalf("reloaded interests = %+v, want the persisted record", got)
	}
	if !reopened.Interests().HasInterest("com.x.toolpaths") {
		t.Error("the reloaded registry must answer the discovery probe")
	}
}
