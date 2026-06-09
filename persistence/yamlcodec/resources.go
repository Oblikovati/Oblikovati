// SPDX-License-Identifier: GPL-2.0-only

package yamlcodec

import (
	"encoding/base64"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// sortedKeys returns a map's keys in ascending order, for deterministic (clean-diff) output.
func sortedKeys(m map[string]Resource) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Resource is one embedded imported file in a document's root resource table (ADR-0031):
// a typed, self-contained copy of a file a recipe step consumed, so the document needs
// nothing outside itself to reopen. Value is the file's raw bytes regardless of how they
// are stored on disk; Encoding records that storage choice.
type Resource struct {
	Type     string // extensible type tag: ObjFile/StepFile/StlFile/TrueTypeFont/...
	Encoding string // how Value is stored on disk: EncodingUTF8 (verbatim) | EncodingBase64
	Value    []byte // the file's bytes
	Origin   string // optional: the original filename (display/round-trip metadata only)
}

const (
	// EncodingUTF8 stores Value verbatim as a YAML literal block scalar — for text files
	// (OBJ, ASCII STL, STEP) so they stay human-readable and line-diffable in git.
	EncodingUTF8 = "utf8"
	// EncodingBase64 stores Value base64-encoded — for binary files (fonts, binary STL/3MF).
	EncodingBase64 = "base64"
)

// onDiskResource is the YAML projection of one Resource. Value carries either verbatim text
// (encoding utf8) or a base64 string (encoding base64); a reader decodes by Encoding, never
// by Type (an STL may be ASCII or binary under one tag — ADR-0031 §3).
type onDiskResource struct {
	Type     string `yaml:"type"`
	Encoding string `yaml:"encoding"`
	Origin   string `yaml:"origin,omitempty"`
	Value    string `yaml:"value"`
}

// resourcesNode builds the `resources:` mapping node, keyed by UUID. utf8 values get a
// literal block scalar (the git-diffable text form); base64 values are plain scalars.
func resourcesNode(resources map[string]Resource) (*yaml.Node, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, id := range sortedKeys(resources) {
		r := resources[id]
		stored, style, err := encodeResource(id, r)
		if err != nil {
			return nil, err
		}
		node.Content = append(node.Content, scalar(id), resourceEntryNode(r, stored, style))
	}
	return node, nil
}

// encodeResource returns a resource's on-disk Value string and the scalar style to render it
// with, per its Encoding.
func encodeResource(id string, r Resource) (string, yaml.Style, error) {
	switch r.Encoding {
	case EncodingBase64:
		return base64.StdEncoding.EncodeToString(r.Value), 0, nil
	case EncodingUTF8:
		return string(r.Value), yaml.LiteralStyle, nil
	default:
		return "", 0, fmt.Errorf("yamlcodec: resource %q has unknown encoding %q (want %q or %q)", id, r.Encoding, EncodingUTF8, EncodingBase64)
	}
}

// resourceEntryNode renders one resource as a mapping node (type, encoding, optional origin,
// value), embedding the already-encoded Value with the given scalar style (literal for utf8,
// plain for base64). Built by hand so the value's block style is preserved.
func resourceEntryNode(r Resource, stored string, valueStyle yaml.Style) *yaml.Node {
	entry := &yaml.Node{Kind: yaml.MappingNode}
	entry.Content = append(entry.Content, scalar("type"), scalar(r.Type))
	entry.Content = append(entry.Content, scalar("encoding"), scalar(r.Encoding))
	if r.Origin != "" {
		entry.Content = append(entry.Content, scalar("origin"), scalar(r.Origin))
	}
	entry.Content = append(entry.Content, scalar("value"), &yaml.Node{Kind: yaml.ScalarNode, Style: valueStyle, Value: stored})
	return entry
}

// decodeResources rebuilds the in-memory resource table from the on-disk `resources:` node,
// decoding each Value by its Encoding. An unknown encoding errors (no silent data loss).
func decodeResources(node *yaml.Node) (map[string]Resource, error) {
	if node == nil || node.Kind == 0 {
		return nil, nil
	}
	var raw map[string]onDiskResource
	if err := node.Decode(&raw); err != nil {
		return nil, fmt.Errorf("yamlcodec: parse resources: %w", err)
	}
	if len(raw) == 0 {
		return nil, nil
	}
	out := make(map[string]Resource, len(raw))
	for id, od := range raw {
		value, err := decodeResourceValue(id, od)
		if err != nil {
			return nil, err
		}
		out[id] = Resource{Type: od.Type, Encoding: od.Encoding, Value: value, Origin: od.Origin}
	}
	return out, nil
}

// decodeResourceValue turns a stored value back into raw bytes per its encoding.
func decodeResourceValue(id string, od onDiskResource) ([]byte, error) {
	switch od.Encoding {
	case EncodingUTF8:
		return []byte(od.Value), nil
	case EncodingBase64:
		b, err := base64.StdEncoding.DecodeString(od.Value)
		if err != nil {
			return nil, fmt.Errorf("yamlcodec: resource %q is not valid base64: %w", id, err)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("yamlcodec: resource %q has unknown encoding %q (want %q or %q)", id, od.Encoding, EncodingUTF8, EncodingBase64)
	}
}

func scalar(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Value: v} }
