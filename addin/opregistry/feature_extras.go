// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// Two more face-referenced features: a boss (a cylindrical stud raised from a face) and a
// cosmetic thread applied to a cylindrical face. Both take a single faceRef reference key.

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
	return &OperationDescriptor{Name: featureargs.KindBoss, Summary: "Raise a cylindrical boss from a face.", Schema: json.RawMessage(bossSchema), Apply: applyBoss}
}

func applyBoss(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Boss](s, raw)
	if err != nil {
		return nil, err
	}
	if in.FaceRef == "" {
		return nil, errors.New("boss: faceRef is empty")
	}
	dia, err := lengthClosure(part, in.Diameter, "boss: diameter")
	if err != nil {
		return nil, err
	}
	h, err := lengthClosure(part, in.Height, "boss: height")
	if err != nil {
		return nil, err
	}
	pf := feature.NewBossFeatures(part.Features()).Add([]byte(in.FaceRef), dia, h)
	return recomputeResult(part, pf)
}

const threadSchema = `{
  "type": "object",
  "properties": {
    "faceRef": {"type": "string", "description": "Reference key of the cylindrical face to thread (get_reference_keys)."},
    "designation": {"type": "string", "description": "Thread designation, e.g. \"M8x1.25\" (enumerable via threads.tableQuery)."},
    "cut": {"type": "boolean", "default": false, "description": "false = cosmetic thread (display only); true = model a real cut thread (the face becomes a threaded surface)."},
    "class": {"type": "string", "description": "Tolerance class recorded on the spec, e.g. \"6H\" (enumerable via threads.tableQuery)."},
    "tapered": {"type": "boolean", "default": false, "description": "Pipe-thread (tapered) data; a cut tapered thread is rejected — model it cosmetic."},
    "modelDiameter": {"type": "string", "enum": ["major", "minor", "pitch", "tapDrill"], "description": "Which thread diameter the modeled face represents (default major)."},
    "length": {"type": "string", "description": "Threaded run along the axis (distance expression) from the face's start edge + offset; empty = full length (Inventor FullDepth)."},
    "offset": {"type": "string", "description": "Distance expression from the face's start edge to where the thread begins; empty = 0. Thread the two ends of a stud with two features on one face."}
  },
  "required": ["faceRef", "designation"]
}`

func threadDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindThread, Summary: "Thread a cylindrical face — cosmetic (display) or a real modeled cut (cut:true).", Schema: json.RawMessage(threadSchema), Apply: applyThread}
}

func applyThread(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Thread](s, raw)
	if err != nil {
		return nil, err
	}
	if in.FaceRef == "" || in.Designation == "" {
		return nil, errors.New("thread: faceRef and designation are required")
	}
	md := types.ModelDiameterFromThread(0)
	if in.ModelDiameter != "" {
		var ok bool
		if md, ok = types.ParseModelDiameterFromThread(in.ModelDiameter); !ok {
			return nil, fmt.Errorf("thread: unknown modelDiameter %q (want major/minor/pitch/tapDrill)", in.ModelDiameter)
		}
	}
	offset, err := optionalLengthClosure(part, in.Offset, "thread: offset")
	if err != nil {
		return nil, err
	}
	length, err := optionalLengthClosure(part, in.Length, "thread: length")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddThreadDef(&feature.ThreadDefinition{
		FaceKey: []byte(in.FaceRef), Designation: in.Designation, Cut: in.Cut,
		Class: in.Class, Tapered: in.Tapered, ModelDiameter: md,
		Offset: offset, Length: length,
	})
	return recomputeResult(part, pf)
}
