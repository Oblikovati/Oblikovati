// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Codec registrations for the dress-up family (fillets, chamfer, shell, draft, lip, and the
// simplify/unwrap clean-ups). Edge/face inputs persist as reference keys that re-bind to the
// regenerated topology on the next recompute. Encode and decode are paired so they cannot drift (#1416).

func init() {
	registerFeatureCodec("fillet", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			ff := f.(*FilletFeature)
			if len(ff.def.EdgeSets) > 0 {
				fd.Fillet = &EdgeDressData{Sets: serializeFilletSets(ff.def.EdgeSets), CornerType: int32(ff.def.CornerType), CrossSection: crossSectionWire(ff.def.CrossSection), Rho: ff.def.Rho}
				return nil
			}
			fd.Fillet = &EdgeDressData{Edges: encodeKeys(ff.def.EdgeKeys), Value: evalFloat(ff.def.Radius), CornerType: int32(ff.def.CornerType), CrossSection: crossSectionWire(ff.def.CrossSection), Rho: ff.def.Rho, GeomEdges: encodeGeomEdges(ff.def.GeomEdges), EdgeAnchors: encodeEdgeAnchors(ff.def.EdgeAnchors)}
			return nil
		},
		decode: decodeFillet,
	})
	registerFeatureCodec("chamfer", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			cf := f.(*ChamferFeature)
			flat := cf.def.FlatCorners
			fd.Chamfer = &EdgeDressData{
				Edges: encodeKeys(cf.def.EdgeKeys), Value: evalFloat(cf.def.Distance), FlatCorners: &flat,
				ChamferType: int32(cf.def.Type), Value2: evalFloat(cf.def.Distance2), Angle: evalFloat(cf.def.Angle),
				GeomEdges: encodeGeomEdges(cf.def.GeomEdges), EdgeAnchors: encodeEdgeAnchors(cf.def.EdgeAnchors),
			}
			return nil
		},
		decode: decodeChamfer,
	})
	registerFeatureCodec("face-fillet", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			ff := f.(*FaceFilletFeature)
			fd.FaceFillet = &FaceFilletData{FacesA: encodeKeys(ff.def.FaceKeysA), FacesB: encodeKeys(ff.def.FaceKeysB), Value: evalFloat(ff.def.Radius)}
			return nil
		},
		decode: decodeFaceFillet,
	})
	registerFeatureCodec("full-round-fillet", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			fr := f.(*FullRoundFilletFeature)
			fd.FullRound = &FullRoundData{Side1: encodeKeys(fr.def.Side1Keys), Center: encodeKeys(fr.def.CenterKeys), Side2: encodeKeys(fr.def.Side2Keys)}
			return nil
		},
		decode: decodeFullRound,
	})
	registerFeatureCodec("rule-fillet", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			rf := f.(*RuleFilletFeature)
			fd.RuleFillet = &RuleFilletData{Rule: rf.def.Rule.String(), Value: evalFloat(rf.def.Radius)}
			return nil
		},
		decode: decodeRuleFillet,
	})
	registerFeatureCodec("shell", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			sf := f.(*ShellFeature)
			fd.Shell = &FaceDressData{Faces: encodeKeys(sf.def.RemovedFaceKeys), Value: evalFloat(sf.def.Thickness), GeomFaces: encodeGeomFaces(sf.def.GeomFaces)}
			return nil
		},
		decode: decodeShell,
	})
	registerFeatureCodec("draft", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			df := f.(*FaceDraftFeature)
			p := df.def.PullDir
			fd.Draft = &FaceDressData{Faces: encodeKeys(df.def.FaceKeys), Value: evalFloat(df.def.Angle), Pull: []float64{p.X, p.Y, p.Z}, GeomFaces: encodeGeomFaces(df.def.GeomFaces)}
			return nil
		},
		decode: decodeDraft,
	})
	registerFeatureCodec("lip", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			lf := f.(*LipFeature)
			fd.Lip = &LipData{Edges: encodeKeys(lf.def.EdgeKeys), Width: evalFloat(lf.def.Width), Height: evalFloat(lf.def.Height), Groove: lf.def.Groove}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreLip(rc.fs, fd.Lip)
		},
	})
	registerFeatureCodec("simplify", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			sf := f.(*SimplifyFeature)
			fd.Simplify = &SimplifyData{RemoveFaces: encodeKeys(sf.def.RemoveFaceKeys), FillVoids: sf.def.FillVoids}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreSimplify(rc.fs, fd.Simplify)
		},
	})
	registerFeatureCodec("unwrap", featureCodec{
		encode: func(fd *FeatureData, f Feature, _ SketchIndexer, _ map[ID]int) error {
			uf := f.(*UnwrapFeature)
			fd.Unwrap = &UnwrapData{Face: encodeKey(uf.def.FaceKey), FaceAnchors: encodeFaceAnchors(uf.def.FaceAnchors)}
			return nil
		},
		decode: func(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
			return restoreUnwrap(rc.fs, fd.Unwrap)
		},
	})
}

