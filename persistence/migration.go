// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"errors"
	"fmt"
)

// Migration upgrades a package one schema version forward, in place. It is keyed
// in [migrations] by the version it upgrades FROM.
type Migration func(*Package) error

// migrations maps a from-version to the step that brings a package to the next
// version. Register a new entry whenever [CurrentSchemaVersion] is bumped; never
// remove old ones — they are the only path forward for files in the wild.
var migrations = map[int]Migration{
	0: migrateV0toV1,
}

// Migrate upgrades a document package to [CurrentSchemaVersion], applying each
// registered step in order and stamping the new version into the manifest. A
// package with no manifest (an arbitrary DataIO container) is left untouched. A
// package from a newer build than this one is rejected rather than silently
// downgraded.
func Migrate(p *Package) error {
	m, err := p.Manifest()
	if errors.Is(err, errNoManifest) {
		return nil
	}
	if err != nil {
		return err
	}
	if m.SchemaVersion > CurrentSchemaVersion {
		return fmt.Errorf("persistence: package schema v%d is newer than supported v%d", m.SchemaVersion, CurrentSchemaVersion)
	}
	for v := m.SchemaVersion; v < CurrentSchemaVersion; v++ {
		step, ok := migrations[v]
		if !ok {
			return fmt.Errorf("persistence: no migration from schema v%d", v)
		}
		if err := step(p); err != nil {
			return fmt.Errorf("persistence: migrating from v%d: %w", v, err)
		}
	}
	m.SchemaVersion = CurrentSchemaVersion
	return p.SetManifest(m)
}

// migrateV0toV1 renames the v0 parameter stream to its v1 name. v0 packages stored
// parameters under "model/params.bin"; v1 standardized on "model/parameters.bin".
// The bytes are carried over unchanged, so no data is lost.
func migrateV0toV1(p *Package) error {
	if data, ok := p.ReadStream("model/params.bin"); ok {
		p.WriteStream("model/parameters.bin", data)
		p.DeleteStream("model/params.bin")
	}
	return nil
}
