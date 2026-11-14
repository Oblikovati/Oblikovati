// SPDX-License-Identifier: GPL-2.0-only
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

type record struct {
	Side    int    `json:"side_index"`
	Source  uint64 `json:"source_line_id"`
	FaceKey string `json:"face_key_hex"`
}

func sideLineage(part *compdef.PartComponentDefinition) ([]record, error) {
	profiles := part.Sketches().Item(0).Profiles()
	if profiles.Count() != 1 || !profiles.Item(0).IsClosed() {
		return nil, fmt.Errorf("profile count %d: expected exactly one closed profile", profiles.Count())
	}
	loop := profiles.Item(0).OuterLoop()
	entities, polygon := loop.Entities(), loop.Polygon()
	if len(entities) != len(polygon) {
		return nil, fmt.Errorf("entities %d and vertices %d: expected one source per boundary edge", len(entities), len(polygon))
	}
	faces := part.SurfaceBodies().All()[0].Faces()
	keys := map[int]string{}
	for _, face := range faces {
		tokens := face.Lineage().Tokens()
		if len(tokens) != 1 || tokens[0].Role != "side" {
			continue
		}
		i := tokens[0].Index
		if _, exists := keys[i]; exists {
			return nil, fmt.Errorf("side index %d: expected unique face", i)
		}
		keys[i] = hex.EncodeToString(face.ReferenceKey())
	}
	if len(keys) != len(entities) {
		return nil, fmt.Errorf("side faces %d, entities %d: expected one face per boundary", len(keys), len(entities))
	}
	result := make([]record, 0, len(entities))
	var area float64
	for i, source := range entities {
		a, b := polygon[i], polygon[(i+1)%len(polygon)]
		area += float64(a.X*b.Y - b.X*a.Y)
		line, ok := source.Entity.(*sketch.Line)
		if !ok {
			return nil, fmt.Errorf("source %d: expected line, got %T", i, source.Entity)
		}
		if !((line.A.Position() == a && line.B.Position() == b) || (line.A.Position() == b && line.B.Position() == a)) {
			return nil, fmt.Errorf("source %d: expected exact endpoints for boundary", i)
		}
		key, ok := keys[i]
		if !ok {
			return nil, fmt.Errorf("side index %d: expected named face", i)
		}
		result = append(result, record{i, uint64(line.EntityID()), key})
	}
	if area <= 0 {
		return nil, fmt.Errorf("signed area %g: expected positive polygon traversal", area)
	}
	return result, nil
}
func main() {
	if len(os.Args) != 2 {
		panic("expected fixture path")
	}
	document, err := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default()).Open(os.Args[1], true)
	if err != nil {
		panic(err)
	}
	part := document.Content().(*compdef.PartComponentDefinition)
	before, err := sideLineage(part)
	if err != nil {
		panic(err)
	}
	size, ok := part.Parameters().ByName("size")
	if !ok {
		panic("size parameter missing")
	}
	if err = part.Parameters().SetExpression(size.ID(), "20 mm"); err != nil {
		panic(err)
	}
	part.RecomputeAfterChange()
	after, err := sideLineage(part)
	if err != nil {
		panic(err)
	}
	out, err := json.MarshalIndent(map[string]any{"before": before, "after": after}, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
}
