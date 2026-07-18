// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"regexp"
)

// FeatureKind classifies a node of the live feature tree by what it does to the model. It is derived
// from the MFC class of the feature node (locale-independent), falling back to the localized display
// name only for a re-used class whose registration is not adjacent (see classifyFeature).
type FeatureKind int

const (
	KindUnknown    FeatureKind = iota
	KindSketch                 // a profile sketch (moProfileFeature_c)
	KindExtrude                // extrude that adds material; the first solid feature is the base (new body)
	KindCut                    // extrude that removes material (moCut_c)
	KindRevolve                // revolve that adds material
	KindRevolveCut             // revolve that removes material
	KindFillet
	KindChamfer
	KindDraft
	KindMirror
	KindCircularPattern
	KindLinearPattern
	KindHole
)

func (k FeatureKind) String() string {
	switch k {
	case KindSketch:
		return "sketch"
	case KindExtrude:
		return "extrude"
	case KindCut:
		return "cut"
	case KindRevolve:
		return "revolve"
	case KindRevolveCut:
		return "revolve-cut"
	case KindFillet:
		return "fillet"
	case KindChamfer:
		return "chamfer"
	case KindDraft:
		return "draft"
	case KindMirror:
		return "mirror"
	case KindCircularPattern:
		return "circular-pattern"
	case KindLinearPattern:
		return "linear-pattern"
	case KindHole:
		return "hole"
	default:
		return "unknown"
	}
}

// Solid reports whether the feature changes the solid body (as opposed to a sketch, which only
// defines a profile). A translator that cannot yet build a solid feature must not silently drop it.
func (k FeatureKind) Solid() bool {
	return k != KindSketch && k != KindUnknown
}

// Material reports whether the feature adds or removes a body volume — as opposed to a cosmetic
// feature (fillet, chamfer, draft) that only rounds or tapers an existing body's edges/faces.
// A part with more than one material feature cannot yet be built correctly (its later profiles need
// the per-sketch object-graph split), so counting these gates whether a base feature is safe to
// build; skipping only the cosmetic tail still yields a faithful, un-rounded body.
func (k FeatureKind) Material() bool {
	switch k {
	case KindExtrude, KindCut, KindRevolve, KindRevolveCut,
		KindMirror, KindCircularPattern, KindLinearPattern, KindHole:
		return true
	default:
		return false
	}
}

// MaterialFeatureCount returns how many material (volume-changing) features the tree holds. Zero
// means no feature-tree information was recovered (e.g. a format-B tree stored elsewhere), which the
// caller distinguishes from a genuine single-feature part.
func (d *Document) MaterialFeatureCount() int {
	n := 0
	for _, f := range d.FeatureTree() {
		if f.Kind.Material() {
			n++
		}
	}
	return n
}

// FeatureNode is one node of the live feature tree, in tree (creation) order.
type FeatureNode struct {
	Name string
	Kind FeatureKind
}

// FeatureTree decodes the part's live feature tree from "Contents/Config-0": the ordered list of
// current features (sketches, extrudes, cuts, fillets, …). It reads the feature-definition region —
// NOT the audit/history log at the top of the stream, which lists deleted features and re-used names
// across editing sessions and would replay features that no longer exist. Each definition node is
// `<class-registration> <name-object-tag> ff fe ff <len> <name UTF-16LE>`; the class name gives the
// feature type locale-independently (see classifyFeature). Returns nil when no feature tree is found
// (e.g. a format-B stream that keeps its tree elsewhere), leaving the sketch-only path unaffected.
func (d *Document) FeatureTree() []FeatureNode {
	region, ok := d.featureRegion()
	if !ok {
		return nil // no feature-definition region (e.g. a format-B tree stored elsewhere)
	}
	var out []FeatureNode
	for _, n := range featureNameTags(region) {
		if k, keep := classifyFeature(n.class, n.name); keep {
			out = append(out, FeatureNode{Name: n.name, Kind: k})
		}
	}
	return out
}

// originFeatureClass marks the start of the feature-definition region: the origin feature is the
// first real feature node, after the audit log and the document metadata (fonts, materials, textures).
const originFeatureClass = "moOriginProfileFeature_c"

// featureRegion returns the slice of the feature-definition stream from the origin feature onward.
// The mo-feature graph lives in "Contents/Config-0" (both container formats keep it there when
// present); it is checked before "Contents/Config-0-ResolvedFeatures", which sketchStream prefers
// for sketch geometry but which does not carry the ordered feature tree.
func (d *Document) featureRegion() ([]byte, bool) {
	for _, name := range []string{"Contents/Config-0", "Contents/Config-0-ResolvedFeatures"} {
		b, err := d.Stream(name)
		if err != nil {
			continue
		}
		if at := bytes.Index(b, []byte(originFeatureClass)); at >= 0 {
			return b[at:], true
		}
	}
	return nil, false
}

