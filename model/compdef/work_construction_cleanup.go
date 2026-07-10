// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"bytes"

	"oblikovati.org/model/feature"
	"oblikovati.org/yamlcodec"
)

// Construction (hidden, consumer-tied) work features are auto-deleted when their last consumer is
// deleted (#1849). The host drives that around a delete: ConstructionConsumerSnapshot records the
// construction datums that currently HAVE a consumer; after the delete + recompute,
// PruneConstructionOrphans removes those from the snapshot that now have none. A construction datum
// is thus removed exactly when its last consumer goes, and a never-consumed datum is never touched.
//
// A datum's consumers span three kinds, all enumerated completely so the prune is safe (an
// unenumerated consumer would wrongly orphan an in-use datum): other work datums (WorkGeometry
// refs), sketches (their host work-plane ref), and part features (their datum refs, retained in the
// serialized recipe). A marshal failure retains everything rather than risk a false prune.

// ConstructionConsumerSnapshot returns the refs of live construction datums that currently have at
// least one consumer — the candidates a following delete could orphan (#1849).
func (d *PartComponentDefinition) ConstructionConsumerSnapshot() []string {
	construction := d.work.ConstructionRefs()
	if len(construction) == 0 {
		return nil
	}
	recipe, _ := d.featureRecipeBytes() // a marshal miss only under-counts feature consumers → fewer, safe candidates
	var had []string
	for _, ref := range construction {
		if d.datumHasConsumer(string(ref), recipe) {
			had = append(had, string(ref))
		}
	}
	return had
}

// PruneConstructionOrphans auto-deletes each snapshot datum whose last consumer the just-applied
// delete removed. It returns the count pruned; the caller recomputes afterward so the tombstones
// take effect. A recipe marshal failure retains everything (returns 0) rather than risk a false
// prune of a feature-consumed datum (#1849).
func (d *PartComponentDefinition) PruneConstructionOrphans(hadConsumers []string) int {
	if len(hadConsumers) == 0 {
		return 0
	}
	recipe, ok := d.featureRecipeBytes()
	if !ok {
		return 0 // retain-on-doubt: cannot confirm feature consumers, so prune nothing
	}
	// Loop to a fixpoint: pruning a construction datum can orphan another construction datum that
	// consumed it (a chain of intermediate datums), so re-scan until a pass prunes nothing. The
	// datum→datum check sees each tombstone immediately; the feature recipe does not change here.
	pruned := 0
	for progress := true; progress; {
		progress = false
		for i, ref := range hadConsumers {
			if ref == "" || d.datumHasConsumer(ref, recipe) {
				continue
			}
			d.work.PruneConstructionOrphan(feature.WorkRef(ref))
			hadConsumers[i] = "" // consumed from the worklist
			pruned++
			progress = true
		}
	}
	return pruned
}

// datumHasConsumer reports whether ref is consumed by any other work datum, any sketch's host
// plane, or any part feature (its datum ref retained in the pre-marshaled recipe bytes).
func (d *PartComponentDefinition) datumHasConsumer(ref string, recipe []byte) bool {
	if d.work.RefConsumedByDatum(feature.WorkRef(ref)) {
		return true
	}
	for i := 0; i < d.sketches.Count(); i++ {
		if d.sketches.Item(i).HostWorkRef() == ref {
			return true
		}
	}
	return refTokenAppears(recipe, ref)
}

// featureRecipeBytes marshals the feature program to YAML so a datum ref can be searched as a token
// (features keep their datum refs in the serialized form). ok is false on a marshal failure.
func (d *PartComponentDefinition) featureRecipeBytes() ([]byte, bool) {
	recipe, err := d.features.MarshalRecipe(sketchIndex{d.sketches})
	if err != nil {
		return nil, false
	}
	b, err := yamlcodec.Marshal(recipe)
	if err != nil {
		return nil, false
	}
	return b, true
}

// refTokenAppears reports whether the datum reference ref (e.g. "plane/5") appears in hay as a whole
// token — not as a prefix of a longer ref like "plane/50" (a trailing digit extends it) nor inside a
// larger identifier (a leading letter/digit). This is how a feature's retained datum reference is
// detected in the marshaled recipe.
func refTokenAppears(hay []byte, ref string) bool {
	needle := []byte(ref)
	for from := 0; ; {
		i := bytes.Index(hay[from:], needle)
		if i < 0 {
			return false
		}
		pos := from + i
		end := pos + len(needle)
		headOK := pos == 0 || !isRefIdentByte(hay[pos-1])
		tailOK := end >= len(hay) || !isRefTailByte(hay[end])
		if headOK && tailOK {
			return true
		}
		from = pos + 1
	}
}

// isRefTailByte reports whether b would extend a datum ref's trailing number (so "plane/5" is not
// matched inside "plane/50").
func isRefTailByte(b byte) bool { return b >= '0' && b <= '9' }

// isRefIdentByte reports whether b is an identifier byte that would make the match part of a larger
// token rather than a standalone reference value.
func isRefIdentByte(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
