// SPDX-License-Identifier: GPL-2.0-only

package yamlcodec

import (
	"bytes"
	"strings"
	"testing"
)

func TestDocumentRoundTrip(t *testing.T) {
	in := Document{
		SchemaVersion: 2,
		DocumentType:  1,
		DisplayName:   "bracket",
		Model:         []byte("parameters:\n  - {name: width, expression: 4 cm}\n"),
		Data:          map[string][]byte{"addins/acme/state.bin": {0xde, 0xad, 0xbe, 0xef, 0x00}},
	}
	raw, err := MarshalDocument(in)
	if err != nil {
		t.Fatalf("MarshalDocument: %v", err)
	}
	out, err := UnmarshalDocument(raw)
	if err != nil {
		t.Fatalf("UnmarshalDocument: %v", err)
	}
	if out.SchemaVersion != 2 || out.DocumentType != 1 || out.DisplayName != "bracket" {
		t.Errorf("identity round trip = %+v, want v2 part bracket", out)
	}
	if !bytes.Equal(out.Data["addins/acme/state.bin"], in.Data["addins/acme/state.bin"]) {
		t.Errorf("binary data section not byte-identical: %v", out.Data)
	}
	// The model must come back parseable to the same content.
	if !strings.Contains(string(out.Model), "width") || !strings.Contains(string(out.Model), "4 cm") {
		t.Errorf("model lost on round trip: %q", out.Model)
	}
}

func TestModelEmbeddedNatively(t *testing.T) {
	raw, err := MarshalDocument(Document{
		SchemaVersion: 2, DocumentType: 1, DisplayName: "p",
		Model: []byte("parameters:\n  - {name: width, expression: 4 cm}\n"),
	})
	if err != nil {
		t.Fatalf("MarshalDocument: %v", err)
	}
	text := string(raw)
	// Native nesting: the model is real YAML under `model:`, not a quoted/escaped blob.
	if !strings.Contains(text, "model:") || !strings.Contains(text, "parameters:") {
		t.Errorf("model not embedded as native YAML:\n%s", text)
	}
	if strings.Contains(text, "\\n") {
		t.Errorf("model appears escaped as a string, not embedded:\n%s", text)
	}
}

func TestLegacyZipRejected(t *testing.T) {
	if _, err := UnmarshalDocument([]byte("PK\x03\x04and the rest")); err == nil {
		t.Error("UnmarshalDocument accepted ZIP magic; legacy packages must be rejected")
	}
}

func TestNonDocumentScalarRejected(t *testing.T) {
	if _, err := UnmarshalDocument([]byte("just a scalar string")); err == nil {
		t.Error("UnmarshalDocument accepted a bare scalar as a document")
	}
}

func TestEmptyModelOmitted(t *testing.T) {
	raw, err := MarshalDocument(Document{SchemaVersion: 2, DocumentType: 1, DisplayName: "p"})
	if err != nil {
		t.Fatalf("MarshalDocument: %v", err)
	}
	if strings.Contains(string(raw), "model:") {
		t.Errorf("empty model should be omitted, got:\n%s", raw)
	}
}
