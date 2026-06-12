// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SplitSolidData is a solid split's recipe: the cutting work plane (by WorkRef), which
// side(s) to keep, and the faces-only flag (the Split Faces mode, #330).
type SplitSolidData struct {
	Plane     string `yaml:"plane"`
	Keep      string `yaml:"keep,omitempty"`
	FacesOnly bool   `yaml:"facesOnly,omitempty"`
}

// splitSideNames map the kept-side enum to/from a stable name.
var splitSideNames = map[SplitSide]string{SplitBoth: "both", SplitPositive: "positive", SplitNegative: "negative"}

func splitSideName(s SplitSide) string {
	if n, ok := splitSideNames[s]; ok {
		return n
	}
	return "both"
}

func parseSplitSide(name string) (SplitSide, error) {
	for s, n := range splitSideNames {
		if n == name {
			return s, nil
		}
	}
	if name == "" {
		return SplitBoth, nil
	}
	return SplitBoth, fmt.Errorf("unknown split side %q", name)
}

func serializeSplitSolid(def *SplitSolidDefinition) (*SplitSolidData, error) {
	if def.Plane == nil {
		return nil, fmt.Errorf("split references no cutting plane")
	}
	return &SplitSolidData{Plane: string(def.Plane.Key()), Keep: splitSideName(def.Keep), FacesOnly: def.FacesOnly}, nil
}

func restoreSplitSolid(fs *PartFeatures, d *SplitSolidData, work *WorkGeometry) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("split feature is missing its payload")
	}
	wp, err := resolvePlaneRef(work, d.Plane)
	if err != nil {
		return nil, err
	}
	if d.FacesOnly {
		return NewModifyFeatures(fs).AddSplitFaces(wp), nil
	}
	keep, err := parseSplitSide(d.Keep)
	if err != nil {
		return nil, err
	}
	return NewModifyFeatures(fs).AddSplitSolid(wp, keep), nil
}
