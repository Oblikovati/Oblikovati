// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import "oblikovati.org/math"

// OccurrenceListener observes occurrence mutations after they apply, so an owner — the
// assembly definition's event source — can raise domain events without this package
// depending on it (the dependency points inward; model/occurrence must not import
// model/compdef). Every method reports an After-phase fact: the change has already
// happened, so a listener never vetoes here. It is the sink behind the reference API's
// AssemblyEvents occurrence notifications (M11-F07, #632).
//
// Install one with [Occurrences.SetListener]; mutations on the collection and on its
// occurrences then call the matching method.
type OccurrenceListener interface {
	// OccurrenceAdded fires after o is placed in the collection.
	OccurrenceAdded(o *Occurrence)
	// OccurrenceRemoved fires after o is deleted from the collection.
	OccurrenceRemoved(o *Occurrence)
	// OccurrenceReplaced fires after o's definition is swapped, carrying the prior one.
	OccurrenceReplaced(o *Occurrence, previous Definition)
	// OccurrenceTransformed fires after o is repositioned, carrying its prior placement.
	// Inside a drag batch ([Occurrences.SuspendNotifications]) it is coalesced to one
	// call per occurrence at resume, carrying the placement held when the batch began.
	OccurrenceTransformed(o *Occurrence, previous math.Matrix4)
	// OccurrenceSuppressionChanged fires after o's suppression flag actually changes.
	OccurrenceSuppressionChanged(o *Occurrence)
}

// silentListener is the null-object default so the collection never nil-checks a
// listener. SetListener(nil) restores it.
type silentListener struct{}

func (silentListener) OccurrenceAdded(*Occurrence)                     {}
func (silentListener) OccurrenceRemoved(*Occurrence)                   {}
func (silentListener) OccurrenceReplaced(*Occurrence, Definition)      {}
func (silentListener) OccurrenceTransformed(*Occurrence, math.Matrix4) {}
func (silentListener) OccurrenceSuppressionChanged(*Occurrence)        {}
