// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// Two more face-referenced features: a boss (a cylindrical stud raised from a face) and a
// cosmetic thread applied to a cylindrical face. Both take a single faceRef reference key.

type bossArgs struct {
	FaceRef  string `json:"faceRef"`
	Diameter string `json:"diameter"`
	Height   string `json:"height"`
}

const bossSchema = `{
  "type": "object",
  "properties": {
    "faceRef": {"type": "string", "description": "Reference key of the planar face to grow the boss from (get_reference_keys)."},
    "diameter": {"type": "string", "description": "Boss diameter, e.g. \"8 mm\"."},
    "height": {"type": "string", "description": "Boss height, e.g. \"5 mm\"."}
  },
  "required": ["faceRef", "diameter", "height"]
}`

func bossDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "boss", Summary: "Raise a cylindrical boss from a face.", Schema: json.RawMessage(bossSchema), Apply: applyBoss}
}

func applyBoss(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in bossArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if in.FaceRef == "" {
		return nil, errors.New("boss: faceRef is empty")
	}
	dia, err := lengthValue(part, in.Diameter, "boss: diameter")
	if err != nil {
		return nil, err
	}
	h, err := lengthValue(part, in.Height, "boss: height")
	if err != nil {
		return nil, err
	}
	pf := feature.NewBossFeatures(part.Features()).Add([]byte(in.FaceRef), constFn(dia), constFn(h))
	return recomputeResult(part, pf)
}

type threadArgs struct {
	FaceRef     string `json:"faceRef"`
	Designation string `json:"designation"`
}

const threadSchema = `{
  "type": "object",
  "properties": {
    "faceRef": {"type": "string", "description": "Reference key of the cylindrical face to thread (get_reference_keys)."},
    "designation": {"type": "string", "description": "Thread designation, e.g. \"M8x1.25\"."}
  },
  "required": ["faceRef", "designation"]
}`

func threadDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "thread", Summary: "Apply a cosmetic thread to a cylindrical face.", Schema: json.RawMessage(threadSchema), Apply: applyThread}
}

func applyThread(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in threadArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if in.FaceRef == "" || in.Designation == "" {
		return nil, errors.New("thread: faceRef and designation are required")
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddThread([]byte(in.FaceRef), in.Designation)
	return recomputeResult(part, pf)
}
