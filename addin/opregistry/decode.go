// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

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

// decodeSheetMetalArgs is decodeFeatureArgs's sheet-metal twin: it resolves the active part
// through activeSheetMetalPart (erroring, named by op, when it is absent or not in the
// sheet-metal environment) instead of modelaccess.ActivePart, and wraps an unmarshal failure
// with op so every sheetMetal* handler reports the same "op: invalid args: …" shape.
func decodeSheetMetalArgs[T any](s *app.Session, raw json.RawMessage, op string) (*compdef.PartComponentDefinition, T, error) {
	var in T
	part, err := activeSheetMetalPart(s, op)
	if err != nil {
		return nil, in, err
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, in, fmt.Errorf("%s: invalid args: %w", op, err)
	}
	return part, in, nil
}
