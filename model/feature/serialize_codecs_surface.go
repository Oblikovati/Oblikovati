// SPDX-License-Identifier: GPL-2.0-only

package feature

// Codec registrations for the surfacing family (boundary/ruled/network/fill/bridge patches and the
// rebuild/match/extend/untrim/fair/fit surface edits). Encode and decode are paired so they cannot
// drift (#1416); the closures call the serializeX/restoreX helpers in this family's serialize_*.go files.

func init() {
	registerFeatureCodec("boundary-patch", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			bp, err := serializeBoundaryPatch(f.(*BoundaryPatchFeature).def, sk)
			fd.BoundaryPatch = bp
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreBoundaryPatch(rc.fs, fd.BoundaryPatch, rc.sk)
		},
	})
	registerFeatureCodec("ruled-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			rs, err := serializeRuledSurface(f.(*RuledSurfaceFeature).def, sk)
			fd.RuledSurface = rs
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRuledSurface(rc.fs, fd.RuledSurface, rc.sk)
		},
	})
	registerFeatureCodec("rebuild-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Rebuild = serializeRebuild(f.(*RebuildFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRebuild(rc.fs, fd.Rebuild)
		},
	})
	registerFeatureCodec("control-point-edit", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.ControlPointEdit = serializeControlPointEdit(f.(*ControlPointEditFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreControlPointEdit(rc.fs, fd.ControlPointEdit)
		},
	})
	registerFeatureCodec("nurbs-plane", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.NurbsPlane = serializeNurbsPlane(f.(*NurbsPlaneFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreNurbsPlane(rc.fs, fd.NurbsPlane)
		},
	})
	registerFeatureCodec("match-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Match = serializeMatch(f.(*MatchFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreMatch(rc.fs, fd.Match)
		},
	})
	registerFeatureCodec("extend-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.ExtendSurface = serializeExtendSurface(f.(*ExtendSurfaceFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreExtendSurface(rc.fs, fd.ExtendSurface)
		},
	})
	registerFeatureCodec("untrim-surface", featureCodec{
		encode: func(fd *FeatureData, _ Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Untrim = &UntrimData{}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreUntrim(rc.fs, fd.Untrim)
		},
	})
	registerFeatureCodec("fill-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.FillSurface = serializeFillSurface(f.(*FillFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreFillSurface(rc.fs, fd.FillSurface)
		},
	})
	registerFeatureCodec("bridge-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.BridgeSurface = serializeBridgeSurface(f.(*BridgeFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreBridgeSurface(rc.fs, fd.BridgeSurface)
		},
	})
	registerFeatureCodec("network-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.NetworkSurface = serializeNetworkSurface(f.(*NetworkFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreNetworkSurface(rc.fs, fd.NetworkSurface)
		},
	})
	registerFeatureCodec("fair-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.FairSurface = serializeFairSurface(f.(*FairFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreFairSurface(rc.fs, fd.FairSurface)
		},
	})
	registerFeatureCodec("fit-surface", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.FitSurface = serializeFitSurface(f.(*FitFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreFitSurface(rc.fs, fd.FitSurface)
		},
	})
}
