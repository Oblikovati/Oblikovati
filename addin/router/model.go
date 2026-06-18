// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// modelTree returns a read-only snapshot of the active part: parameter names, sketch
// count, the feature program, and the resulting body count.
func modelTree(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	params := part.Parameters().All()
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = p.Name()
	}
	feats := part.Features()
	fis := make([]wire.FeatureInfo, feats.Count())
	for i := 0; i < feats.Count(); i++ {
		fis[i] = featureInfo(feats.Item(i))
	}
	return json.Marshal(wire.ModelTreeResult{
		Document:   s.ActiveDocument().DisplayName(),
		Parameters: names,
		Sketches:   part.Sketches().Count(),
		Features:   fis,
		Bodies:     part.SurfaceBodies().Count(),
	})
}

// modelSelection returns the current selection summary (read-only). Mutating the
// selection by reference key is a fast-follow.
func modelSelection(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(currentSelection(s))
}
