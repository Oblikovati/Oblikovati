// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Codec registrations for the solid/modify feature families (sketched solids, booleans, holes, and the
// per-face/whole-body modifiers). Each pairs the kind's encode and decode so they cannot drift (#1416);
// the closures call the serializeX/restoreX helpers that live in this family's serialize_*.go files.

func init() {
	registerFeatureCodec("extrude", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			ed, err := serializeExtrude(f.(*ExtrudeFeature).def, sk)
			fd.Extrude = ed
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return requireExtrude(rc.fs, fd.Extrude, rc.sk, rc.work)
		},
	})
	registerFeatureCodec("revolve", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			rv, err := serializeRevolve(f.(*RevolveFeature).def, sk)
			fd.Revolve = rv
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRevolve(rc.fs, fd.Revolve, rc.sk, rc.work)
		},
	})
	registerFeatureCodec("coil", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			cd, err := serializeCoil(f.(*CoilFeature).def, sk)
			fd.Coil = cd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreCoil(rc.fs, fd.Coil, rc.sk, rc.work)
		},
	})
	registerFeatureCodec("sweep", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			sw, err := serializeSweep(f.(*SweepFeature).def, sk)
			fd.Sweep = sw
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSweep(rc.fs, fd.Sweep, rc.sk)
		},
	})
	registerFeatureCodec("loft", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			lo, err := serializeLoft(f.(*LoftFeature).def, sk)
			fd.Loft = lo
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreLoft(rc.fs, fd.Loft, rc.sk)
		},
	})
	registerFeatureCodec("rib", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			rd, err := serializeRib(f.(*RibFeature).def, sk)
			fd.Rib = rd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRib(rc.fs, fd.Rib, rc.sk)
		},
	})
	registerFeatureCodec("emboss", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			ed, err := serializeEmboss(f.(*EmbossFeature).def, sk)
			fd.Emboss = ed
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreEmboss(rc.fs, fd.Emboss, rc.sk)
		},
	})
	registerFeatureCodec("rest", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			rd, err := serializeRest(f.(*RestFeature).def, sk)
			fd.Rest = rd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreRest(rc.fs, fd.Rest, rc.sk)
		},
	})
	registerFeatureCodec("grill", featureCodec{
		encode: func(fd *FeatureData, f Feature, sk SketchIndexer, _ map[ID]int) error {
			gd, err := serializeGrill(f.(*GrillFeature).def, sk)
			fd.Grill = gd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreGrill(rc.fs, fd.Grill, rc.sk)
		},
	})
	registerFeatureCodec("combine", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			cf := f.(*CombineFeature)
			op, err := operationName(cf.def.Operation)
			if err != nil {
				return err
			}
			fd.Combine = &CombineData{Target: cf.def.TargetIndex, Tool: cf.def.ToolIndex, Operation: op}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreCombine(rc.fs, fd.Combine)
		},
	})
	registerFeatureCodec("splitSolid", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			sd, err := serializeSplitSolid(f.(*SplitSolidFeature).def)
			fd.SplitSolid = sd
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSplitSolid(rc.fs, fd.SplitSolid, rc.work)
		},
	})
	registerFeatureCodec("delete-body", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.DeleteBody = serializeDeleteBody(f.(*DeleteBodyFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreDeleteBody(rc.fs, fd.DeleteBody)
		},
	})
	registerFeatureCodec("hole", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			h, err := serializeHole(f.(*HoleFeature).def)
			fd.Hole = h
			return err
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreHole(rc.fs, fd.Hole)
		},
	})
	registerFeatureCodec("boss", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			b := f.(*BossFeature)
			fd.Boss = &BossData{Face: encodeKey(b.def.PlacementFaceKey), Diameter: evalFloat(b.def.Diameter), Height: evalFloat(b.def.Height)}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreBoss(rc.fs, fd.Boss)
		},
	})
	registerFeatureCodec("thread", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			t := f.(*ThreadFeature)
			fd.Thread = &ThreadData{Face: encodeKey(t.def.FaceKey), Designation: t.def.Designation, Cut: t.def.Cut,
				Class: t.def.Class, Tapered: t.def.Tapered, ModelDiameter: threadModelDiameterName(t.def.ModelDiameter)}
			return nil
		},
		decode: decodeThread,
	})
	registerFeatureCodec("snap-fit", featureCodec{
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
	registerFeatureCodec("thicken", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			t := f.(*ThickenFeature)
			fd.Thicken = &ThickenData{Value: t.Thickness(), Approximation: approximationName(t.Approximation())}
			return nil
		},
		decode: decodeThicken,
	})
	registerFeatureCodec("move", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Move = serializeMove(f.(*MoveFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreMove(rc.fs, fd.Move)
		},
	})
	registerFeatureCodec("directEdit", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.DirectEdit = serializeDirectEdit(f.(*DirectEditFeature).def)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreDirectEdit(rc.fs, fd.DirectEdit)
		},
	})
	registerFeatureCodec("importedBody", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.Import = serializeImportedBody(f.(*ImportedBodyFeature))
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreImportedBody(rc.fs, fd.Import)
		},
	})
	registerFeatureCodec("mesh-solid", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fd.MeshSolid = serializeMeshSolid(f.(*MeshSolidFeature).geom)
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreMeshSolid(rc.fs, fd.MeshSolid)
		},
	})
	registerFeatureCodec("modelTolerance", featureCodec{
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
	return NewDressUpFeatures(rc.fs).AddThreadDef(&ThreadDefinition{FaceKey: key, Designation: fd.Thread.Designation,
		Cut: fd.Thread.Cut, Class: fd.Thread.Class, Tapered: fd.Thread.Tapered, ModelDiameter: md}), nil
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

// decodeThicken rebuilds a thicken, restoring the surface-offset approximation mode.
func decodeThicken(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	if fd.Thicken == nil {
		return nil, fmt.Errorf("thicken feature is missing its payload")
	}
	approx, err := approximationOf(fd.Thicken.Approximation)
	if err != nil {
		return nil, err
	}
	pf := NewModifyFeatures(rc.fs).AddThicken(fd.Thicken.Value)
	pf.Definition().(*ThickenFeature).SetApproximation(approx)
	return pf, nil
}