// formatASketchRegions splits a format-A (CFBF) part's "Contents/Config-0" into one byte region per
// sketch feature, using the live feature tree's sketch-node offsets. Format A repeats no per-feature
// name marker in the sketch stream (only the object-tagged tree names, which sketchRegions cannot
// see), so without this every sketch's points merge into one region and reconstruct as a single wrong
// cross-sketch loop. Each region runs from a sketch node to the next kept feature node, taking in the
// sketch's geometry and dimension sub-nodes. Returns nil for a single-sketch part (the existing
// whole-stream path already handles it) or when no feature tree is present, so those paths are
// untouched. Format B is excluded (it keeps its validated name-marker split).
func (d *Document) formatASketchRegions() [][]byte {
	if d.cf == nil {
		return nil // format A only
	}
	b, err := d.Stream("Contents/Config-0")
	if err != nil {
		return nil
	}
	start := bytes.Index(b, []byte(originFeatureClass))
	if start < 0 {
		return nil
	}
	return splitByFeatureTree(b[start:])
}

// splitByFeatureTree cuts a feature-definition region into one slice per sketch feature, each running
// from its sketch node to the next kept feature node. Returns nil for a region with fewer than two
// sketches (the whole-stream path already handles the single-sketch case).
func splitByFeatureTree(region []byte) [][]byte {
	var bounds, sketchOffs []int
	for _, n := range featureNameTags(region) {
		k, keep := classifyFeature(n.class, n.name)
		if !keep {
			continue
		}
		bounds = append(bounds, n.off)
		if k == KindSketch {
			sketchOffs = append(sketchOffs, n.off)
		}
	}
	if len(sketchOffs) < 2 {
		return nil
	}
	var regions [][]byte
	for _, so := range sketchOffs {
		regions = append(regions, region[so:nextBound(bounds, so, len(region))])
	}
	return regions
}

// nextBound returns the smallest boundary offset greater than off, or end when off is the last
// feature. bounds is in ascending stream order (feature nodes are scanned in order).
func nextBound(bounds []int, off, end int) int {
	for _, b := range bounds {
		if b > off {
			return b
		}
	}
	return end
}

// cstringMarker precedes a UTF-16 CString's length+bytes in the MFC CArchive. A live feature name is
// an object-tagged CString: `<tag:u16 with 0x8000 set> ff fe ff <len:u8> <name UTF-16LE>`. The
// object tag's high byte (the byte just before the marker) carries the 0x80 bit — and rises past 0x80
// on large parts whose object index exceeds 255 (e.g. pistone's 0x818b). Requiring that top bit
// distinguishes a definition name from an audit-log name, whose length prefix leaves a 0x00 there.
var cstringMarker = []byte{0xff, 0xfe, 0xff}

// classRegistration matches an MFC class name (…_c) so the type of a freshly-registered feature node
// can be read from the bytes immediately preceding its name tag.
var classRegistration = regexp.MustCompile(`[A-Za-z][A-Za-z0-9]{2,40}_c`)

// namedNode is a decoded name tag with the class name that immediately precedes it (empty when the
// class was registered earlier and only referenced here) and the offset of its name tag in the region.
type namedNode struct {
	class string
	name  string
	off   int
}

// featureNameTags scans a feature-definition region for every live feature name, in stream (tree)
// order, pairing each with its adjacent class registration when present.
func featureNameTags(region []byte) []namedNode {
	var out []namedNode
	for i := 0; ; {
		m := bytes.Index(region[i:], cstringMarker)
		if m < 0 {
			break
		}
		at := i + m
		i = at + 1
		if at < 1 || region[at-1]&0x80 == 0 { // not an object-tagged (definition) name
			continue
		}
		name, ok := readNameTag(region, at+len(cstringMarker))
		if !ok {
			continue
		}
		out = append(out, namedNode{class: classBefore(region, at-2), name: name, off: at})
	}
	return out
}

// readNameTag reads the CString name that follows a name tag: a one-byte character count then that
// many UTF-16LE characters. It rejects a false tag match whose "name" is not a plausible identifier.
func readNameTag(region []byte, off int) (string, bool) {
	if off >= len(region) {
		return "", false
	}
	n := int(region[off])
	if n == 0 || off+1+2*n > len(region) {
		return "", false
	}
	name := decodeUTF16(region[off+1 : off+1+2*n])
	if len([]rune(name)) != n || !plausibleName(name) {
		return "", false
	}
	return name, true
}

// classBefore returns the MFC class name whose bytes end at handleEnd (the byte just before the name
// tag), or "" when no class registration abuts the tag (a re-used class referenced by object tag).
// The class name must abut the handle so a stale registration from an earlier node is not picked up.
func classBefore(region []byte, handleEnd int) string {
	lo := handleEnd - 44
	if lo < 0 {
		lo = 0
	}
	if lo >= handleEnd {
		return ""
	}
	locs := classRegistration.FindAllIndex(region[lo:handleEnd], -1)
	if len(locs) == 0 {
		return ""
	}
	last := locs[len(locs)-1]
	if handleEnd-(lo+last[1]) > 2 { // the class name must end within the handle byte(s)
		return ""
	}
	return string(region[lo+last[0] : lo+last[1]])
}

// plausibleName reports whether s reads as a feature/identifier name (printable, no control bytes),
// filtering false `80 ff fe ff` matches inside binary payloads.
func plausibleName(s string) bool {
	for _, r := range s {
		if r < 0x20 || r > 0x2fff {
			return false
		}
	}
	return true
}

// decodeUTF16 decodes little-endian UTF-16 bytes to a string (length already validated by the caller).
func decodeUTF16(b []byte) string {
	runes := make([]rune, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		runes = append(runes, rune(uint16(b[i])|uint16(b[i+1])<<8))
	}
	return string(runes)
}
