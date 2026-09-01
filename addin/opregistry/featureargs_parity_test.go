// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/api/wire/featureargs"
)

// Every feature kind add_feature can create must have a typed argument struct in
// api/wire/featureargs (so an add-in builds it with a compile-checked type, not a raw
// JSON blob) OR be an explicit, tracked exception in dynamicKinds. This is the wire<->host
// parity guard for feature creation (#1616, audit B5), the same shape as the
// wire/router/client registration parity guard. Because opregistry now decodes into the
// SAME featureargs types the client marshals, a promoted kind's wire shape and host decoder
// cannot drift — the round-trip is guaranteed by type identity (the featureargs package's
// own marshal round-trip test proves each type is JSON-symmetric).

// dynamicKinds are the registered kinds NOT promoted to featureargs — the exception list the
// parity guard tolerates. #1709 promoted the remaining 63 kinds (composite, multi-kind, args-less,
// and mechanical-remainder) to typed featureargs structs, so this is now EMPTY: every registered
// feature kind has a typed argument struct. A truly dynamic add-in-registered op with no compile-
// time contract would be the only reason to add an entry back here (with its own justification).
var dynamicKinds = map[string]string{}

func TestEveryRegisteredKindHasWireArgsOrIsAllowlisted(t *testing.T) {
	t.Parallel()
	promoted := map[string]bool{}
	for _, k := range featureargs.Kinds() {
		promoted[k] = true
	}
	registered := map[string]bool{}
	for _, d := range Default().All() {
		registered[d.Name] = true
		if promoted[d.Name] {
			continue
		}
		if _, ok := dynamicKinds[d.Name]; !ok {
			t.Errorf("feature kind %q has no api/wire/featureargs arg struct and is not in "+
				"dynamicKinds — promote it (define featureargs.X with Kind()==%q, make its "+
				"descriptor decode that type) or allowlist it with the tracking issue (#1616/#1709).",
				d.Name, d.Name)
		}
	}
	reportStaleDynamicKinds(t, registered, promoted)
}

// reportStaleDynamicKinds keeps dynamicKinds honest: an entry that is now promoted, or that
// names no registered kind, must be deleted.
func reportStaleDynamicKinds(t *testing.T, registered, promoted map[string]bool) {
	for kind, why := range dynamicKinds {
		if promoted[kind] {
			t.Errorf("dynamicKinds[%q] (%s) is stale — %q is promoted to featureargs now; delete the entry.", kind, why, kind)
		}
		if !registered[kind] {
			t.Errorf("dynamicKinds[%q] (%s) names no registered feature kind; delete the entry.", kind, why)
		}
	}
}
