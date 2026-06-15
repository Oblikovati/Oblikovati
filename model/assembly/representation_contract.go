// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/api/contract"

// Representations satisfies contract.RepresentationsManager (M12-F04): the in-proc read
// surface an add-in uses to enumerate the three families and the model states. The collections
// are thin views over the live slices; every mutation travels over api/wire.

// DesignViews returns the design-view representation collection.
func (r *Representations) DesignViews() contract.DesignViewRepresentations {
	return designViewCollection{r.design}
}

// Positionals returns the positional representation collection.
func (r *Representations) Positionals() contract.PositionalRepresentations {
	return positionalCollection{r.pos}
}

// LevelsOfDetail returns the level-of-detail representation collection.
func (r *Representations) LevelsOfDetail() contract.LevelOfDetailRepresentations {
	return lodCollection{r.lod}
}

// ModelStates returns the model-state collection.
func (r *Representations) ModelStates() contract.ModelStates {
	return modelStateCollection{r.models}
}

type designViewCollection struct{ items []*designViewRep }

func (c designViewCollection) Count() int { return len(c.items) }
func (c designViewCollection) Item(i int) contract.DesignViewRepresentation {
	if i < 0 || i >= len(c.items) {
		return nil
	}
	return c.items[i]
}

type positionalCollection struct{ items []*positionalRep }

func (c positionalCollection) Count() int { return len(c.items) }
func (c positionalCollection) Item(i int) contract.PositionalRepresentation {
	if i < 0 || i >= len(c.items) {
		return nil
	}
	return c.items[i]
}

type lodCollection struct{ items []*lodRep }

func (c lodCollection) Count() int { return len(c.items) }
func (c lodCollection) Item(i int) contract.LevelOfDetailRepresentation {
	if i < 0 || i >= len(c.items) {
		return nil
	}
	return c.items[i]
}

type modelStateCollection struct{ items []*modelState }

func (c modelStateCollection) Count() int { return len(c.items) }
func (c modelStateCollection) Item(i int) contract.ModelState {
	if i < 0 || i >= len(c.items) {
		return nil
	}
	return c.items[i]
}
