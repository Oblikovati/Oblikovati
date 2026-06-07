// SPDX-License-Identifier: GPL-2.0-only

// Package yamlcodec is the project's single point of contact with the YAML library
// (gopkg.in/yaml.v3). Per the dependency rule (CLAUDE.md) and ADR-0020, only this
// package imports yaml — everything else marshals through these functions, so a
// future library swap stays local.
//
// It also owns the on-disk .obk document shape: one readable YAML file whose manifest
// fields sit at the top level and whose recipe is a NATIVE nested node (not a quoted
// blob), so a model diffs line-by-line in git. Binary data sections (add-in/attribute
// scratch — the rare non-recipe payload) are base64-encoded, the one concession to
// non-text data (ADR-0020).
package yamlcodec

import (
	"encoding/base64"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Marshal renders v as YAML bytes.
func Marshal(v any) ([]byte, error) { return yaml.Marshal(v) }

// Unmarshal parses YAML bytes into v.
func Unmarshal(data []byte, v any) error { return yaml.Unmarshal(data, v) }

// Document is the decoded form of a .obk file: identity, the recipe section (raw YAML
// bytes, embedded natively on disk for readability), and named binary data sections.
// Model is nil when the document carries no recipe (e.g. a manifest-only stub or a
// pure DataIO container).
type Document struct {
	SchemaVersion int
	DocumentType  uint32
	DisplayName   string
	Model         []byte
	Data          map[string][]byte
	Views         *ViewsSection
}

// ViewFrame is the persisted camera of one view: a name and a look-at frame (eye, target,
// up in model units; fov in radians). Stored natively (not base64) so the saved view is
// readable in the YAML file.
type ViewFrame struct {
	Name   string     `yaml:"name,omitempty"`
	Eye    [3]float64 `yaml:"eye"`
	Target [3]float64 `yaml:"target"`
	Up     [3]float64 `yaml:"up"`
	FOV    float64    `yaml:"fov"`
}

// ViewsSection is a document's persisted view collection: the views, the active index, and
// the tiling layout. A loaded view is, by definition, framed — the framed flag is not
// persisted; the host marks loaded views framed on restore.
type ViewsSection struct {
	Views  []ViewFrame `yaml:"views"`
	Active int         `yaml:"active,omitempty"`
	Layout int32       `yaml:"layout,omitempty"`
}

// onDisk is the YAML projection of a Document: manifest at top level, recipe as a
// native node, data sections base64-encoded. omitempty keeps a minimal file readable.
type onDisk struct {
	SchemaVersion int               `yaml:"schemaVersion,omitempty"`
	DocumentType  uint32            `yaml:"documentType,omitempty"`
	DisplayName   string            `yaml:"displayName,omitempty"`
	Model         yaml.Node         `yaml:"model,omitempty"`
	Views         *ViewsSection     `yaml:"views,omitempty"`
	Data          map[string]string `yaml:"data,omitempty"`
}

// MarshalDocument renders d as the on-disk YAML file. The recipe bytes are parsed and
// embedded as a native node so the model is real nested YAML, not an escaped string.
func MarshalDocument(d Document) ([]byte, error) {
	od := onDisk{
		SchemaVersion: d.SchemaVersion,
		DocumentType:  d.DocumentType,
		DisplayName:   d.DisplayName,
	}
	if len(d.Model) > 0 {
		node, err := modelNode(d.Model)
		if err != nil {
			return nil, err
		}
		od.Model = *node
	}
	od.Views = d.Views
	if len(d.Data) > 0 {
		od.Data = make(map[string]string, len(d.Data))
		for name, raw := range d.Data {
			od.Data[name] = base64.StdEncoding.EncodeToString(raw)
		}
	}
	return yaml.Marshal(od)
}

// UnmarshalDocument decodes a .obk file's bytes. It rejects a legacy ZIP package with
// a clear message (ADR-0020: the format is now YAML) and surfaces base64/YAML errors.
func UnmarshalDocument(raw []byte) (Document, error) {
	if isZip(raw) {
		return Document{}, errors.New("yamlcodec: this looks like a legacy ZIP .obk; the document format is now a YAML text file (ADR-0020) and old ZIP packages are not supported")
	}
	var od onDisk
	if err := yaml.Unmarshal(raw, &od); err != nil {
		return Document{}, fmt.Errorf("yamlcodec: parse document: %w", err)
	}
	d := Document{
		SchemaVersion: od.SchemaVersion,
		DocumentType:  od.DocumentType,
		DisplayName:   od.DisplayName,
	}
	if od.Model.Kind != 0 {
		b, err := yaml.Marshal(&od.Model)
		if err != nil {
			return Document{}, fmt.Errorf("yamlcodec: extract model: %w", err)
		}
		d.Model = b
	}
	d.Views = od.Views
	data, err := decodeDataSections(od.Data)
	if err != nil {
		return Document{}, err
	}
	d.Data = data
	return d, nil
}

// decodeDataSections base64-decodes the on-disk data sections into raw bytes, or nil when
// there are none.
func decodeDataSections(enc map[string]string) (map[string][]byte, error) {
	if len(enc) == 0 {
		return nil, nil
	}
	out := make(map[string][]byte, len(enc))
	for name, s := range enc {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("yamlcodec: data section %q is not valid base64: %w", name, err)
		}
		out[name] = b
	}
	return out, nil
}

// modelNode parses recipe YAML bytes into the mapping node to embed under `model:`.
// yaml.Unmarshal yields a document node wrapping the real content; we splice in that
// content root so the embedded model is not double-wrapped.
func modelNode(modelYAML []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(modelYAML, &doc); err != nil {
		return nil, fmt.Errorf("yamlcodec: embed model: %w", err)
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		return doc.Content[0], nil
	}
	return &doc, nil
}

// isZip reports whether raw begins with the local-file-header magic of a ZIP archive
// ("PK\x03\x04") — i.e. a pre-ADR-0020 package.
func isZip(raw []byte) bool {
	return len(raw) >= 4 && raw[0] == 'P' && raw[1] == 'K' && raw[2] == 0x03 && raw[3] == 0x04
}
