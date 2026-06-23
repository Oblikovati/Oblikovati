// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
)

// Serialized form of the Match feature (M36-F05): the continuity order and the two edges.

// MatchData is a surface match's recipe.
type MatchData struct {
	Order      int `yaml:"order"`
	SourceEdge int `yaml:"sourceEdge"`
	TargetEdge int `yaml:"targetEdge"`
}

// serializeMatch captures a match definition as its persisted recipe.
func serializeMatch(def *MatchDefinition) *MatchData {
	return &MatchData{Order: def.Order, SourceEdge: int(def.SourceEdge), TargetEdge: int(def.TargetEdge)}
}

// restoreMatch rebuilds a Match feature from its recipe.
func restoreMatch(fs *PartFeatures, d *MatchData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("match feature is missing its payload")
	}
	return NewMatchFeatures(fs).Add(d.Order, geom.Boundary(d.SourceEdge), geom.Boundary(d.TargetEdge)), nil
}
