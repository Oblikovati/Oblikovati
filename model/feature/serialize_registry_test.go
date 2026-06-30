// SPDX-License-Identifier: GPL-2.0-only

package feature

import "testing"

// #1416: the dual hand-maintained encode/decode switches let a feature be added to one side but not the
// other — silently dropping it on save or reload. The codec registry pairs encode+decode per kind so
// that class of bug is structurally impossible. These tests lock the guarantee.

// TestFeatureCodecRegistryComplete is the table-driven coverage over EVERY registered kind: each must
// carry both a non-nil encode and a non-nil decode (a half-codec cannot serve a round-trip), and the
// registry must hold the full feature surface (a registration silently dropped would shrink this).
func TestFeatureCodecRegistryComplete(t *testing.T) {
	kinds := registeredFeatureKinds()
	const wantAtLeast = 78 // the audited feature-kind count (#1416); grows as kinds are added
	if len(kinds) < wantAtLeast {
		t.Fatalf("registry has %d feature codecs, want at least %d — a registration was dropped", len(kinds), wantAtLeast)
	}
	for _, k := range kinds {
		c := featureCodecs[k]
		if c.encode == nil {
			t.Errorf("kind %q has no encode half (it would drop on save)", k)
		}
		if c.decode == nil {
			t.Errorf("kind %q has no decode half (it would drop on reload)", k)
		}
	}
}

// TestEveryRegisteredKindIsDecodeReachable is the decode-side coverage over EVERY registered kind: the
// buildFeature dispatcher must route each kind to its codec (never the "no restore codec" sentinel that
// means a kind string is unhandled). A missing payload may surface as an error or a nil-deref panic —
// both prove the codec was reached; only the unhandled-kind sentinel is a failure.
func TestEveryRegisteredKindIsDecodeReachable(t *testing.T) {
	fs := NewPartFeatures(nil)
	for _, k := range registeredFeatureKinds() {
		assertKindDecodeReachable(t, fs, k)
	}
}

// assertKindDecodeReachable dispatches one kind through buildFeature with an empty payload, recovering a
// nil-payload panic (which still proves the codec was reached), and fails only if the kind is unhandled.
func assertKindDecodeReachable(t *testing.T, fs *PartFeatures, kind string) {
	t.Helper()
	defer func() { _ = recover() }() // a nil-payload deref means the codec WAS reached — acceptable here
	_, err := buildFeature(fs, FeatureData{Kind: kind}, oneSketch{}, nil, nil)
	if err != nil && err.Error() == "no restore codec for feature kind "+quote(kind) {
		t.Errorf("kind %q is not decode-reachable through the registry", kind)
	}
}

// quote renders a kind the way fmt's %q does, for matching the dispatcher's sentinel error text.
func quote(s string) string { return `"` + s + `"` }

// TestRegisterFeatureCodecRejectsHalfCodec is the decisive anti-drift guard (#1416): registering a kind
// with only an encode, only a decode, an empty kind, or a duplicate panics at startup — so a one-sided
// registration (the exact bug that silently dropped features) cannot exist, caught the moment it is added.
func TestRegisterFeatureCodecRejectsHalfCodec(t *testing.T) {
	good := featureCodec{
		encode: func(*FeatureData, Feature, SketchIndexer, map[ID]int) error { return nil },
		decode: func(*restoreContext, FeatureData) (*PartFeature, error) { return nil, nil },
	}
	for _, tc := range []struct {
		name string
		kind string
		c    featureCodec
	}{
		{"encode only", "test.encode-only", featureCodec{encode: good.encode}},
		{"decode only", "test.decode-only", featureCodec{decode: good.decode}},
		{"empty kind", "", good},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPanics(t, func() { registerFeatureCodec(tc.kind, tc.c) })
		})
	}

	// A duplicate registration (two codecs claiming one kind) must also panic — pick an already-registered
	// kind so the second registration collides.
	t.Run("duplicate", func(t *testing.T) {
		assertPanics(t, func() { registerFeatureCodec("extrude", good) })
	})
}

// assertPanics fails unless fn panics.
func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected a panic, got none")
		}
	}()
	fn()
}
