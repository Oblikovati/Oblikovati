// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"strconv"

	"oblikovati.org/event"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/health"
	"oblikovati.org/model/occurrence"
)

// endOfFeaturesAtEnd is the assembly EOF marker value meaning "evaluate every feature".
const endOfFeaturesAtEnd = -1

// AssemblyFeature is one machining feature authored in the assembly: a reused part
// feature triangle ([feature.Feature]) plus the assembly-context state the part
// engine has no notion of — which occurrences it participates on, its suppression,
// and its evaluation health. Authored in the assembly, it cuts/modifies the placed
// geometry of its participant occurrences in place, never the shared part definitions
// (M11-F08, #633).
type AssemblyFeature struct {
	id           uint64
	name         string
	feature      feature.Feature
	participants map[*occurrence.Occurrence]bool
	order        []*occurrence.Occurrence // first-added participant order, for determinism
	// pathFilter, when non-empty, restricts machining to specific nested occurrence
	// paths (keyed by pathKey) — disambiguating a sub-assembly placed more than once.
	// Empty means "every path through a participating leaf occurrence" (the default).
	pathFilter map[string]occurrence.OccurrencePath
	suppress   bool
	health     health.Health
}

// ID returns the feature's stable handle within its collection (unchanged by rename).
func (f *AssemblyFeature) ID() uint64 { return f.id }

// Name/SetName get and set the display name; the id is stable across renames.
func (f *AssemblyFeature) Name() string     { return f.name }
func (f *AssemblyFeature) SetName(n string) { f.name = n }

// Kind returns the wrapped feature's type name.
func (f *AssemblyFeature) Kind() string { return f.feature.Kind() }

// Definition returns the wrapped feature recipe (the reused part triangle).
func (f *AssemblyFeature) Definition() feature.Feature { return f.feature }

// Health returns the feature's current evaluation health.
func (f *AssemblyFeature) Health() health.Health { return f.health }

// Suppressed reports whether the feature is explicitly suppressed.
func (f *AssemblyFeature) Suppressed() bool { return f.suppress }

// SetSuppressed toggles explicit suppression. A suppressed assembly feature passes its
// participants' geometry through unchanged on the next recompute.
func (f *AssemblyFeature) SetSuppressed(s bool) { f.suppress = s }

// AddParticipant adds o to the occurrences this feature machines (idempotent). Later-
// added components do not participate until added here — matching the reference API,
// where only components present when the feature is created participate by default.
func (f *AssemblyFeature) AddParticipant(o *occurrence.Occurrence) {
	if f.participants[o] {
		return
	}
	f.participants[o] = true
	f.order = append(f.order, o)
}

// RemoveParticipant drops o from participation (idempotent), so the feature no longer
// affects that occurrence.
func (f *AssemblyFeature) RemoveParticipant(o *occurrence.Occurrence) {
	if !f.participants[o] {
		return
	}
	delete(f.participants, o)
	for i, p := range f.order {
		if p == o {
			f.order = append(f.order[:i], f.order[i+1:]...)
			break
		}
	}
}

// SetParticipants replaces the feature's participation set with occs (in order),
// dropping any current participant not in the new set. It is the "set participation"
// edit the wire surface drives, distinct from incremental add/remove.
func (f *AssemblyFeature) SetParticipants(occs []*occurrence.Occurrence) {
	for _, o := range f.Participants() {
		f.RemoveParticipant(o)
	}
	for _, o := range occs {
		f.AddParticipant(o)
	}
}

// Participates reports whether o is in this feature's participation set.
func (f *AssemblyFeature) Participates(o *occurrence.Occurrence) bool { return f.participants[o] }

// SetParticipantPaths restricts the feature to the given nested occurrence paths,
// disambiguating a sub-assembly placed more than once (each placement is a distinct
// path to the same shared leaf). Passing no paths clears the restriction, so the
// feature again machines every path through a participating leaf occurrence (the
// default). A path still only takes effect if its leaf occurrence is a participant.
func (f *AssemblyFeature) SetParticipantPaths(paths []occurrence.OccurrencePath) {
	if len(paths) == 0 {
		f.pathFilter = nil
		return
	}
	f.pathFilter = make(map[string]occurrence.OccurrencePath, len(paths))
	for _, p := range paths {
		f.pathFilter[pathKey(p)] = append(occurrence.OccurrencePath(nil), p...)
	}
}

// ParticipantPaths returns the feature's path restriction, or nil when it is
// unrestricted (machining every path through a participating leaf).
func (f *AssemblyFeature) ParticipantPaths() []occurrence.OccurrencePath {
	if len(f.pathFilter) == 0 {
		return nil
	}
	out := make([]occurrence.OccurrencePath, 0, len(f.pathFilter))
	for _, p := range f.pathFilter {
		out = append(out, p)
	}
	return out
}

// participatesContribution reports whether the feature machines a placed contribution
// reached through leaf at the given path key: its leaf must be a participant, and —
// when a path restriction is set — its path must be listed.
func (f *AssemblyFeature) participatesContribution(leaf *occurrence.Occurrence, key string) bool {
	if !f.participants[leaf] {
		return false
	}
	if len(f.pathFilter) == 0 {
		return true
	}
	_, ok := f.pathFilter[key]
	return ok
}

// Participants returns the participant occurrences in first-added order.
func (f *AssemblyFeature) Participants() []*occurrence.Occurrence {
	return append([]*occurrence.Occurrence(nil), f.order...)
}

