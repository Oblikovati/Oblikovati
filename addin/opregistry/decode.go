// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// decodeFeatureArgs resolves the active part and decodes raw into a typed api/wire/featureargs
// value T — the shared front of the feature operations. #1709 replaced the per-kind twin structs
// with the featureargs types decoded here, so the wire shape (what an add-in marshals through
// api/client) and the host decoder are the SAME type and cannot drift.
func decodeFeatureArgs[T any](s *app.Session, raw json.RawMessage) (*compdef.PartComponentDefinition, T, error) {
	var in T
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, in, err
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, in, err
	}
	return part, in, nil
}
