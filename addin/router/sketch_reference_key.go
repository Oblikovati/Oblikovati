// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/sketch"
)

// Persistent sketch reference keys (#153). A sketch and each of its entities are named by a
// UUID derived from the document's stable GUID and the object's document-local id (see
// model/identity). The key survives save/load and unrelated edits, so an add-in can hold a
// durable reference. sketch.referenceKey hands out a sketch's key; sketch.resolveReference
// rebinds a stored key (sketch or entity) to its current location.

// registerSketchReferenceKeyHandlers wires the sketch.referenceKey / sketch.resolveReference
// / sketch3d.referenceKey methods.
func (r *Router) registerSketchReferenceKeyHandlers() {
	r.readOnly(wire.MethodSketchReferenceKey, typedPart(sketchReferenceKey))
	r.readOnly(wire.MethodSketchResolveReference, typedPart(sketchResolveReference))
	r.readOnly(wire.MethodSketch3DReferenceKey, typedPart(sketch3DReferenceKey))
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

// keyedSketch is the shared shape of a 2D or 3D sketch the reference-key methods need: its
// own id and its entities. Both *sketch.Sketch and *sketch.Sketch3D satisfy it, so the
// derive/resolve logic is written once for both dimensions.
type keyedSketch interface {
	ID() sketch.ID
	Entities() []sketch.Entity
}

// sketchCollection is a 2D or 3D sketch collection (both expose Count/Item over a keyedSketch).
type sketchCollection[T keyedSketch] interface {
	Count() int
	Item(int) T
}

// sketchReferenceKey returns the persistent reference key of the addressed 2D sketch.
func sketchReferenceKey(s *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.SketchReferenceKeyResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SketchReferenceKeyResult{}, err
	}
	return sketchKeyResult(s, sk)
}

// sketch3DReferenceKey returns the persistent reference key of the addressed 3D sketch.
func sketch3DReferenceKey(s *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.SketchReferenceKeyResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SketchReferenceKeyResult{}, err
	}
	return sketchKeyResult(s, sk)
}

// sketchKeyResult derives a sketch's own reference key (against the active document's GUID) —
// shared by the 2D and 3D referenceKey handlers.
func sketchKeyResult(s *app.Session, sk keyedSketch) (wire.SketchReferenceKeyResult, error) {
	guid, err := activeDocumentGUID(s)
	if err != nil {
		return wire.SketchReferenceKeyResult{}, err
	}
	key, err := identity.SketchKey(guid, uint64(sk.ID()))
	if err != nil {
		return wire.SketchReferenceKeyResult{}, err
	}
	return wire.SketchReferenceKeyResult{ReferenceKey: key}, nil
}

// sketchResolveReference rebinds a stored sketch/entity key (2D or 3D) to its current
// location, or reports found=false when nothing in the active part matches it (the referent
// was deleted). 2D sketches are scanned first, then 3D.
func sketchResolveReference(s *app.Session, part *compdef.PartComponentDefinition, in wire.ResolveSketchReferenceArgs) (wire.ResolveSketchReferenceResult, error) {
	guid, err := activeDocumentGUID(s)
	if err != nil {
		return wire.ResolveSketchReferenceResult{}, err
	}
	res := matchReference(part.Sketches(), guid, in.ReferenceKey, "sketch", "sketchEntity")
	if !res.Found {
		res = matchReference(part.Sketches3D(), guid, in.ReferenceKey, "sketch3d", "sketch3dEntity")
	}
	return res, nil
}

// matchReference scans a 2D or 3D sketch collection for the sketch or entity whose derived
// key equals want, tagging a match with the given kind labels (sketch keys before entity
// keys). A miss is found=false. The same logic serves both dimensions via [keyedSketch].
func matchReference[T keyedSketch](c sketchCollection[T], guid, want, sketchKind, entityKind string) wire.ResolveSketchReferenceResult {
	for i := 0; i < c.Count(); i++ {
		sk := c.Item(i)
		if k, err := identity.SketchKey(guid, uint64(sk.ID())); err == nil && k == want {
			return wire.ResolveSketchReferenceResult{Found: true, Kind: sketchKind, SketchIndex: i}
		}
		if id, ok := matchSketchEntity(sk, guid, want); ok {
			return wire.ResolveSketchReferenceResult{Found: true, Kind: entityKind, SketchIndex: i, EntityID: id}
		}
	}
	return wire.ResolveSketchReferenceResult{Found: false}
}

// matchSketchEntity returns the session id of the sketch entity whose derived key equals want.
func matchSketchEntity(sk keyedSketch, guid, want string) (uint64, bool) {
	for _, e := range sk.Entities() {
		if k, err := identity.SketchEntityKey(guid, uint64(e.EntityID())); err == nil && k == want {
			return uint64(e.EntityID()), true
		}
	}
	return 0, false
}
