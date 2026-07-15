// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import "strings"

// classifyFeature maps a feature-definition node to its kind, and reports whether the node is a
// feature the tree should keep (keep=false for the folders, planes, lights, materials and dimension
// sub-nodes that share the same name-tag grammar). The MFC class name is authoritative and
// locale-independent; the localized display name is a fallback only when the class was registered
// earlier and is not adjacent to this node (class == "").
func classifyFeature(class, name string) (FeatureKind, bool) {
	if class != "" {
		if k, known := classKind[class]; known {
			return k, k != KindUnknown // a known non-feature class (folder/plane/…) maps to KindUnknown → drop
		}
	}
	if isDimensionNode(name) {
		return KindUnknown, false
	}
	if k := kindByName(name); k != KindUnknown {
		return k, true
	}
	return KindUnknown, false
}

// classKind maps an MFC feature class to its kind. Classes that are structural (folders, reference
// planes, lights, materials) map to KindUnknown so classifyFeature drops them.
var classKind = map[string]FeatureKind{
	"moProfileFeature_c":       KindSketch,
	"moOriginProfileFeature_c": KindUnknown, // the origin, not a user feature
	"moExtrusion_c":            KindExtrude,
	"moBoss_c":                 KindExtrude,
	"moCut_c":                  KindCut,
	"moRevolution_c":           KindRevolve,
	"moRevolve_c":              KindRevolve,
	"moRevBoss_c":              KindRevolve,
	"moRevCut_c":               KindRevolveCut,
	"Fillet_c":                 KindFillet,
	"moFillet_c":               KindFillet,
	"moConstRadiusFillet_c":    KindFillet,
	"Chamfer_c":                KindChamfer,
	"moChamfer_c":              KindChamfer,
	"moDraft_c":                KindDraft,
	"moMirrorPattern_c":        KindMirror,
	"moMirrorSolid_c":          KindMirror,
	"moCirPattern_c":           KindCircularPattern,
	"moLocalCirPattern_c":      KindCircularPattern,
	"moLinearPattern_c":        KindLinearPattern,
	"moLocalLinearPattern_c":   KindLinearPattern,
	"moSimpleHole_c":           KindHole,
	"moWizardHole_c":           KindHole,
	// Structural nodes that share the name-tag grammar but are not features:
	"moDetailCabinet_c":     KindUnknown,
	"moCommentsFolder_c":    KindUnknown,
	"moDocsFolder_c":        KindUnknown,
	"moSurfaceBodyFolder_c": KindUnknown,
	"moSolidBodyFolder_c":   KindUnknown,
	"moMaterialFolder_c":    KindUnknown,
	"moEnvFolder_c":         KindUnknown,
	"moAmbientLight_c":      KindUnknown,
	"moDirectionLight_c":    KindUnknown,
	"moRefPlane_c":          KindUnknown,
	"moLengthParameter_c":   KindUnknown,
	"moAngleParameter_c":    KindUnknown,
}

// namePrefixKind maps a localized feature-name prefix to its kind, for re-used classes whose
// registration is not adjacent. The corpus is Italian; English prefixes are included so a
// mixed-locale part still classifies. Cut is checked before extrude because the Italian cut name
// ("Taglio-Estrusione") contains the extrude name.
var namePrefixKind = []struct {
	prefix string
	kind   FeatureKind
}{
	{"Schizzo", KindSketch},
	{"Sketch", KindSketch},
	{"Taglio", KindCut},
	{"Cut", KindCut},
	{"Estrusione", KindExtrude},
	{"Extrude", KindExtrude},
	{"Boss", KindExtrude},
	{"Rivoluzione", KindRevolve},
	{"Revolve", KindRevolve},
	{"Revolution", KindRevolve},
	{"Raccordo", KindFillet},
	{"Fillet", KindFillet},
	{"Smusso", KindChamfer},
	{"Chamfer", KindChamfer},
	{"Sformo", KindDraft},
	{"Draft", KindDraft},
	{"Specchia", KindMirror},
	{"Mirror", KindMirror},
	{"Foro", KindHole},
	{"Hole", KindHole},
}

// kindByName classifies a feature by its localized display-name prefix.
func kindByName(name string) FeatureKind {
	for _, m := range namePrefixKind {
		if strings.HasPrefix(name, m.prefix) {
			return m.kind
		}
	}
	return KindUnknown
}

// isDimensionNode reports whether name is a dimension sub-node ("D1", "D2", …) rather than a feature.
func isDimensionNode(name string) bool {
	if len(name) < 2 || name[0] != 'D' {
		return false
	}
	for _, r := range name[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
