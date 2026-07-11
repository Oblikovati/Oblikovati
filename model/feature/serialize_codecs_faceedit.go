// SPDX-License-Identifier: GPL-2.0-only

package feature

// Codec registrations for the direct face-edit family (split, move-face, face-offset, delete-face,
// replace-face). They share one decode (restoreFaceEdit dispatches on the kind) but each persists a
// different slice of FaceEditData; split and delete-face carry only their face keys (the generic
// faceEditor path). Encode and decode are paired so they cannot drift (#1416).

// decodeFaceEdit rebuilds any face-edit kind, dispatching on the kind string inside restoreFaceEdit.
func decodeFaceEdit(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	return restoreFaceEdit(rc.fs, fd.Kind, fd.FaceEdit)
}

// registerFaceEditCodecs contributes this family's codecs to the default set (#1617);
// formerly an init() registration.
func (r featureCodecSet) registerFaceEditCodecs() {
	r.register("move-face", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.FaceEdit = serializeMoveFace(f.(*MoveFaceFeature))
			return nil
		},
		decode: decodeFaceEdit,
	})
	r.register("face-offset", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fo := f.(*FaceOffsetFeature)
			fd.FaceEdit = &FaceEditData{Faces: encodeKeys(fo.FaceKeys()), Distance: fo.Distance(),
				Approximation: approximationName(fo.Approximation())}
			return nil
		},
		decode: decodeFaceEdit,
	})
	r.register("replace-face", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			rf := f.(*ReplaceFaceFeature)
			fd.FaceEdit = &FaceEditData{Faces: encodeKeys(rf.FaceKeys()), Target: encodeKey(rf.TargetKey()), NewFaces: encodePlanes(rf.TargetPlanes())}
			return nil
		},
		decode: decodeFaceEdit,
	})
	// delete-face carries its face keys plus the inverse-heal flag (#1884); Open is negated so a
	// pre-#1884 recipe (no field) restores as healed.
	r.register("delete-face", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			df := f.(*DeleteFaceFeature)
			fd.FaceEdit = &FaceEditData{Faces: encodeKeys(df.FaceKeys()), Open: !df.Heal()}
			return nil
		},
		decode: decodeFaceEdit,
	})
	// split carries only its face keys, via the generic faceEditor interface.
	r.register("split", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.FaceEdit = &FaceEditData{Faces: encodeKeys(f.(faceEditor).FaceKeys())}
			return nil
		},
		decode: decodeFaceEdit,
	})
}
