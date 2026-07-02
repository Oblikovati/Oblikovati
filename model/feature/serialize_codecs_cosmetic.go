// SPDX-License-Identifier: GPL-2.0-only

package feature

// Codec registrations for the cosmetic (decal/reference/client/mark/finish), derived-component
// (derived assembly/part, shrinkwrap), and part-bend families. The cosmetic kinds share one decode
// (restoreCosmetic dispatches on the kind) but each persists a different payload. Encode and decode are
// paired so they cannot drift (#1416).

// decodeCosmetic rebuilds any cosmetic kind, dispatching on the kind string inside restoreCosmetic.
func decodeCosmetic(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	return restoreCosmetic(rc.fs, fd)
}

// registerCosmeticCodecs contributes this family's codecs to the default set (#1617);
// formerly an init() registration.
func (r featureCodecSet) registerCosmeticCodecs() {
	r.register("decal", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			d := f.(*DecalFeature)
			fd.Decal = &DecalData{Face: encodeKey(d.def.FaceKey), Image: d.def.Image}
			return nil
		},
		decode: decodeCosmetic,
	})
	r.register("reference", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			r := f.(*ReferenceFeature)
			fd.Reference = &ReferenceData{Label: r.def.Label, Source: encodeKey(r.def.SourceKey)}
			return nil
		},
		decode: decodeCosmetic,
	})
	r.register("client", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			c := f.(*ClientFeature)
			fd.Client = &ClientData{AddIn: c.def.AddInID, Attributes: c.def.Attributes}
			return nil
		},
		decode: decodeCosmetic,
	})
	r.register("mark", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			m := f.(*MarkFeature)
			fd.Mark = &MarkData{Faces: encodeKeys(m.def.FaceKeys), Text: m.def.Text}
			return nil
		},
		decode: decodeCosmetic,
	})
	r.register("finish", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fn := f.(*FinishFeature)
			fd.Finish = &FinishData{Faces: encodeKeys(fn.def.FaceKeys), Spec: fn.def.Spec}
			return nil
		},
		decode: decodeCosmetic,
	})
	r.register("derivedAssembly", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.DerivedAssembly = serializeDerivedAssembly(f.(*DerivedAssemblyComponent))
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreDerivedAssembly(rc.fs, fd.DerivedAssembly)
		},
	})
	r.register("derived", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.DerivedPart = serializeDerivedPart(f.(*DerivedPartComponent))
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreDerivedPart(rc.fs, fd.DerivedPart)
		},
	})
	r.register("shrinkwrap", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Shrinkwrap = serializeShrinkwrap(f.(*ShrinkwrapComponent))
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreShrinkwrap(rc.fs, fd.Shrinkwrap)
		},
	})
	r.register(kindBendPart, featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			bd, err := serializeBend(f.(*BendPartFeature).def, sk)
			fd.Bend = bd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreBend(rc.fs, fd.Bend, rc.sk)
		},
	})
}
