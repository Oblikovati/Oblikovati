// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// modelTree returns a read-only snapshot of the active part: parameter names, sketch
// count, the feature program, and the resulting body count.
func modelTree(s *app.Session, part *compdef.PartComponentDefinition) (wire.ModelTreeResult, error) {
	params := part.Parameters().All()
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name()
	}
	return wire.ModelTreeResult{
		Document:   s.ActiveDocument().DisplayName(),
		Parameters: names,
		Sketches:   part.Sketches().Count(),
		Features:   projectAll(part.Features(), func(_ int, f *feature.PartFeature) wire.FeatureInfo { return featureInfo(f) }),
		Bodies:     part.SurfaceBodies().Count(),
	}, nil
}

// modelSelection returns the current selection summary (read-only). Mutating the
// selection by reference key is a fast-follow.
func modelSelection(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(currentSelection(s))
}
