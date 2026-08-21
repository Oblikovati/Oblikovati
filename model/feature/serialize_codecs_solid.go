// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Codec registrations for the solid/modify feature families (sketched solids, booleans, holes, and the
// per-face/whole-body modifiers). Each pairs the kind's encode and decode so they cannot drift (#1416);
// the closures call the serializeX/restoreX helpers that live in this family's serialize_*.go files.

// registerSolidCodecs contributes this family's codecs to the default set (#1617);
// formerly an init() registration.
func (r featureCodecSet) registerSolidCodecs() {
	r.register("extrude", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			ed, err := serializeExtrude(f.(*ExtrudeFeature).def, sk)
			fd.Extrude = ed
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return requireExtrude(rc.fs, fd.Extrude, rc.sk, rc.work)
		},
	})
	r.register("revolve", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			rv, err := serializeRevolve(f.(*RevolveFeature).def, sk)
			fd.Revolve = rv
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRevolve(rc.fs, fd.Revolve, rc.sk, rc.work)
		},
	})
	r.register("coil", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			cd, err := serializeCoil(f.(*CoilFeature).def, sk)
			fd.Coil = cd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreCoil(rc.fs, fd.Coil, rc.sk, rc.work)
		},
	})
	r.register("sweep", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			sw, err := serializeSweep(f.(*SweepFeature).def, sk)
			fd.Sweep = sw
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSweep(rc.fs, fd.Sweep, rc.sk)
		},
	})
	r.register("loft", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			lo, err := serializeLoft(f.(*LoftFeature).def, sk)
			fd.Loft = lo
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreLoft(rc.fs, fd.Loft, rc.sk)
		},
	})
	r.register("rib", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			rd, err := serializeRib(f.(*RibFeature).def, sk)
			fd.Rib = rd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRib(rc.fs, fd.Rib, rc.sk)
		},
	})
	r.register("emboss", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			ed, err := serializeEmboss(f.(*EmbossFeature).def, sk)
			fd.Emboss = ed
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreEmboss(rc.fs, fd.Emboss, rc.sk)
		},
	})
	r.register("rest", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			rd, err := serializeRest(f.(*RestFeature).def, sk)
			fd.Rest = rd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRest(rc.fs, fd.Rest, rc.sk)
		},
	})
	r.register("grill", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			gd, err := serializeGrill(f.(*GrillFeature).def, sk)
			fd.Grill = gd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreGrill(rc.fs, fd.Grill, rc.sk)
		},
	})
	r.register("combine", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			cf := f.(*CombineFeature)
			op, err := operationName(cf.def.Operation)
			if err != nil {
				return err
			}
			tool, tools := combineToolData(cf.def.ToolIndices)
			fd.Combine = &CombineData{Target: cf.def.TargetIndex, Tool: tool, Tools: tools,
				Operation: op, KeepTools: cf.def.KeepTools}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreCombine(rc.fs, fd.Combine)
		},
	})
	r.register("splitSolid", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			sd, err := serializeSplitSolid(f.(*SplitSolidFeature).def, sk)
			fd.SplitSolid = sd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSplitSolid(rc.fs, fd.SplitSolid, rc.work, rc.sk)
		},
	})
	r.register("delete-body", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.DeleteBody = serializeDeleteBody(f.(*DeleteBodyFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreDeleteBody(rc.fs, fd.DeleteBody)
		},
	})
	r.register("hole", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			h, err := serializeHole(f.(*HoleFeature).def, sk)
			fd.Hole = h
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreHole(rc.fs, fd.Hole, rc.sk, rc.work)
		},
	})
	r.register("boss", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			b := f.(*BossFeature)
			fd.Boss = &BossData{Face: encodeKey(b.def.PlacementFaceKey), Diameter: evalFloat(b.def.Diameter), Height: evalFloat(b.def.Height), FaceAnchors: encodeFaceAnchors(b.def.FaceAnchors)}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreBoss(rc.fs, fd.Boss)
		},
	})
	r.register("thread", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			t := f.(*ThreadFeature)
			fd.Thread = &ThreadData{Face: encodeKey(t.def.FaceKey), Designation: t.def.Designation, Cut: t.def.Cut,
				Class: t.def.Class, Tapered: t.def.Tapered, ModelDiameter: threadModelDiameterName(t.def.ModelDiameter),
				Offset: evalFloat(t.def.Offset), Length: evalFloat(t.def.Length),
				LeftHanded:  t.def.LeftHanded,
				FaceAnchors: encodeFaceAnchors(t.def.FaceAnchors)}
			return nil
		},
		decode: decodeThread,
	})
	r.register("snap-fit", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			d := f.(*SnapFitFeature).def
			fd.SnapFit = &SnapFitData{
				Length: evalFloat(d.Length), Width: evalFloat(d.Width), Thickness: evalFloat(d.Thickness),
				CatchLength: evalFloat(d.CatchLength), CatchHeight: evalFloat(d.CatchHeight),
			}
			return nil
		},
		decode: decodeSnapFit,
	})
	r.register("thicken", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			t := f.(*ThickenFeature)
			chain, blend := t.ChainBlend()
			fd.Thicken = &ThickenData{
				Value: t.Thickness(), Approximation: approximationName(t.Approximation()),
				Direction: thickenDirectionName(t.Direction()),
				Operation: thickenOperationName(t.Operation(), t.AsSurface()),
				Faces:     encodeKeys(t.FaceKeys()), NoWalls: !t.Walls(),
				AutoChain: chain, AutoBlend: blend,
			}
			return nil
		},
		decode: decodeThicken,
	})
	r.register("move", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Move = serializeMove(f.(*MoveFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreMove(rc.fs, fd.Move)
		},
	})
	r.register("directEdit", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.DirectEdit = serializeDirectEdit(f.(*DirectEditFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreDirectEdit(rc.fs, fd.DirectEdit)
		},
	})
	r.register("importedBody", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Import = serializeImportedBody(f.(*ImportedBodyFeature))
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreImportedBody(rc.fs, fd.Import)
		},
	})
	r.register("mesh-solid", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.MeshSolid = serializeMeshSolid(f.(*MeshSolidFeature).geom)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreMeshSolid(rc.fs, fd.MeshSolid)
		},
	})
	r.register("modelTolerance", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.ModelTolerance = serializeModelTolerance(f.(*ModelToleranceFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreModelTolerance(rc.fs, fd.ModelTolerance)
		},
	})
}

