// SPDX-License-Identifier: GPL-2.0-only

package yamlcodec

import (
	"bytes"
	"strings"
	"testing"
)

// TestResourceRoundTrip checks a utf8 (text) and a base64 (binary) resource survive a
// marshal→unmarshal cycle with their bytes intact (ADR-0031).
func TestResourceRoundTrip(t *testing.T) {
	in := Document{
		SchemaVersion: 2,
		DocumentType:  1,
		Resources: map[string]Resource{
			"9f1c2e7a-5b40-4d3e-9a21-2c7e0b8f4d61": {
				Type: "StepFile", Encoding: EncodingUTF8, Origin: "bracket.step",
				Value: []byte("ISO-10303-21;\nHEADER;\nEND-ISO-10303-21;\n"),
			},
			"3d6b1f08-9e2a-4c11-8f7d-1a5e6c904b22": {
				Type: "TrueTypeFont", Encoding: EncodingBase64, Origin: "Arial.ttf",
				Value: []byte{0x00, 0x01, 0x00, 0x00, 0xFF, 0xAB},
			},
			"7c4a1b00-0000-4000-8000-000000000001": {
				Type: "TrueTypeFont", Encoding: EncodingEmbedded, Origin: "Liberation Sans",
			},
		},
	}
	raw, err := MarshalDocument(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// The text resource stays human-readable as a literal block scalar (git-diffable).
	if !strings.Contains(string(raw), "value: |") {
		t.Errorf("utf8 resource not emitted as a literal block scalar:\n%s", raw)
	}
	out, err := UnmarshalDocument(raw)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for id, want := range in.Resources {
		got, ok := out.Resources[id]
		if !ok {
			t.Fatalf("resource %s lost in round-trip", id)
		}
		if got.Type != want.Type || got.Encoding != want.Encoding || got.Origin != want.Origin {
			t.Errorf("resource %s metadata = %+v, want type/encoding/origin %s/%s/%s", id, got, want.Type, want.Encoding, want.Origin)
		}
		if !bytes.Equal(got.Value, want.Value) {
			t.Errorf("resource %s bytes = %v, want %v", id, got.Value, want.Value)
		}
	}
}

// TestNoResourcesIsClean confirms a document without resources emits no `resources:` key.
func TestNoResourcesIsClean(t *testing.T) {
	raw, err := MarshalDocument(Document{SchemaVersion: 2, DocumentType: 1, DisplayName: "p"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "resources:") {
		t.Errorf("empty document emitted a resources key:\n%s", raw)
	}
}

// TestUnknownEncodingRejected ensures an unknown encoding fails loudly (no silent data loss).
func TestUnknownEncodingRejected(t *testing.T) {
	_, err := MarshalDocument(Document{Resources: map[string]Resource{
		"id": {Type: "StlFile", Encoding: "ascii", Value: []byte("x")},
	}})
	if err == nil || !strings.Contains(err.Error(), "unknown encoding") {
		t.Fatalf("want unknown-encoding error, got %v", err)
	}
}
