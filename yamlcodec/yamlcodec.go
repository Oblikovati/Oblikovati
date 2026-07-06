// SPDX-License-Identifier: GPL-2.0-only

// Package yamlcodec is the project's single point of contact with the YAML library
// (gopkg.in/yaml.v3). Per the dependency rule (CLAUDE.md) and ADR-0020, only this
// package imports yaml — everything else marshals through these functions, so a
// future library swap stays local.
//
// It also owns the on-disk .obk document shape: one readable YAML file whose manifest
// fields sit at the top level and whose recipe is a NATIVE nested node (not a quoted
// blob), so a model diffs line-by-line in git. Binary data sections (add-in/attribute
// scratch — the rare non-recipe payload) are base64-encoded, the one concession to
// non-text data (ADR-0020).
package yamlcodec

import (
	"encoding/base64"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Marshal renders v as YAML bytes.
func Marshal(v any) ([]byte, error) { return yaml.Marshal(v) }

// Unmarshal parses YAML bytes into v.
func Unmarshal(data []byte, v any) error { return yaml.Unmarshal(data, v) }

// Document is the decoded form of a .obk file: identity, the recipe section (raw YAML
// bytes, embedded natively on disk for readability), and named binary data sections.
// Model is nil when the document carries no recipe (e.g. a manifest-only stub or a
// pure DataIO container).
type Document struct {
	SchemaVersion int
	DocumentType  uint32
	SubType       string // add-in flavored subtype id (M05-F15)
	DisplayName   string
	Model         []byte
	Data          map[string][]byte
	// Resources is the root resource table (ADR-0031): imported files embedded in the
	// document, keyed by a per-import UUID and referenced from the recipe by that key.
	Resources map[string]Resource
	// Identity is the file identity block (M03-F07, #159); nil for pre-identity files.
	Identity *FileIdentityRecord
	// References are the as-saved file-to-file reference records (M03-F07).
	References []FileReferenceRecord
	// Attachments are the external-file attachment records (M03-F08).
	Attachments []AttachmentRecord
	// Interests are the add-in data registry records (M03-F10).
	Interests []InterestRecord
	// PointClouds are the attached scan records (M17-F06, #645); their bytes are in Resources.
	PointClouds []PointCloudRecord
	// Attributes is the add-in attribute-set block (#155): the encoded AttributeManager
	// (model/attr), nil for a document that carries no attributes.
	Attributes []byte
	// DisplaySettings are the per-document display settings (background/edges/ground/shadows),
	// nil for a document that never customized them (M16-F07 #643).
	DisplaySettings *DisplaySettingsRecord `yaml:"displaySettings,omitempty"`
	// SketchSettings are the per-document sketch-authoring defaults (constraint inference),
	// nil for a document that never customized them (#147).
	SketchSettings *SketchSettingsRecord `yaml:"sketchSettings,omitempty"`
	// BodyNames are the per-body display names keyed by body reference key (#1078), absent for a
	// document where no body was renamed.
	BodyNames map[string]string `yaml:"bodyNames,omitempty"`
	// BodyColorStyles are the per-body color-style names keyed by body reference key (M16-F02
	// #403/#408, S5 #1640), absent for a document where no body was colored.
	BodyColorStyles map[string]string `yaml:"bodyColorStyles,omitempty"`
}

// SketchSettingsRecord is the on-disk per-document sketch settings (#147): the constraint-inference
// toggles and family priority. The priority enum is stored as its frozen integer id.
type SketchSettingsRecord struct {
	InferConstraints     bool  `yaml:"inferConstraints"`
	AutoApplyConstraints bool  `yaml:"autoApplyConstraints"`
	ConstraintPriority   int32 `yaml:"constraintPriority"`
}

// ColorRecord is a color value object on disk: 8-bit rgb, opacity, and the color-source enum.
type ColorRecord struct {
	R       uint8   `yaml:"r,omitempty"`
	G       uint8   `yaml:"g,omitempty"`
	B       uint8   `yaml:"b,omitempty"`
	Opacity float64 `yaml:"opacity,omitempty"`
	Source  int32   `yaml:"source,omitempty"`
}

// GroundPlaneRecord is the on-disk ground-plane block of the display settings.
type GroundPlaneRecord struct {
	Visible                    bool        `yaml:"visible"`
	Color                      ColorRecord `yaml:"color,omitempty"`
	HeightOffset               float64     `yaml:"heightOffset,omitempty"`
	DisplayGridLines           bool        `yaml:"displayGridLines,omitempty"`
	MinorGridLineSpacing       float64     `yaml:"minorGridLineSpacing,omitempty"`
	MinorLinesPerMajorGridLine int         `yaml:"minorLinesPerMajorGridLine,omitempty"`
	Opacity                    float64     `yaml:"opacity,omitempty"`
	Reflectivity               float64     `yaml:"reflectivity,omitempty"`
}

// DisplaySettingsRecord is the on-disk per-document display settings (M16-F07 #643). Enums are
// stored as their frozen integer ids; colors as [ColorRecord]. The application converts to/from
// its model display settings.
type DisplaySettingsRecord struct {
	BackgroundType           int32             `yaml:"backgroundType,omitempty"`
	EdgeColor                ColorRecord       `yaml:"edgeColor,omitempty"`
	DepthDimming             bool              `yaml:"depthDimming,omitempty"`
	DisplaySilhouettes       bool              `yaml:"displaySilhouettes,omitempty"`
	HiddenLineDimmingPercent int               `yaml:"hiddenLineDimmingPercent,omitempty"`
	NewWindowDisplayMode     int32             `yaml:"newWindowDisplayMode,omitempty"`
	DisplayModeSource        int32             `yaml:"displayModeSource,omitempty"`
	NewWindowProjection      int32             `yaml:"newWindowProjection,omitempty"`
	GroundPlane              GroundPlaneRecord `yaml:"groundPlane,omitempty"`
	GroundShadow             int32             `yaml:"groundShadow,omitempty"`
	ShadowDirection          int32             `yaml:"shadowDirection,omitempty"`
	ShowGroundReflections    bool              `yaml:"showGroundReflections,omitempty"`
	ShowObjectShadows        bool              `yaml:"showObjectShadows,omitempty"`
	ShowAmbientShadows       bool              `yaml:"showAmbientShadows,omitempty"`
	TexturesOn               bool              `yaml:"texturesOn,omitempty"`
}

// FileIdentityRecord is the on-disk file identity block: the stable GUID plus
// the revision stamps a referencing file compares against (M03-F07, #159).
type FileIdentityRecord struct {
	InternalName       string `yaml:"internalName,omitempty"`
	RevisionID         string `yaml:"revisionId,omitempty"`
	DatabaseRevisionID string `yaml:"databaseRevisionId,omitempty"`
	SaveCounter        int    `yaml:"saveCounter,omitempty"`
	VersionCreated     string `yaml:"versionCreated,omitempty"`
	VersionSaved       string `yaml:"versionSaved,omitempty"`
	ModelDigest        string `yaml:"modelDigest,omitempty"`
}

// FileReferenceRecord is one as-saved file-to-file reference: the logical
// names, the location class (a wire spelling, readable in the file), and the
// referenced file's identity at save time (M03-F07).
type FileReferenceRecord struct {
	FullFileName           string `yaml:"fullFileName"`
	RelativeFileName       string `yaml:"relativeFileName,omitempty"`
	LibraryName            string `yaml:"libraryName,omitempty"`
	LocationType           string `yaml:"locationType,omitempty"`
	ReferencedInternalName string `yaml:"referencedInternalName,omitempty"`
	SaveCounter            int    `yaml:"saveCounter,omitempty"`
}

// AttachmentRecord is one external-file attachment (M03-F08): a named link to
// a foreign file, with an embedded payload carried base64 (the same concession
// as data sections, ADR-0020).
type AttachmentRecord struct {
	Name              string `yaml:"name"`
	Kind              string `yaml:"kind"`
	FullFileName      string `yaml:"fullFileName,omitempty"`
	ResourceID        string `yaml:"resourceId,omitempty"`
	Payload           string `yaml:"payload,omitempty"` // base64
	LastKnownFileTime string `yaml:"lastKnownFileTime,omitempty"`
	BrowserVisible    bool   `yaml:"browserVisible,omitempty"`
}

// PointCloudRecord is one attached scan's metadata (M17-F06, #645): its name, source path, the
// id of the resource holding its bytes, visibility, display mode, scale, the 16 cloud→model
// transform cells, and the display point budget. The points themselves live once in the resource
// table.
type PointCloudRecord struct {
	Name        string                 `yaml:"name"`
	Source      string                 `yaml:"source,omitempty"`
	ResourceID  string                 `yaml:"resourceId,omitempty"`
	Visible     bool                   `yaml:"visible"`
	DisplayMode string                 `yaml:"displayMode,omitempty"`
	Scale       float64                `yaml:"scale"`
	Transform   [16]float64            `yaml:"transform,flow"`
	MaxPoints   int                    `yaml:"maxPoints,omitempty"`
	Crops       []PointCloudCropRecord `yaml:"crops,omitempty"`
}

// PointCloudCropRecord is one crop volume (#645): a name, the active flag, and the model-space box
// min/max corners.
type PointCloudCropRecord struct {
	Name   string     `yaml:"name"`
	Active bool       `yaml:"active"`
	Min    [3]float64 `yaml:"min,flow"`
	Max    [3]float64 `yaml:"max,flow"`
}

// InterestRecord is one add-in data-registry entry (M03-F10): client X has
// data in / depends on this document, readable without loading the add-in.
type InterestRecord struct {
	ClientID     string `yaml:"clientId"`
	Name         string `yaml:"name"`
	InterestType string `yaml:"interestType,omitempty"`
	DataVersion  int    `yaml:"dataVersion,omitempty"`
	ClientData   string `yaml:"clientData,omitempty"`
}

// onDisk is the YAML projection of a Document: manifest at top level, recipe as a
// native node, data sections base64-encoded. omitempty keeps a minimal file readable.
type onDisk struct {
	SchemaVersion   int                    `yaml:"schemaVersion,omitempty"`
	DocumentType    uint32                 `yaml:"documentType,omitempty"`
	SubType         string                 `yaml:"subType,omitempty"`
	DisplayName     string                 `yaml:"displayName,omitempty"`
	Identity        *FileIdentityRecord    `yaml:"identity,omitempty"`
	References      []FileReferenceRecord  `yaml:"references,omitempty"`
	Attachments     []AttachmentRecord     `yaml:"attachments,omitempty"`
	Interests       []InterestRecord       `yaml:"interests,omitempty"`
	PointClouds     []PointCloudRecord     `yaml:"pointClouds,omitempty"`
	Attributes      []byte                 `yaml:"attributes,omitempty"`
	DisplaySettings *DisplaySettingsRecord `yaml:"displaySettings,omitempty"`
	SketchSettings  *SketchSettingsRecord  `yaml:"sketchSettings,omitempty"`
	BodyNames       map[string]string      `yaml:"bodyNames,omitempty"`
	BodyColorStyles map[string]string      `yaml:"bodyColorStyles,omitempty"`
	Resources       yaml.Node              `yaml:"resources,omitempty"`
	Model           yaml.Node              `yaml:"model,omitempty"`
	Data            map[string]string      `yaml:"data,omitempty"`
}

// MarshalDocument renders d as the on-disk YAML file. The recipe bytes are parsed and
// embedded as a native node so the model is real nested YAML, not an escaped string.
func MarshalDocument(d Document) ([]byte, error) {
	od := onDisk{
		SchemaVersion:   d.SchemaVersion,
		DocumentType:    d.DocumentType,
		SubType:         d.SubType,
		DisplayName:     d.DisplayName,
		Identity:        d.Identity,
		References:      d.References,
		Attachments:     d.Attachments,
		Interests:       d.Interests,
		PointClouds:     d.PointClouds,
		Attributes:      d.Attributes,
		DisplaySettings: d.DisplaySettings,
		SketchSettings:  d.SketchSettings,
		BodyNames:       d.BodyNames,
		BodyColorStyles: d.BodyColorStyles,
	}
	if err := embedNativeNodes(&od, d); err != nil {
		return nil, err
	}
	if len(d.Data) > 0 {
		od.Data = make(map[string]string, len(d.Data))
		for name, raw := range d.Data {
			od.Data[name] = base64.StdEncoding.EncodeToString(raw)
		}
	}
	return yaml.Marshal(od)
}

// embedNativeNodes parses the resource table and the recipe into native YAML
// nodes so both read as real nested YAML on disk, not escaped strings.
func embedNativeNodes(od *onDisk, d Document) error {
	if len(d.Resources) > 0 {
		node, err := resourcesNode(d.Resources)
		if err != nil {
			return err
		}
		od.Resources = *node
	}
	if len(d.Model) > 0 {
		node, err := modelNode(d.Model)
		if err != nil {
			return err
		}
		od.Model = *node
	}
	return nil
}

// UnmarshalDocument decodes a .obk file's bytes. It rejects a legacy ZIP package with
// a clear message (ADR-0020: the format is now YAML) and surfaces base64/YAML errors.
func UnmarshalDocument(raw []byte) (Document, error) {
	if isZip(raw) {
		return Document{}, errors.New("yamlcodec: this looks like a legacy ZIP .obk; the document format is now a YAML text file (ADR-0020) and old ZIP packages are not supported")
	}
	var od onDisk
	if err := yaml.Unmarshal(raw, &od); err != nil {
		return Document{}, fmt.Errorf("yamlcodec: parse document: %w", err)
	}
	d := documentHeader(od)
	resources, err := decodeResources(&od.Resources)
	if err != nil {
		return Document{}, err
	}
	d.Resources = resources
	if od.Model.Kind != 0 {
		b, err := yaml.Marshal(&od.Model)
		if err != nil {
			return Document{}, fmt.Errorf("yamlcodec: extract model: %w", err)
		}
		d.Model = b
	}
	data, err := decodeDataSections(od.Data)
	if err != nil {
		return Document{}, err
	}
	d.Data = data
	return d, nil
}

// documentHeader copies the manifest fields off the on-disk projection.
func documentHeader(od onDisk) Document {
	return Document{
		SchemaVersion:   od.SchemaVersion,
		DocumentType:    od.DocumentType,
		SubType:         od.SubType,
		DisplayName:     od.DisplayName,
		Identity:        od.Identity,
		References:      od.References,
		Attachments:     od.Attachments,
		Interests:       od.Interests,
		PointClouds:     od.PointClouds,
		Attributes:      od.Attributes,
		DisplaySettings: od.DisplaySettings,
		SketchSettings:  od.SketchSettings,
		BodyNames:       od.BodyNames,
		BodyColorStyles: od.BodyColorStyles,
	}
}

// decodeDataSections base64-decodes the on-disk data sections into raw bytes, or nil when
// there are none.
func decodeDataSections(enc map[string]string) (map[string][]byte, error) {
	if len(enc) == 0 {
		return nil, nil
	}
	out := make(map[string][]byte, len(enc))
	for name, s := range enc {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("yamlcodec: data section %q is not valid base64: %w", name, err)
		}
		out[name] = b
	}
	return out, nil
}

// modelNode parses recipe YAML bytes into the mapping node to embed under `model:`.
// yaml.Unmarshal yields a document node wrapping the real content; we splice in that
// content root so the embedded model is not double-wrapped.
func modelNode(modelYAML []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(modelYAML, &doc); err != nil {
		return nil, fmt.Errorf("yamlcodec: embed model: %w", err)
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		return doc.Content[0], nil
	}
	return &doc, nil
}

// isZip reports whether raw begins with the local-file-header magic of a ZIP archive
// ("PK\x03\x04") — i.e. a pre-ADR-0020 package.
func isZip(raw []byte) bool {
	return len(raw) >= 4 && raw[0] == 'P' && raw[1] == 'K' && raw[2] == 0x03 && raw[3] == 0x04
}
