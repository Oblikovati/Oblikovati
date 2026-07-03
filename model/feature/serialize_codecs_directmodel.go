// SPDX-License-Identifier: GPL-2.0-only

package feature

// Codec registrations for the direct-geometry family (mesh / freeform / alias-freeform / hull /
// core-cavity). Encode and decode are paired so they cannot drift (#1416); the closures call the
// serializeX/restoreX helpers in serialize_direct_model.go. Before #1617 these kinds had no codec,
// so a part containing any of them refused to marshal.

// registerDirectModelCodecs contributes this family's codecs to the default set (#1617).
func (r featureCodecSet) registerDirectModelCodecs() {
	r.register("mesh", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Mesh = serializeMesh(f.(*MeshFeature).Geometry())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreMesh(rc.fs, fd.Mesh)
		},
	})
	r.register("freeform", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Freeform = serializeFreeform(f.(*FreeformFeature).FreeformBody())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreFreeform(rc.fs, fd.Freeform)
		},
	})
	r.register("alias-freeform", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Freeform = serializeFreeform(f.(*AliasFreeformFeature).FreeformBody())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreAliasFreeform(rc.fs, fd.Freeform)
		},
	})
	r.register("hull", featureCodec{
		encode: func(fd *FeatureData, _ Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Hull = &HullData{}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreHull(rc.fs, fd.Hull)
		},
	})
	r.register("core-cavity", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.CoreCavity = serializeCoreCavity(f.(*CoreCavityFeature).Definition())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreCoreCavity(rc.fs, fd.CoreCavity)
		},
	})
}