// decodeFillet rebuilds a fillet from either the per-edge-set form or the flat edge-key list, restoring
// the corner/cross-section options (an absent corner type is the historical miter default).
func decodeFillet(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	du := NewDressUpFeatures(rc.fs)
	corner := types.FilletCornerType(fd.Fillet.cornerTypeOrZero())
	if corner == 0 {
		corner = types.FilletCornerMiter // absent / older recipe ⇒ the miter default
	}
	cross := fd.Fillet.crossSectionOrArc()
	rho := fd.Fillet.rhoOrZero()
	if fd.Fillet != nil && len(fd.Fillet.Sets) > 0 {
		sets, err := restoreFilletSets(fd.Fillet.Sets)
		if err != nil {
			return nil, err
		}
		return du.addFillet(&FilletDefinition{EdgeSets: sets, CornerType: corner, CrossSection: cross, Rho: rho}), nil
	}
	d, err := requireEdgeDress(fd.Fillet, "fillet")
	if err != nil {
		return nil, err
	}
	return du.addFillet(&FilletDefinition{
		EdgeKeys: d.keys, GeomEdges: d.geom, EdgeAnchors: d.anchors, Radius: constFloat(d.value), CornerType: corner,
		CrossSection: cross, Rho: rho,
	}), nil
}

// decodeChamfer rebuilds a chamfer, restoring the stored flat-corner flag for EVERY mode (the
// asymmetric builders default it to true, but a recipe carries the saved value).
func decodeChamfer(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	d, err := requireEdgeDress(fd.Chamfer, "chamfer")
	if err != nil {
		return nil, err
	}
	def := &ChamferDefinition{
		EdgeKeys: d.keys, GeomEdges: d.geom, EdgeAnchors: d.anchors, Distance: constFloat(d.value),
		Type: types.ChamferType(fd.Chamfer.ChamferType), FlatCorners: chamferFlatCornersOr(fd.Chamfer.FlatCorners),
	}
	switch def.Type {
	case types.ChamferTwoDistances:
		def.Distance2 = constFloat(fd.Chamfer.Value2)
	case types.ChamferDistanceAndAngle:
		def.Angle = constFloat(fd.Chamfer.Angle)
	}
	return NewDressUpFeatures(rc.fs).addChamfer(def), nil
}

// decodeFaceFillet rebuilds a two-face-set variable fillet, re-binding both face sets by key.
func decodeFaceFillet(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	if fd.FaceFillet == nil {
		return nil, fmt.Errorf("face-fillet feature is missing its payload")
	}
	a, err := decodeKeys(fd.FaceFillet.FacesA)
	if err != nil {
		return nil, err
	}
	b, err := decodeKeys(fd.FaceFillet.FacesB)
	if err != nil {
		return nil, err
	}
	return NewDressUpFeatures(rc.fs).AddFaceFillet(a, b, constFloat(fd.FaceFillet.Value)), nil
}

// decodeFullRound rebuilds a full-round fillet from its three face-set keys.
func decodeFullRound(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	if fd.FullRound == nil {
		return nil, fmt.Errorf("full-round-fillet feature is missing its payload")
	}
	s1, err := decodeKeys(fd.FullRound.Side1)
	if err != nil {
		return nil, err
	}
	ctr, err := decodeKeys(fd.FullRound.Center)
	if err != nil {
		return nil, err
	}
	s2, err := decodeKeys(fd.FullRound.Side2)
	if err != nil {
		return nil, err
	}
	return NewDressUpFeatures(rc.fs).AddFullRoundFillet(s1, ctr, s2), nil
}

// decodeRuleFillet rebuilds a rule-driven fillet from its rule name and radius.
func decodeRuleFillet(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	if fd.RuleFillet == nil {
		return nil, fmt.Errorf("rule-fillet feature is missing its payload")
	}
	rule, ok := ParseRuleFilletRule(fd.RuleFillet.Rule)
	if !ok {
		return nil, fmt.Errorf("rule-fillet: unknown rule %q", fd.RuleFillet.Rule)
	}
	return NewDressUpFeatures(rc.fs).AddRuleFillet(rule, constFloat(fd.RuleFillet.Value)), nil
}

// decodeShell rebuilds a shell, re-binding its removed faces by key.
func decodeShell(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	d, err := requireFaceDress(fd.Shell, "shell")
	if err != nil {
		return nil, err
	}
	return NewDressUpFeatures(rc.fs).addShell(&ShellDefinition{
		RemovedFaceKeys: d.keys, GeomFaces: d.geomFaces, Thickness: constFloat(d.value),
	}), nil
}

// decodeDraft rebuilds a face draft, re-binding its faces by key and its pull direction.
func decodeDraft(rc *restoreContext, fd FeatureData) (*PartFeature, error) {
	d, err := requireFaceDress(fd.Draft, "draft")
	if err != nil {
		return nil, err
	}
	return NewDressUpFeatures(rc.fs).addFaceDraft(&FaceDraftDefinition{
		FaceKeys: d.keys, GeomFaces: d.geomFaces, PullDir: draftPull(fd.Draft.Pull), Angle: constFloat(d.value),
	}), nil
}