// AssemblyFeatures is the assembly's ordered feature program and its participation
// state — the assembly analogue of [feature.PartFeatures]. It hosts reused part
// feature triangles in assembly context: it tracks an end-of-features rollback marker,
// supports batch suppression, and on [AssemblyFeatures.Recompute] threads each
// unsuppressed feature through the assembly-space bodies of its participant
// occurrences, recording per-occurrence results without touching the shared part
// definitions (M11-F08, #633).
type AssemblyFeatures struct {
	items  []*AssemblyFeature
	byID   map[uint64]*AssemblyFeature
	nextID uint64
	eof    int // end-of-features feature index; endOfFeaturesAtEnd ⇒ full program
	// result holds each placed contribution's machined assembly-space bodies after the
	// last recompute, keyed by occurrence-path key; resultLeaf maps that key back to the
	// leaf occurrence so Result aggregates per occurrence. See assembly_features_recompute.go.
	result     map[string][]*topo.Body
	resultLeaf map[string]*occurrence.Occurrence
	// bus, when set by the assembly definition, is where the program raises
	// AssemblyFeaturesRecomputed after each recompute (see assembly_features_events.go).
	bus *event.Bus
}

// NewAssemblyFeatures returns an empty assembly feature program with the EOF marker at
// the end (every feature evaluated).
func NewAssemblyFeatures() *AssemblyFeatures {
	return &AssemblyFeatures{byID: map[uint64]*AssemblyFeature{}, eof: endOfFeaturesAtEnd}
}

// Add appends an assembly feature wrapping f, participating on the given occurrences
// (the default participation the assembly snapshots from the components present). The
// feature starts unsuppressed and healthy.
func (fs *AssemblyFeatures) Add(f feature.Feature, participants []*occurrence.Occurrence) *AssemblyFeature {
	fs.nextID++
	af := &AssemblyFeature{
		id:           fs.nextID,
		name:         f.Kind(),
		feature:      f,
		participants: map[*occurrence.Occurrence]bool{},
		health:       health.Healthy,
	}
	for _, o := range participants {
		af.AddParticipant(o)
	}
	fs.items = append(fs.items, af)
	fs.byID[af.id] = af
	return af
}

// Remove deletes the feature with the given id, reporting whether it was present.
func (fs *AssemblyFeatures) Remove(id uint64) bool {
	if _, ok := fs.byID[id]; !ok {
		return false
	}
	delete(fs.byID, id)
	for i, af := range fs.items {
		if af.id == id {
			fs.items = append(fs.items[:i], fs.items[i+1:]...)
			break
		}
	}
	return true
}

// Count returns the number of features; Item returns the i-th in history order.
func (fs *AssemblyFeatures) Count() int                  { return len(fs.items) }
func (fs *AssemblyFeatures) Item(i int) *AssemblyFeature { return fs.items[i] }

// ByID returns the feature with the given id.
func (fs *AssemblyFeatures) ByID(id uint64) (*AssemblyFeature, bool) {
	f, ok := fs.byID[id]
	return f, ok
}

// ByName returns the first feature with the given name.
func (fs *AssemblyFeatures) ByName(name string) (*AssemblyFeature, bool) {
	for _, f := range fs.items {
		if f.name == name {
			return f, true
		}
	}
	return nil, false
}

// UniqueName returns base suffixed with the smallest positive integer not already a
// feature name, so the browser shows distinct rows (mirrors PartFeatures.UniqueName).
func (fs *AssemblyFeatures) UniqueName(base string) string {
	for n := 1; ; n++ {
		name := base + strconv.Itoa(n)
		if _, taken := fs.ByName(name); !taken {
			return name
		}
	}
}

// EndOfFeaturesPosition returns the feature index the program evaluates up to, or -1
// when every feature is evaluated (the marker is at the end).
func (fs *AssemblyFeatures) EndOfFeaturesPosition() int { return fs.eof }

// IsRolledBack reports whether the EOF marker sits before the end of the program, so
// some trailing features are currently rolled back (not evaluated).
func (fs *AssemblyFeatures) IsRolledBack() bool { return fs.eof != endOfFeaturesAtEnd }

// SetEndOfFeatures moves the EOF marker to position, rolling the assembly back to that
// point; features at or after it are skipped on the next recompute. A negative
// position restores the marker to the end (see [AssemblyFeatures.RollToEnd]).
func (fs *AssemblyFeatures) SetEndOfFeatures(position int) {
	if position < 0 {
		position = endOfFeaturesAtEnd
	}
	fs.eof = position
}

// RollToEnd moves the EOF marker back to the end, re-including every feature.
func (fs *AssemblyFeatures) RollToEnd() { fs.SetEndOfFeatures(endOfFeaturesAtEnd) }

// SuppressFeatures suppresses every named feature in one batch — the reference API's
// SuppressFeatures. Unknown ids are ignored.
func (fs *AssemblyFeatures) SuppressFeatures(ids ...uint64) { fs.setSuppressed(ids, true) }

// UnsuppressFeatures clears suppression on every named feature in one batch — the
// reference API's UnsuppressFeatures. Unknown ids are ignored.
func (fs *AssemblyFeatures) UnsuppressFeatures(ids ...uint64) { fs.setSuppressed(ids, false) }

// setSuppressed applies a suppression state to each named feature that exists.
func (fs *AssemblyFeatures) setSuppressed(ids []uint64, suppressed bool) {
	for _, id := range ids {
		if af, ok := fs.byID[id]; ok {
			af.suppress = suppressed
		}
	}
}

// effectiveEnd returns the evaluation cutoff (the EOF marker, clamped to the length).
func (fs *AssemblyFeatures) effectiveEnd() int {
	if fs.eof == endOfFeaturesAtEnd || fs.eof > len(fs.items) {
		return len(fs.items)
	}
	return fs.eof
}
