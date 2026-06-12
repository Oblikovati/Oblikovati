// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Interactive sketch-block tools (M06-F07, #622): Create Block gathers a
// selection and moves it into a named definition (an instance replaces it in
// place); Place Block stamps instances of a chosen definition at clicked
// insertion points.

// SketchCreateBlockTool turns the selection into a block definition.
type SketchCreateBlockTool struct {
	sketchSelectTool
	name string
}

// NewSketchCreateBlockTool returns the create-from-selection tool.
func NewSketchCreateBlockTool() *SketchCreateBlockTool { return &SketchCreateBlockTool{} }

func (t *SketchCreateBlockTool) Name() string { return "Create Block" }
func (t *SketchCreateBlockTool) Prompt(*Session) string {
	return "Select geometry, name the block, then OK."
}

// SetBlockName names the definition the commit will create.
func (t *SketchCreateBlockTool) SetBlockName(name string) { t.name = name }

// CanCommit needs a selection and a name.
func (t *SketchCreateBlockTool) CanCommit() bool {
	return len(t.picks) > 0 && t.name != ""
}

func (t *SketchCreateBlockTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "createBlock")
	if err != nil {
		return err
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	_, _, err = sk.Blocks().CreateFromSelection(part.SketchBlocks(), t.name, t.picks)
	return err
}

// SketchPlaceBlockTool stamps instances of a definition at clicked points.
type SketchPlaceBlockTool struct {
	definition string
	rotation   float64
	scale      float64
	at         *math.Point2
}

// NewSketchPlaceBlockTool returns the placement tool for the named definition.
func NewSketchPlaceBlockTool(definition string) *SketchPlaceBlockTool {
	return &SketchPlaceBlockTool{definition: definition, scale: 1}
}

func (t *SketchPlaceBlockTool) Name() string { return "Place Block" }
func (t *SketchPlaceBlockTool) Prompt(*Session) string {
	if t.definition == "" {
		return "Choose the block to place, then click its insertion point."
	}
	return fmt.Sprintf("Click the insertion point for block %q.", t.definition)
}

// SetDefinition / SetRotation / SetScale feed the placement parameters from
// the dialog.
func (t *SketchPlaceBlockTool) SetDefinition(name string)    { t.definition = name }
func (t *SketchPlaceBlockTool) SetRotation(radians float64)  { t.rotation = radians }
func (t *SketchPlaceBlockTool) SetScale(factor float64)      { t.scale = factor }
func (t *SketchPlaceBlockTool) Start(*Session)               {}
func (t *SketchPlaceBlockTool) Pick(*Session, Selectable)    {}
func (t *SketchPlaceBlockTool) Cancel(*Session)              { t.at = nil }
func (t *SketchPlaceBlockTool) CanCommit() bool              { return t.at != nil && t.definition != "" }
func (t *SketchPlaceBlockTool) AutoCommits() bool            { return true }
func (t *SketchPlaceBlockTool) ClickAt(_ *Session, px, py float64) {
	p := math.P2(math.Scalar(px), math.Scalar(py))
	t.at = &p
}

func (t *SketchPlaceBlockTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "placeBlock")
	if err != nil {
		return err
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	def, ok := part.SketchBlocks().ByName(t.definition)
	if !ok {
		return errors.New("placeBlock: block definition " + t.definition + " does not exist")
	}
	if t.at == nil {
		return errors.New("placeBlock: click the insertion point first")
	}
	sk.Blocks().Insert(def, sketch.PlacementTransform(*t.at, t.rotation, t.scale))
	t.at = nil // ready for the next stamp (the tool auto-commits per click)
	return nil
}
