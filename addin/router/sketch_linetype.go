// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"os"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/sketch"
)

// Custom sketch line types from .lin definition files (issue #161):
// sketch.setCustomLineType loads a named pattern onto the sketch (lineType becomes
// "custom"), sketch.getCustomLineType reports the loaded definition. The pattern is
// persisted with the document, so the .lin file is only read here.

// setSketchCustomLineType handles sketch.setCustomLineType.
func setSketchCustomLineType(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetSketchCustomLineTypeArgs) (wire.SketchCustomLineTypeResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SketchCustomLineTypeResult{}, err
	}
	if err := guardLineTypeReplace(sk, in); err != nil {
		return wire.SketchCustomLineTypeResult{}, err
	}
	def, err := loadLineTypeDefinition(in.FullFileName, in.LineTypeName)
	if err != nil {
		return wire.SketchCustomLineTypeResult{}, err
	}
	sk.SetCustomLineType(def, in.FullFileName)
	return customLineTypeResult(sk), nil
}

// guardLineTypeReplace enforces the replaceExisting contract: re-loading a name that
// is already loaded requires the flag.
func guardLineTypeReplace(sk *sketch.Sketch, in wire.SetSketchCustomLineTypeArgs) error {
	d, _, ok := sk.CustomLineType()
	if ok && !in.ReplaceExisting && d.Name == in.LineTypeName {
		return fmt.Errorf("sketch.setCustomLineType: %q is already loaded and replaceExisting is false", in.LineTypeName)
	}
	return nil
}

// loadLineTypeDefinition reads a .lin file and finds the named definition in it.
func loadLineTypeDefinition(file, name string) (linetype.Definition, error) {
	if name == "" {
		return linetype.Definition{}, fmt.Errorf("sketch.setCustomLineType: lineTypeName is empty (want a definition name from %q)", file)
	}
	src, err := os.ReadFile(file)
	if err != nil {
		return linetype.Definition{}, fmt.Errorf("sketch.setCustomLineType: reading %q: %w", file, err)
	}
	defs, err := linetype.ParseLIN(string(src))
	if err != nil {
		return linetype.Definition{}, fmt.Errorf("sketch.setCustomLineType: %w", err)
	}
	def, ok := linetype.Find(defs, name)
	if !ok {
		return linetype.Definition{}, fmt.Errorf("sketch.setCustomLineType: %q has no definition named %q (%d definitions)", file, name, len(defs))
	}
	return def, nil
}

// getSketchCustomLineType handles sketch.getCustomLineType.
func getSketchCustomLineType(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.SketchCustomLineTypeResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SketchCustomLineTypeResult{}, err
	}
	return customLineTypeResult(sk), nil
}

// customLineTypeResult renders the sketch's loaded definition (or loaded=false).
func customLineTypeResult(sk *sketch.Sketch) wire.SketchCustomLineTypeResult {
	d, file, ok := sk.CustomLineType()
	if !ok {
		return wire.SketchCustomLineTypeResult{}
	}
	return wire.SketchCustomLineTypeResult{
		Loaded: true, LineTypeName: d.Name, FullFileName: file, Pattern: d.Pattern,
	}
}
