// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"errors"
	"fmt"
)

// Migration upgrades a document one schema version forward, in place. It is keyed in
// [migrations] by the version it upgrades FROM.
type Migration func(*Package) error

// migrations maps a from-version to the step that brings a document to the next
// version. It is empty at the v2 (YAML) baseline: the pre-v2 ZIP format is not
// migrated (ADR-0020). Register a step here whenever [CurrentSchemaVersion] is bumped
// past 2; never remove one — it is the only path forward for files in the wild.
var migrations = map[int]Migration{}

// Migrate upgrades a document to [CurrentSchemaVersion], applying each registered step
// in order and stamping the new version into the manifest. A document with no manifest
// (an arbitrary DataIO container) is left untouched. A document from a newer build than
// this one is rejected rather than silently downgraded.
func Migrate(p *Package) error {
	m, err := p.Manifest()
	if errors.Is(err, errNoManifest) {
		return nil
	}
	if err != nil {
		return err
	}
	if m.SchemaVersion > CurrentSchemaVersion {
		return fmt.Errorf("persistence: document schema v%d is newer than supported v%d", m.SchemaVersion, CurrentSchemaVersion)
	}
	for v := m.SchemaVersion; v < CurrentSchemaVersion; v++ {
		step, ok := migrations[v]
		if !ok {
			return fmt.Errorf("persistence: no migration from schema v%d (pre-v2 ZIP packages are unsupported, ADR-0020)", v)
		}
		if err := step(p); err != nil {
			return fmt.Errorf("persistence: migrating from v%d: %w", v, err)
		}
	}
	m.SchemaVersion = CurrentSchemaVersion
	return p.SetManifest(m)
}
