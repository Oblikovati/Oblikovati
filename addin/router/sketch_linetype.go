// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"os"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/sketch"
)

// Custom sketch line types from .lin definition files (issue #161):
// sketch.setCustomLineType loads a named pattern onto the sketch (lineType becomes
// "custom"), sketch.getCustomLineType reports the loaded definition. The pattern is
// persisted with the document, so the .lin file is only read here.

// setSketchCustomLineType handles sketch.setCustomLineType.
func setSketchCustomLineType(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetSketchCustomLineTypeArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	if err := guardLineTypeReplace(sk, in); err != nil {
		return nil, err
	}
	def, err := loadLineTypeDefinition(in.FullFileName, in.LineTypeName)
	if err != nil {
		return nil, err
	}
	sk.SetCustomLineType(def, in.FullFileName)
	return customLineTypeResult(sk)
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
func getSketchCustomLineType(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	return customLineTypeResult(sk)
}

// customLineTypeResult marshals the sketch's loaded definition (or loaded=false).
func customLineTypeResult(sk *sketch.Sketch) (json.RawMessage, error) {
	d, file, ok := sk.CustomLineType()
	if !ok {
		return json.Marshal(wire.SketchCustomLineTypeResult{})
	}
	return json.Marshal(wire.SketchCustomLineTypeResult{
		Loaded: true, LineTypeName: d.Name, FullFileName: file, Pattern: d.Pattern,
	})
}
