// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"strings"
	"testing"
)

// TestPackageBodyNamesSurvivesMarshal checks the per-body display names round-trip through the
// .obk YAML (marshal → decode), written as a readable block (#1078).
func TestPackageBodyNamesSurvivesMarshal(t *testing.T) {
	p := NewPackage()
	if err := p.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "P"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	names := map[string]string{"body-key-1": "Housing", "body-key-2": "Lid"}
	p.SetBodyNames(names)

	data, err := p.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), "bodyNames:") || !strings.Contains(string(data), "Housing") {
		t.Errorf("body names should be written as a readable YAML block, got:\n%s", data)
	}

	p2, err := decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := p2.BodyNames()
	if len(got) != 2 || got["body-key-1"] != "Housing" || got["body-key-2"] != "Lid" {
		t.Errorf("decoded body names = %+v, want {body-key-1:Housing, body-key-2:Lid}", got)
	}
}

// TestPackageBodyNamesAbsentWhenEmpty: a document with no renamed body writes no bodyNames block.
func TestPackageBodyNamesAbsentWhenEmpty(t *testing.T) {
	p := NewPackage()
	if err := p.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "P"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	data, err := p.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "bodyNames:") {
		t.Errorf("a document with no renamed body must not write a bodyNames block, got:\n%s", data)
	}
}
