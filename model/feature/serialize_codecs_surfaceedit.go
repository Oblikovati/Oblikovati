// SPDX-License-Identifier: GPL-2.0-only

package feature

// Codec registrations for the surface-editing family (trim / extend / surface-offset / mid-surface /
// stitch / sculpt). Encode and decode are paired so they cannot drift (#1416); the closures call the
// serializeX/restoreX helpers in serialize_surface_edits.go. Before #1617 these kinds had no codec at
// all, so a part containing one refused to marshal — the exact silent-drop class the registry closes.

// registerSurfaceEditCodecs contributes this family's codecs to the default set (#1617).
func (r featureCodecSet) registerSurfaceEditCodecs() {
	r.register("trim", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Trim = serializeTrim(f.(*TrimFeature).Definition())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreTrim(rc.fs, fd.Trim)
		},
	})
	r.register("extend", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Extend = serializeExtend(f.(*ExtendFeature).Definition())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreExtend(rc.fs, fd.Extend)
		},
	})
	r.register("surface-offset", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.SurfaceOffset = serializeSurfaceOffset(f.(*SurfaceOffsetFeature).Definition())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSurfaceOffset(rc.fs, fd.SurfaceOffset)
		},
	})
	r.register("mid-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.MidSurface = serializeMidSurface(f.(*MidSurfaceFeature).Definition())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreMidSurface(rc.fs, fd.MidSurface)
		},
	})
	r.register("stitch", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Stitch = serializeStitch(f.(*StitchFeature).Definition())
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreStitch(rc.fs, fd.Stitch)
		},
	})
	r.register("sculpt", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			s, err := serializeSculpt(f.(*SculptFeature).Definition())
			fd.Sculpt = s
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSculpt(rc.fs, fd.Sculpt)
		},
	})
}
