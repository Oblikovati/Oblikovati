// SPDX-License-Identifier: GPL-2.0-only

package feature

// Codec registrations for the pattern family (rectangular, circular, sketch-driven, mirror). A pattern
// records the earlier features it replicates as program indices (resolved through idx on encode and the
// restored slice on decode). Encode and decode are paired so they cannot drift (#1416).

// registerPatternCodecs contributes this family's codecs to the default set (#1617);
// formerly an init() registration.
func (r featureCodecSet) registerPatternCodecs() {
	r.register("rectangular-pattern", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, idx map[ID]int) error {
			p := f.(*RectangularPatternFeature)
			src, err := sourceIndices(p.def.SourceFeatures, idx)
			if err != nil {
				return err
			}
			fd.RectPattern = &RectPatternData{
				Source: src, CountX: evalInt(p.def.CountX), CountY: evalInt(p.def.CountY),
				StepX: encodeVec3(p.def.StepX), StepY: encodeVec3(p.def.StepY),
				Options:   encodePatternOptions(p.def.Options),
				MidPlaneX: p.def.MidPlaneX, MidPlaneY: p.def.MidPlaneY,
				Suppressed: p.SuppressedIndices(),
			}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRectPattern(rc.fs, fd.RectPattern, rc.restored)
		},
	})
	r.register("circular-pattern", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, idx map[ID]int) error {
			p := f.(*CircularPatternFeature)
			src, err := sourceIndices(p.def.SourceFeatures, idx)
			if err != nil {
				return err
			}
			fd.CircPattern = &CircPatternData{
				Source: src, Count: evalInt(p.def.Count), Angle: evalFloat(p.def.Angle),
				AxisPoint: encodePoint3(p.def.AxisPoint), AxisDir: encodeVec3(p.def.AxisDir),
				Options:    encodePatternOptions(p.def.Options),
				MidPlane:   p.def.MidPlane,
				Suppressed: p.SuppressedIndices(),
			}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreCircPattern(rc.fs, fd.CircPattern, rc.restored)
		},
	})
	r.register("sketch-driven-pattern", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, idx map[ID]int) error {
			p := f.(*SketchDrivenPatternFeature)
			src, err := sourceIndices(p.def.SourceFeatures, idx)
			if err != nil {
				return err
			}
			fd.SketchPattern = &SketchDrivenPatternData{Source: src, Points: encodePoints(callPoints(p.def.Points))}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSketchPattern(rc.fs, fd.SketchPattern, rc.restored)
		},
	})
	r.register("mirror", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, idx map[ID]int) error {
			p := f.(*MirrorFeature)
			src, err := sourceIndices(p.def.SourceFeatures, idx)
			if err != nil {
				return err
			}
			fd.Mirror = &MirrorData{
				Source: src, Plane: encodeKey(p.def.MirrorPlaneKey),
				Origin: encodePoint3(p.def.Origin), Normal: encodeVec3(p.def.Normal),
				OfBody:         p.def.OfBody,
				RemoveOriginal: p.def.RemoveOriginal,
				JoinToOriginal: p.def.JoinToOriginal,
			}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreMirror(rc.fs, fd.Mirror, rc.restored)
		},
	})
}
