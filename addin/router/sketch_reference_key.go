// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/sketch"
)

// Persistent sketch reference keys (#153). A sketch and each of its entities are named by a
// UUID derived from the document's stable GUID and the object's document-local id (see
// model/identity). The key survives save/load and unrelated edits, so an add-in can hold a
// durable reference. sketch.referenceKey hands out a sketch's key; sketch.resolveReference
// rebinds a stored key (sketch or entity) to its current location.

// registerSketchReferenceKeyHandlers wires the sketch.referenceKey / sketch.resolveReference
// methods.
func (r *Router) registerSketchReferenceKeyHandlers() {
	r.handlers[wire.MethodSketchReferenceKey] = sketchReferenceKey
	r.handlers[wire.MethodSketchResolveReference] = sketchResolveReference
}

// activeDocumentGUID returns the active document's stable GUID — the namespace every sketch
// reference key in the document is derived from.
func activeDocumentGUID(s *app.Session) (string, error) {
	d := s.ActiveDocument()
	if d == nil {
		return "", modelaccess.ErrNoActiveDocument
	}
	guid := d.FileIdentity().InternalName
	if guid == "" {
		return "", fmt.Errorf("router: active document %q has no identity guid", d.DisplayName())
	}
	return guid, nil
}

// sketchReferenceKey returns the persistent reference key of the addressed sketch.
func sketchReferenceKey(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	guid, err := activeDocumentGUID(s)
	if err != nil {
		return nil, err
	}
	key, err := identity.SketchKey(guid, uint64(sk.ID()))
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.SketchReferenceKeyResult{ReferenceKey: key})
}

// sketchResolveReference rebinds a stored sketch/entity key to its current location, or
// reports found=false when nothing in the active part matches it (the referent was deleted).
func sketchResolveReference(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ResolveSketchReferenceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	guid, err := activeDocumentGUID(s)
	if err != nil {
		return nil, err
	}
	res := matchReference(part.Sketches(), guid, in.ReferenceKey)
	return json.Marshal(res)
}

// matchReference scans every sketch and entity for the one whose derived key equals want,
// returning the first match (sketch keys before entity keys). A miss is found=false.
func matchReference(sketches *sketch.Sketches, guid, want string) wire.ResolveSketchReferenceResult {
	for i := 0; i < sketches.Count(); i++ {
		sk := sketches.Item(i)
		if k, err := identity.SketchKey(guid, uint64(sk.ID())); err == nil && k == want {
			return wire.ResolveSketchReferenceResult{Found: true, Kind: "sketch", SketchIndex: i}
		}
		if id, ok := matchEntity(sk, guid, want); ok {
			return wire.ResolveSketchReferenceResult{Found: true, Kind: "sketchEntity", SketchIndex: i, EntityID: id}
		}
	}
	return wire.ResolveSketchReferenceResult{Found: false}
}

// matchEntity returns the session id of the sketch entity whose derived key equals want.
func matchEntity(sk *sketch.Sketch, guid, want string) (uint64, bool) {
	for _, e := range sk.Entities() {
		if k, err := identity.SketchEntityKey(guid, uint64(e.EntityID())); err == nil && k == want {
			return uint64(e.EntityID()), true
		}
	}
	return 0, false
}
