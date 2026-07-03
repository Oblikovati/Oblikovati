// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"strings"
	"testing"
)

// TestPackageBodyColorStylesSurvivesMarshal checks the per-body color-style names round-trip through
// the .obk YAML (marshal → decode), written as a readable block (S5 #1640) — the same lifecycle body
// names already had, on the same reference keys.
func TestPackageBodyColorStylesSurvivesMarshal(t *testing.T) {
	p := NewPackage()
	if err := p.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "P"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	styles := map[string]string{"body-key-1": "Brass", "body-key-2": "Steel"}
	p.SetBodyColorStyles(styles)

	data, err := p.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), "bodyColorStyles:") || !strings.Contains(string(data), "Brass") {
		t.Errorf("body color styles should be a readable YAML block, got:\n%s", data)
	}

	p2, err := decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := p2.BodyColorStyles()
	if len(got) != 2 || got["body-key-1"] != "Brass" || got["body-key-2"] != "Steel" {
		t.Errorf("decoded body color styles = %+v, want {body-key-1:Brass, body-key-2:Steel}", got)
	}
}

// TestPackageBodyColorStylesAbsentWhenEmpty: a document with no colored body writes no block.
func TestPackageBodyColorStylesAbsentWhenEmpty(t *testing.T) {
	p := NewPackage()
	if err := p.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "P"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	data, err := p.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "bodyColorStyles:") {
		t.Errorf("a document with no colored body must not write a bodyColorStyles block, got:\n%s", data)
	}
}
