// SPDX-License-Identifier: GPL-2.0-only

package persistence

// CurrentSchemaVersion is the document schema version written by this build. Bump it
// whenever the on-disk layout changes and add a [Migration] from the prior version so
// old files keep opening (architecture core/05). v2 is the YAML single-file format
// (ADR-0020); v1 and earlier were the ZIP-package format and are not migrated (the
// only pre-v2 files were regenerable fixtures).
const CurrentSchemaVersion = 2

// Manifest is the document's identity header: schema version (for migration), the
// root document kind, and the display name. On disk these are the top-level fields of
// the YAML document (see persistence/yamlcodec); in memory the [Package] holds it as
// a value. It is the only part of the document the store needs to reconstruct a
// document's identity.
type Manifest struct {
	SchemaVersion int
	DocumentType  uint32
	SubType       string // add-in flavored subtype id (M05-F15)
	DisplayName   string
}
