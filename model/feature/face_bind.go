// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/identity"
)

// Face references are the counterpart of the edge dress-up references (chamfer.go resolveEdges):
// they too must rebind to a freshly-built body on every recompute, and an upstream edit can
// destroy the exact-keyed face while leaving a defensible successor. This file is the single seam
// every face-key call site binds through (#1579, ADR-0043 P6), so the exact→ancestral→geometric→
// none tier ladder and the P0 collision guard live in one place rather than scattered across the
// dozen FindFaceByKey sites. It is the face mirror of resolveEdges and reuses the SAME tested
// binder (model/identity.fallbackMatch) via the face adapter in identity_adapter.go.

// bindFace binds one face reference key against the running body, recovering through the tiered
// binder when the exact face is gone. It returns the bound face and the match tier (MatchExact, or
// a fallback tier on a heal), or an error when the key is ambiguous or lost with no defensible
// recovery — so a caller goes Sick honestly rather than dressing up the wrong face. anchor is the
// key's mint-time point (nil degrades to ancestral-only recovery).
//
// Example: f, mt, err := bindFace(body, def.PlacementFaceKey, anchorFor(key, def.FaceAnchors))
func bindFace(body *topo.Body, key []byte, anchor *math.Point3) (*topo.Face, identity.MatchType, error) {
	match := body.FacesByKey(key)
	if len(match) == 1 {
		return match[0], identity.MatchExact, nil
	}
	if f, mt := recoverFace(key, anchor, faceEntities(body)); mt.IsFallback() && f != nil {
		return f, mt, nil
	}
	if len(match) > 1 {
		return nil, identity.MatchNone, fmt.Errorf("face reference %q is ambiguous — it matches %d faces (a topological-naming collision) and no surviving sibling could be recovered", keyText(key), len(match))
	}
	return nil, identity.MatchNone, fmt.Errorf("face reference %q lost (no face with that lineage on the running body, and no surviving sibling to recover it)", keyText(key))
}

// resolveFaces binds every face key against the running body, collecting the heals that the engine
// turns into a Warning (ADR-0043 P6). It is the batch, heal-reporting form for features that pick
// several faces and return an Output; a single lost/ambiguous key is a hard error. It mirrors
// resolveEdges so the two reference kinds behave identically.
func resolveFaces(body *topo.Body, keys [][]byte, anchors map[string]math.Point3) ([]*topo.Face, []ReferenceHeal, error) {
	faces := make([]*topo.Face, len(keys))
	var heals []ReferenceHeal
	for i, k := range keys {
		f, mt, err := bindFace(body, k, anchorFor(k, anchors))
		if err != nil {
			return nil, nil, err
		}
		faces[i] = f
		if mt.IsFallback() {
			heals = append(heals, ReferenceHeal{Key: append([]byte(nil), k...), Match: mt})
		}
	}
	return faces, heals, nil
}

// faceHeal returns the one-element heal slice for a single recovered face binding, or nil for an
// exact match — the single-key counterpart of resolveFaces' heal collection, for features that
// pick one face and return an Output (boss, thread, unwrap).
func faceHeal(key []byte, mt identity.MatchType) []ReferenceHeal {
	if !mt.IsFallback() {
		return nil
	}
	return []ReferenceHeal{{Key: append([]byte(nil), key...), Match: mt}}
}

// FindOrRecoverFace resolves a face key, recovering a lone ancestral sibling when the exact face is
// gone — the drop-in replacement for body.FindFaceByKey at best-effort callers that have no health
// channel to report a heal through (display helpers, surface sources, sheet-metal fold maps, some
// in model/compdef). It never guesses: an ambiguous (including a P0 collision) or lost key returns
// (nil, false), so a silent recovery is only ever the unambiguous ancestral one. These sites hold
// no mint-time anchor, so the geometric tier does not apply here.
//
// Example: f, ok := feature.FindOrRecoverFace(body, faceKey)
func FindOrRecoverFace(body *topo.Body, key []byte) (*topo.Face, bool) {
	f, _, err := bindFace(body, key, nil)
	if err != nil {
		return nil, false
	}
	return f, true
}
