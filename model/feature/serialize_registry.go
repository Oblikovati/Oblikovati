// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Feature serialization is a registry keyed by Feature.Kind(): each kind contributes ONE
// featureCodec pairing its encode and decode closures, registered together in its family's
// serialize_codecs_*.go init(). Replacing the two hand-maintained switches (a 78-case type switch to
// encode, a mirrored kind-string switch to decode, with nothing linking them) closes the worst failure
// class — adding a feature to one side but not the other silently dropped it on save or reload (#1416).
// Now a kind without BOTH halves does not round-trip, and an enumerating test (serialize_registry_test)
// asserts every registered kind survives marshal→unmarshal. Adding a feature type is one registration in
// one place; encode and decode can never drift.

// featureCodec is one kind's serialization: encode writes the feature's payload into a base FeatureData
// (Kind/Name/Suppressed/Seq already filled), decode rebuilds the feature from that FeatureData. Both are
// registered as a pair so neither can be added without the other.
type featureCodec struct {
	encode func(fd *FeatureData, f Feature, sk SketchIndexer, idx map[ID]int) error
	decode func(rc *restoreContext, fd FeatureData) (*PartFeature, error)
}

// restoreContext bundles everything a decode closure may need: the target feature collection, the
// sketch index, the features restored so far (a pattern/mirror resolves the earlier features it
// replicates), and the work geometry (an extrude/revolve may reference a work plane/axis).
type restoreContext struct {
	fs       *PartFeatures
	sk       SketchIndexer
	restored []*PartFeature
	work     *WorkGeometry
}

// featureCodecSet maps feature kinds to their codecs — the value serializeFeature/buildFeature
// consult, and the seam a test can inject a minimal set through (#1617, audit B6).
type featureCodecSet map[string]featureCodec

// featureCodecs is the package default: every production codec, constructed in ONE
// visible order by defaultFeatureCodecs. This is the package's composition site —
// there are no init() registrations (#1617), so what serializes is decided by
// explicit construction, not import linkage.
var featureCodecs = defaultFeatureCodecs()

// defaultFeatureCodecs assembles the full production codec set, one register call
// per feature family (the former serialize_codecs_*.go init() bodies, now named
// and explicitly ordered).
func defaultFeatureCodecs() featureCodecSet {
	r := featureCodecSet{}
	r.registerSolidCodecs()
	r.registerDressUpCodecs()
	r.registerPatternCodecs()
	r.registerFaceEditCodecs()
	r.registerSurfaceCodecs()
	r.registerSurfaceEditCodecs()
	r.registerDirectModelCodecs()
	r.registerSheetMetalCodecs()
	r.registerCosmeticCodecs()
	return r
}

// register records one kind's codec, panicking on a duplicate or an incomplete pair — a
// programming error caught at construction, not a silent overwrite.
func (r featureCodecSet) register(kind string, c featureCodec) {
	if kind == "" {
		panic("feature: register with empty kind")
	}
	if c.encode == nil || c.decode == nil {
		panic(fmt.Sprintf("feature: codec for kind %q must have both encode and decode", kind))
	}
	if _, dup := r[kind]; dup {
		panic(fmt.Sprintf("feature: duplicate serialization codec for kind %q", kind))
	}
	r[kind] = c
}

// registeredFeatureKinds returns the kinds with a codec, for the enumerating round-trip test.
func registeredFeatureKinds() []string {
	out := make([]string, 0, len(featureCodecs))
	for k := range featureCodecs {
		out = append(out, k)
	}
	return out
}
