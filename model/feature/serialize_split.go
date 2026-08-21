// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SplitSolidData is a solid split's recipe: the cutting work plane (by WorkRef), which
// side(s) to keep, and the faces-only flag (the Split Faces mode, #330).
//
// Tool/ToolIndex name a surface tool instead of the plane (#1891); both are absent for the
// work-plane split, so a document written before the tool existed reads back unchanged.
type SplitSolidData struct {
	Plane     string `yaml:"plane"`
	Keep      string `yaml:"keep,omitempty"`
	FacesOnly bool   `yaml:"facesOnly,omitempty"`
	Tool      string `yaml:"tool,omitempty"`
	ToolIndex int    `yaml:"toolIndex,omitempty"`
	// Sketch/ProfileIndex reference the SplitByPath tool's sketch by its index in the part (#2068);
	// both absent for every other tool, so an older document reads back unchanged.
	Sketch       int `yaml:"sketch,omitempty"`
	ProfileIndex int `yaml:"profileIndex,omitempty"`
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

func serializeSplitSolid(def *SplitSolidDefinition, sk SketchIndexer) (*SplitSolidData, error) {
	d := &SplitSolidData{Keep: splitSideName(def.Keep), FacesOnly: def.FacesOnly,
		Tool: SplitToolName(def.Tool), ToolIndex: def.ToolIndex}
	switch def.Tool {
	case SplitByWorkPlane:
		if def.Plane == nil {
			return nil, fmt.Errorf("split references no cutting plane")
		}
		d.Plane = string(def.Plane.Key())
	case SplitByPath:
		idx, ok := sk.IndexOf(def.Sketch)
		if !ok {
			return nil, fmt.Errorf("split: the path tool's sketch is not in the part")
		}
		d.Sketch, d.ProfileIndex = idx, def.ProfileIndex
	}
	return d, nil
}

func restoreSplitSolid(fs *PartFeatures, d *SplitSolidData, work *WorkGeometry, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("split feature is missing its payload")
	}
	tool, ok := ParseSplitTool(d.Tool)
	if !ok {
		return nil, fmt.Errorf("split: unknown tool %q (want workPlane/workSurface/surfaceBody/path)", d.Tool)
	}
	keep, err := parseSplitSide(d.Keep)
	if err != nil {
		return nil, err
	}
	def := &SplitSolidDefinition{Tool: tool, ToolIndex: d.ToolIndex, Keep: keep, FacesOnly: d.FacesOnly}
	switch tool {
	case SplitByWorkPlane:
		if def.Plane, err = resolvePlaneRef(work, d.Plane); err != nil {
			return nil, err
		}
	case SplitByPath:
		skt, ok := sk.At(d.Sketch)
		if !ok {
			return nil, fmt.Errorf("split references sketch index %d, which does not exist", d.Sketch)
		}
		def.Sketch, def.ProfileIndex = skt, d.ProfileIndex
	}
	return NewModifyFeatures(fs).AddSplitByDefinition(def), nil
}
