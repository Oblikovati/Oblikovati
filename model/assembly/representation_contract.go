// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/model/internal/collview"
)

// Representations satisfies contract.RepresentationsManager (M12-F04): the in-proc read
// surface an add-in uses to enumerate the three families and the model states. The collections
// are thin views over the live slices — one-line collview constructions since #1655, so the
// out-of-range nil guard lives in one tested place; every mutation travels over api/wire.

// DesignViews returns the design-view representation collection.
func (r *Representations) DesignViews() contract.DesignViewRepresentations {
	return collview.Over(r.design, asDesignView)
}

// Positionals returns the positional representation collection.
func (r *Representations) Positionals() contract.PositionalRepresentations {
	return collview.Over(r.pos, asPositional)
}

// LevelsOfDetail returns the level-of-detail representation collection.
func (r *Representations) LevelsOfDetail() contract.LevelOfDetailRepresentations {
	return collview.Over(r.lod, asLevelOfDetail)
}

// ModelStates returns the model-state collection.
func (r *Representations) ModelStates() contract.ModelStates {
	return collview.Over(r.models, asModelState)
}

// The Elem→Iface widenings collview.Over needs (Go does not implicitly convert
// []*designViewRep to []contract.DesignViewRepresentation).
func asDesignView(v *designViewRep) contract.DesignViewRepresentation { return v }
func asPositional(v *positionalRep) contract.PositionalRepresentation { return v }
func asLevelOfDetail(v *lodRep) contract.LevelOfDetailRepresentation  { return v }
func asModelState(v *modelState) contract.ModelState                  { return v }
