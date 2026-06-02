// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"encoding/json"
	"fmt"
)

// CurrentSchemaVersion is the package schema version written by this build. Bump
// it whenever the on-disk layout changes and add a [Migration] from the prior
// version so old files keep opening (architecture core/05).
const CurrentSchemaVersion = 1

// manifestStream is the well-known stream holding the package manifest.
const manifestStream = "manifest.json"

// Manifest is the package header: schema version (for migration), the root
// document kind, the display name, and an optional thumbnail stream reference. It
// is the JSON document at [manifestStream] and is the only part of the package the
// store needs to reconstruct a document's identity today.
type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	DocumentType  uint32 `json:"documentType"`
	DisplayName   string `json:"displayName"`
	Thumbnail     string `json:"thumbnail,omitempty"`
}

// encode renders the manifest as indented JSON so the package stays diff-friendly.
func (m Manifest) encode() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// decodeManifest parses a manifest stream.
func decodeManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("persistence: invalid manifest: %w", err)
	}
	return m, nil
}
