// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/sketch"
)

// addRectangleSketch adds one sketch with a four-line rectangle and a circle to a part,
// returning the part's first line entity for key derivation.
func addRectangleSketch(t *testing.T, part *compdef.PartComponentDefinition) sketch.Entity {
	t.Helper()
	sk := part.Sketches().Add(sketch.XYPlane())
	corners := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 3), math.P2(0, 3)}
	pts := make([]*sketch.Point, len(corners))
	for i, c := range corners {
		pts[i] = sk.NewPoint(c)
	}
	first := sk.Lines().Add(pts[0], pts[1])
	for i := 1; i < len(pts); i++ {
		sk.Lines().Add(pts[i], pts[(i+1)%len(pts)])
	}
	sk.Circles().Add(sk.NewPoint(math.P2(2, 1.5)), 0.5)
	return first
}

// entityKey derives an entity's persistent reference key against a document's GUID, failing
// the test on a derivation error.
func entityKey(t *testing.T, d *doc.Document, e sketch.Entity) string {
	t.Helper()
	k, err := identity.SketchEntityKey(d.FileIdentity().InternalName, uint64(e.EntityID()))
	if err != nil {
		t.Fatalf("derive key: %v", err)
	}
	return k
}

// firstLine returns the first line entity of a reopened part's first sketch.
func firstLine(t *testing.T, part *compdef.PartComponentDefinition) sketch.Entity {
	t.Helper()
	for _, e := range part.Sketches().Item(0).Entities() {
		if _, ok := e.(*sketch.Line); ok {
			return e
		}
	}
	t.Fatal("reopened sketch has no line")
	return nil
}

// TestSketchKeySurvivesRealFileRoundTrip is the integration proof for #153: a persistent
// reference key derived BEFORE a real .obk save still derives identically AFTER reopening
// the file from disk — tying the persistence store, the document GUID, verbatim-id restore,
// and key derivation together (not just the in-memory recipe codec).
func TestSketchKeySurvivesRealFileRoundTrip(t *testing.T) {
	t.Parallel()
	store, ws, dir := assemblyWorkspace(t)
	path := filepath.Join(dir, "keyed.obk")

	partDoc, err := compdef.AddPart(ws, path, true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	part := partDoc.Content().(*compdef.PartComponentDefinition)
	line := addRectangleSketch(t, part)
	wantKey := entityKey(t, partDoc, line)
	wantGUID := partDoc.FileIdentity().InternalName

	if err := ws.Save(partDoc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reopen from disk in a fresh workspace to force the on-disk load path.
	reopenedDoc, err := doc.NewWorkspace(store, contentset.Default()).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// The document GUID — the key's namespace — must survive the file round trip.
	if got := reopenedDoc.FileIdentity().InternalName; got != wantGUID {
		t.Fatalf("document GUID changed across save/reopen: %q -> %q", wantGUID, got)
	}
	reopenedPart := reopenedDoc.Content().(*compdef.PartComponentDefinition)
	gotKey := entityKey(t, reopenedDoc, firstLine(t, reopenedPart))
	if gotKey != wantKey {
		t.Errorf("entity key changed across the real file round trip: %q -> %q", wantKey, gotKey)
	}
}
