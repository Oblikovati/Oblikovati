// SPDX-License-Identifier: GPL-2.0-only

package feature

// Codec registrations for the sheet-metal family (M13): the wall/flange/hem/bend builders, the
// cut/rip/punch/lip modifiers, and the unfold/refold flat-pattern operations. Encode and decode are
// paired so they cannot drift (#1416); the closures call the serializeX/restoreX helpers in this
// family's serialize_sheet_metal_*.go files.

// registerSheetMetalCodecs contributes this family's codecs to the default set (#1617);
// formerly an init() registration.
func (r featureCodecSet) registerSheetMetalCodecs() {
	r.register("sheet-metal-face", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			sm, err := serializeSheetMetalFace(f.(*SheetMetalFaceFeature).def, sk)
			fd.SheetMetalFace = sm
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalFace(rc.fs, fd.SheetMetalFace, rc.sk)
		},
	})
	r.register("sheet-metal-flange", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.SheetMetalFlange = serializeSheetMetalFlange(f.(*SheetMetalFlangeFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalFlange(rc.fs, fd.SheetMetalFlange)
		},
	})
	r.register("sheet-metal-hem", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.SheetMetalHem = serializeSheetMetalHem(f.(*SheetMetalHemFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalHem(rc.fs, fd.SheetMetalHem)
		},
	})
	r.register("sheet-metal-bend", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smb, err := serializeSheetMetalBend(f.(*SheetMetalBendFeature).def, sk)
			fd.SheetMetalBend = smb
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalBend(rc.fs, fd.SheetMetalBend, rc.sk)
		},
	})
	r.register("sheet-metal-cosmetic-bend", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smcb, err := serializeSheetMetalCosmeticBend(f.(*SheetMetalCosmeticBendFeature).def, sk)
			fd.SheetMetalCosmeticBend = smcb
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalCosmeticBend(rc.fs, fd.SheetMetalCosmeticBend, rc.sk)
		},
	})
	r.register("sheet-metal-rip", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smr, err := serializeSheetMetalRip(f.(*SheetMetalRipFeature).def, sk)
			fd.SheetMetalRip = smr
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalRip(rc.fs, fd.SheetMetalRip, rc.sk)
		},
	})
	r.register("sheet-metal-punch", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smp, err := serializeSheetMetalPunch(f.(*SheetMetalPunchFeature).def, sk)
			fd.SheetMetalPunch = smp
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalPunch(rc.fs, fd.SheetMetalPunch, rc.sk)
		},
	})
	r.register("sheet-metal-lip", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.SheetMetalLip = serializeSheetMetalLip(f.(*SheetMetalLipFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalLip(rc.fs, fd.SheetMetalLip)
		},
	})
	r.register("sheet-metal-fold", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smf, err := serializeSheetMetalFold(f.(*SheetMetalFoldFeature).def, sk)
			fd.SheetMetalFold = smf
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalFold(rc.fs, fd.SheetMetalFold, rc.sk)
		},
	})
	r.register("sheet-metal-corner", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.SheetMetalCorner = serializeSheetMetalCorner(f.(*SheetMetalCornerFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalCorner(rc.fs, fd.SheetMetalCorner)
		},
	})
	r.register("sheet-metal-contour-flange", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smcf, err := serializeSheetMetalContourFlange(f.(*SheetMetalContourFlangeFeature).def, sk)
			fd.SheetMetalContourFlange = smcf
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalContourFlange(rc.fs, fd.SheetMetalContourFlange, rc.sk)
		},
	})
	r.register("sheet-metal-lofted-flange", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smlf, err := serializeSheetMetalLoftedFlange(f.(*SheetMetalLoftedFlangeFeature).def, sk)
			fd.SheetMetalLoftedFlange = smlf
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalLoftedFlange(rc.fs, fd.SheetMetalLoftedFlange, rc.sk)
		},
	})
	r.register("sheet-metal-contour-roll", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smcr, err := serializeSheetMetalContourRoll(f.(*SheetMetalContourRollFeature).def, sk)
			fd.SheetMetalContourRoll = smcr
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalContourRoll(rc.fs, fd.SheetMetalContourRoll, rc.sk)
		},
	})
	r.register("sheet-metal-corner-seam", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.SheetMetalCornerSeam = serializeSheetMetalCornerSeam(f.(*SheetMetalCornerSeamFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalCornerSeam(rc.fs, fd.SheetMetalCornerSeam)
		},
	})
	r.register("sheet-metal-cut", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			smc, err := serializeSheetMetalCut(f.(*SheetMetalCutFeature).def, sk)
			fd.SheetMetalCut = smc
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalCut(rc.fs, fd.SheetMetalCut, rc.sk)
		},
	})
	r.register("sheet-metal-unfold", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.SheetMetalUnfold = &SheetMetalUnfoldData{Bends: serializeBendTransforms(f.(*SheetMetalUnfoldFeature).def.Bends)}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalUnfold(rc.fs, fd.SheetMetalUnfold)
		},
	})
	r.register("sheet-metal-refold", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.SheetMetalRefold = &SheetMetalRefoldData{Bends: serializeBendTransforms(f.(*SheetMetalRefoldFeature).def.Bends)}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSheetMetalRefold(rc.fs, fd.SheetMetalRefold)
		},
	})
}