// decodeThread rebuilds a cut/tapped thread, re-binding its placement face by reference key.
func decodeThread(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	if fd.Thread == nil {
		return nil, fmt.Errorf("thread feature is missing its payload")
	}
	key, err := decodeKey(fd.Thread.Face)
	if err != nil {
		return nil, err
	}
	md, err := threadModelDiameterOf(fd.Thread.ModelDiameter)
	if err != nil {
		return nil, err
	}
	anchors, err := decodeFaceAnchors(fd.Thread.FaceAnchors)
	if err != nil {
		return nil, err
	}
	return NewDressUpFeatures(rc.fs).addThreadDef(&ThreadDefinition{FaceKey: key, Designation: fd.Thread.Designation,
		Cut: fd.Thread.Cut, Class: fd.Thread.Class, Tapered: fd.Thread.Tapered, ModelDiameter: md,
		Offset: constFloat(fd.Thread.Offset), Length: constFloat(fd.Thread.Length),
		LeftHanded: fd.Thread.LeftHanded, FaceAnchors: anchors}), nil
}

// decodeSnapFit rebuilds a cantilever snap-fit from its persisted dimensions.
func decodeSnapFit(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	if fd.SnapFit == nil {
		return nil, fmt.Errorf("snap-fit feature is missing its payload")
	}
	d := fd.SnapFit
	return NewPlasticFeatures(rc.fs).AddCantileverSnapFit(
		constFloat(d.Length), constFloat(d.Width), constFloat(d.Thickness),
		constFloat(d.CatchLength), constFloat(d.CatchHeight)), nil
}

// decodeThicken rebuilds a thicken, restoring the #331 approximation mode and #1876 options.
func decodeThicken(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	if fd.Thicken == nil {
		return nil, fmt.Errorf("thicken feature is missing its payload")
	}
	d := fd.Thicken
	approx, err := approximationOf(d.Approximation)
	if err != nil {
		return nil, err
	}
	keys, err := decodeKeys(d.Faces)
	if err != nil {
		return nil, err
	}
	pf := NewModifyFeatures(rc.fs).AddThicken(d.Value)
	tf := pf.Definition().(*ThickenFeature)
	tf.SetApproximation(approx)
	op, asSurface := thickenOperationOf(d.Operation)
	tf.SetThickenOptions(thickenDirectionOf(d.Direction), op, asSurface, keys, !d.NoWalls, d.AutoChain, d.AutoBlend)
	return pf, nil
}
