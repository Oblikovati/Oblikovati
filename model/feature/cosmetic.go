// SPDX-License-Identifier: GPL-2.0-only

package feature

// Cosmetic and reference features annotate or reference the model rather than cutting
// material: a decal image on a face, a frozen reference body, an add-in-owned client
// feature, a manufacturing mark, and a surface finish. Their recompute passes the
// running body state through unchanged — they carry a payload (image / label /
// attributes / spec) that downstream consumers (drawings, manufacturing) read. They
// exist for full API & persistence parity with the Inventor feature set.

// DecalFeature places an image on a target face (Inventor DecalFeature).
type DecalDefinition struct {
	FaceKey []byte
	Image   string // an image resource id / path
}

// DecalFeature is the realized decal.
type DecalFeature struct{ def *DecalDefinition }

func (d *DecalFeature) Definition() *DecalDefinition { return d.def }
func (d *DecalFeature) Kind() string                 { return "decal" }
func (d *DecalFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: in.Bodies}, nil
}

// ReferenceDefinition is a frozen reference to geometry produced elsewhere, kept for
// downstream reference without participating in the solid (Inventor ReferenceFeature).
type ReferenceDefinition struct {
	Label     string
	SourceKey []byte
}

// ReferenceFeature is the realized reference.
type ReferenceFeature struct{ def *ReferenceDefinition }

func (r *ReferenceFeature) Definition() *ReferenceDefinition { return r.def }
func (r *ReferenceFeature) Kind() string                     { return "reference" }
func (r *ReferenceFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: in.Bodies}, nil
}

// ClientDefinition is an add-in-owned feature: the owning add-in's id plus an opaque
// typed attribute payload the add-in interprets (Inventor ClientFeature).
type ClientDefinition struct {
	AddInID    string
	Attributes map[string]string
}

// ClientFeature is the realized client feature.
type ClientFeature struct{ def *ClientDefinition }

func (c *ClientFeature) Definition() *ClientDefinition { return c.def }
func (c *ClientFeature) Kind() string                  { return "client" }
func (c *ClientFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: in.Bodies}, nil
}

// MarkDefinition is a manufacturing mark (laser/etch) on faces with text
// (Inventor MarkFeature).
type MarkDefinition struct {
	FaceKeys [][]byte
	Text     string
}

// MarkFeature is the realized mark.
type MarkFeature struct{ def *MarkDefinition }

func (m *MarkFeature) Definition() *MarkDefinition { return m.def }
func (m *MarkFeature) Kind() string                { return "mark" }
func (m *MarkFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: in.Bodies}, nil
}

// FinishDefinition is a surface-finish specification applied to faces
// (Inventor FinishFeature).
type FinishDefinition struct {
	FaceKeys [][]byte
	Spec     string
}

// FinishFeature is the realized finish.
type FinishFeature struct{ def *FinishDefinition }

func (f *FinishFeature) Definition() *FinishDefinition { return f.def }
func (f *FinishFeature) Kind() string                  { return "finish" }
func (f *FinishFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: in.Bodies}, nil
}

// CosmeticFeatures adds the cosmetic/reference features into the engine.
type CosmeticFeatures struct{ engine *PartFeatures }

// NewCosmeticFeatures binds the collection to an engine.
func NewCosmeticFeatures(engine *PartFeatures) *CosmeticFeatures { return &CosmeticFeatures{engine} }

// AddDecal places image on the target face.
func (c *CosmeticFeatures) AddDecal(faceKey []byte, image string) *DecalFeature {
	f := &DecalFeature{def: &DecalDefinition{FaceKey: faceKey, Image: image}}
	c.engine.Add(f)
	return f
}

// AddReference records a frozen reference labelled label to the source key.
func (c *CosmeticFeatures) AddReference(label string, sourceKey []byte) *ReferenceFeature {
	f := &ReferenceFeature{def: &ReferenceDefinition{Label: label, SourceKey: sourceKey}}
	c.engine.Add(f)
	return f
}

// AddClient records an add-in-owned feature with its attribute payload.
func (c *CosmeticFeatures) AddClient(addInID string, attributes map[string]string) *ClientFeature {
	f := &ClientFeature{def: &ClientDefinition{AddInID: addInID, Attributes: attributes}}
	c.engine.Add(f)
	return f
}

// AddMark applies a manufacturing mark with text to the given faces.
func (c *CosmeticFeatures) AddMark(faceKeys [][]byte, text string) *MarkFeature {
	f := &MarkFeature{def: &MarkDefinition{FaceKeys: faceKeys, Text: text}}
	c.engine.Add(f)
	return f
}

// AddFinish applies a surface-finish spec to the given faces.
func (c *CosmeticFeatures) AddFinish(faceKeys [][]byte, spec string) *FinishFeature {
	f := &FinishFeature{def: &FinishDefinition{FaceKeys: faceKeys, Spec: spec}}
	c.engine.Add(f)
	return f
}
