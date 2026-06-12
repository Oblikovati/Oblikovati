// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/bodyapi"
	"oblikovati.org/model/facetstore"
)

// Session-owned body services (M07): the tolerance-keyed facet/stroke cache
// (#293) and the transient B-rep registry (#628). Both are lazy — most
// sessions never touch them.

// FacetStore returns the session's facet/stroke cache.
func (s *Session) FacetStore() *facetstore.FacetStore {
	if s.facetStore == nil {
		s.facetStore = facetstore.NewFacetStore()
	}
	return s.facetStore
}

// TransientBodies returns the session's transient B-rep factory/registry.
func (s *Session) TransientBodies() *bodyapi.TransientBRep {
	if s.transientBodies == nil {
		s.transientBodies = bodyapi.NewTransientBRep(ops.DefaultQuality())
	}
	return s.transientBodies
}
