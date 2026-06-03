// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// YAML codecs for the cosmetic/reference features. They carry a payload (image /
// label / attributes / spec) and re-bind their target faces by reference key on the
// next recompute; the running geometry is untouched.

// DecalData is a decal: the target face key and the image resource.
type DecalData struct {
	Face  string `yaml:"face"`
	Image string `yaml:"image"`
}

// ReferenceData is a frozen reference: a label and the referenced source key.
type ReferenceData struct {
	Label  string `yaml:"label"`
	Source string `yaml:"source"`
}

// ClientData is an add-in-owned feature: the add-in id and its attribute payload.
type ClientData struct {
	AddIn      string            `yaml:"addIn"`
	Attributes map[string]string `yaml:"attributes,omitempty"`
}

// MarkData is a manufacturing mark: the marked faces and the mark text.
type MarkData struct {
	Faces []string `yaml:"faces"`
	Text  string   `yaml:"text"`
}

// FinishData is a surface finish: the finished faces and the finish spec.
type FinishData struct {
	Faces []string `yaml:"faces"`
	Spec  string   `yaml:"spec"`
}

// restoreCosmetic rebuilds a cosmetic/reference feature from its payload, erroring on a
// missing payload (no silent loss).
func restoreCosmetic(fs *PartFeatures, fd FeatureData) (*PartFeature, error) {
	c := NewCosmeticFeatures(fs)
	switch fd.Kind {
	case "decal":
		return buildDecal(c, fd.Decal)
	case "reference":
		return buildReference(c, fd.Reference)
	case "client":
		return buildClient(c, fd.Client)
	case "mark":
		return buildMark(c, fd.Mark)
	case "finish":
		return buildFinish(c, fd.Finish)
	default:
		return nil, fmt.Errorf("unknown cosmetic feature kind %q", fd.Kind)
	}
}

func buildDecal(c *CosmeticFeatures, d *DecalData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("decal feature is missing its payload")
	}
	key, err := decodeKey(d.Face)
	if err != nil {
		return nil, err
	}
	c.AddDecal(key, d.Image)
	return lastFeature(c.engine), nil
}

func buildReference(c *CosmeticFeatures, d *ReferenceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("reference feature is missing its payload")
	}
	key, err := decodeKey(d.Source)
	if err != nil {
		return nil, err
	}
	c.AddReference(d.Label, key)
	return lastFeature(c.engine), nil
}

func buildClient(c *CosmeticFeatures, d *ClientData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("client feature is missing its payload")
	}
	c.AddClient(d.AddIn, d.Attributes)
	return lastFeature(c.engine), nil
}

func buildMark(c *CosmeticFeatures, d *MarkData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("mark feature is missing its payload")
	}
	keys, err := decodeKeys(d.Faces)
	if err != nil {
		return nil, err
	}
	c.AddMark(keys, d.Text)
	return lastFeature(c.engine), nil
}

func buildFinish(c *CosmeticFeatures, d *FinishData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("finish feature is missing its payload")
	}
	keys, err := decodeKeys(d.Faces)
	if err != nil {
		return nil, err
	}
	c.AddFinish(keys, d.Spec)
	return lastFeature(c.engine), nil
}
