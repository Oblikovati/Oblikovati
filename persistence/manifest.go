// SPDX-License-Identifier: GPL-2.0-only

package persistence

// CurrentSchemaVersion is the document schema version written by this build. Bump it
// whenever the on-disk layout changes and add a [Migration] from the prior version so
// old files keep opening (architecture core/05). v2 is the YAML single-file format
// (ADR-0020); v1 and earlier were the ZIP-package format and are not migrated (the
// only pre-v2 files were regenerable fixtures). v3 adds the optional `views:` section
// (per-document cameras) — additive, so the v2→v3 step is a no-op (a v2 file simply has
// no views and the host seeds a default view on open).
const CurrentSchemaVersion = 3

// Manifest is the document's identity header: schema version (for migration), the
// root document kind, and the display name. On disk these are the top-level fields of
// the YAML document (see persistence/yamlcodec); in memory the [Package] holds it as
// a value. It is the only part of the document the store needs to reconstruct a
// document's identity.
type Manifest struct {
	SchemaVersion int
	DocumentType  uint32
	DisplayName   string
}
