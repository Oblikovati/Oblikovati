// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
)

// Sheet-metal Unfold/Refold operations (M13-F04). Unfold flattens every edge bend of the
// active part (leaving it flat for a subsequent cut); Refold re-folds them, carrying any
// edits made while flat. Both take no arguments — they act on the part's recorded bends.

const sheetMetalNoArgsSchema = `{"type": "object", "properties": {}}`

// sheetMetalUnfoldDescriptor is the self-describing "sheetMetalUnfold" operation.
func sheetMetalUnfoldDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalUnfold,
		Summary: "Flatten every bend of the active sheet-metal part (develop it flat) so a following cut works in developed space.",
		Schema:  json.RawMessage(sheetMetalNoArgsSchema),
		Apply:   applySheetMetalUnfold,
	}
}

// sheetMetalRefoldDescriptor is the self-describing "sheetMetalRefold" operation.
func sheetMetalRefoldDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalRefold,
		Summary: "Re-fold the bends an earlier unfold flattened, restoring the folded part and carrying any edits made while flat.",
		Schema:  json.RawMessage(sheetMetalNoArgsSchema),
		Apply:   applySheetMetalRefold,
	}
}

func applySheetMetalUnfold(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalUnfold")
	if err != nil {
		return nil, err
	}
	pf, err := part.AddUnfold()
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

func applySheetMetalRefold(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalRefold")
	if err != nil {
		return nil, err
	}
	pf, err := part.AddRefold()
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}
