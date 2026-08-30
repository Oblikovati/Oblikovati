// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"sync/atomic"

	"oblikovati.org/api/types"
	"oblikovati.org/model/health"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/seq"
)

// Sketch entity base (M48 #2245 split of sketch.go). The Entity interface and the embedded base struct
// every sketch entity carries — its id, name, edit/visible/shared flags, health, and display style
// (colour, line type/weight) — plus the process-wide entity id sequence. The Sketch aggregate lives in
// sketch.go.

// ID is the session-stable handle of a sketch or sketch entity, used by
// constraints and selection. Like document ids it is not persisted across
// sessions (regenerated on load); cross-session identity is via reference keys.
type ID uint64

var idSeq atomic.Uint64

func nextID() ID { return ID(idSeq.Add(1)) }

// raiseIDSeq lifts the entity id clock to at least n. Restoring a document pins its
// entity/sketch ids to their persisted (verbatim) values; raising the clock past the
// largest restored id ensures ids minted afterwards never collide with them (#153).
func raiseIDSeq(n uint64) {
	for {
		cur := idSeq.Load()
		if cur >= n {
			return
		}
		if idSeq.CompareAndSwap(cur, n) {
			return
		}
	}
}

// Entity is a piece of sketch geometry (line, arc, circle, point, …). The full
// interface — constrainable points, curve evaluation — is filled in by the entity
// types (M06-F02); here it is the minimum the container needs.
type Entity interface {
	// EntityID returns the entity's sketch-local id.
	EntityID() ID
}

// base holds the identity and edit/visibility/display state common to every sketch kind.
type base struct {
	id      ID
	name    string
	editing bool
	visible bool
	shared  bool   // Inventor PlanarSketch.Shared: stays visible at top level, may feed many features
	seq     uint64 // global creation stamp, shared with features/work features; see model/seq
	health  health.Health

	// Display + solve overrides (Inventor Sketch.Color/LineType/LineWeight/DeferUpdates).
	color        string  // empty ⇒ inherit the document default
	lineType     string  // empty ⇒ inherit (api/types.SketchLineType value)
	lineWeight   float64 // 0 ⇒ inherit
	deferUpdates bool    // true ⇒ batch edits, solve on resume (M21-F08)

	// Custom line type loaded from a .lin definition file (issue #161); non-nil
	// exactly when lineType is "custom". The pattern persists with the document;
	// the source file path is kept only for reporting.
	customLineType     *linetype.Definition
	customLineTypeFile string

	// Per-entity format overrides (#2015). They live on the shared base rather than on Sketch
	// because a 3D sketch carries the same overrides — its Format panel was registered from the
	// same list while only the planar half had storage, so every list on the 3D tab edited
	// nothing (#2039).
	formats   map[ID]EntityFormat // absent ⇒ sketch defaults
	formatRev uint64              // bumped on every format edit, so a drawing cache can see one
}

func newBase(name string) base {
	return base{id: nextID(), name: name, visible: true, seq: seq.Next(), health: health.Healthy}
}

// ID returns the sketch's session id.
func (b *base) ID() ID { return b.id }

// Name returns the sketch's display name.
func (b *base) Name() string { return b.name }

// SetName renames the sketch.
func (b *base) SetName(name string) { b.name = name }

// IsEditing reports whether the sketch is in edit mode (open for geometry changes).
func (b *base) IsEditing() bool { return b.editing }

// Edit enters edit mode; ExitEdit leaves it.
func (b *base) Edit()     { b.editing = true }
func (b *base) ExitEdit() { b.editing = false }

// Visible reports whether the sketch is shown; SetVisible toggles it.
func (b *base) Visible() bool     { return b.visible }
func (b *base) SetVisible(v bool) { b.visible = v }

// Shared reports whether the sketch is shared (Inventor PlanarSketch.Shared): a shared
// sketch stays at the browser's top level even after a feature consumes it, and may be
// consumed by several features. SetShared toggles it (browser "Share Sketch", issue #132).
func (b *base) Shared() bool     { return b.shared }
func (b *base) SetShared(s bool) { b.shared = s }

// Seq returns the sketch's global creation stamp (restoring a saved recipe pins it via
// seq.Restore so reopened documents keep their original interleaving).
func (b *base) Seq() uint64 { return b.seq }

// Health returns the sketch's solve health (set by the solver, M06-F05).
func (b *base) Health() health.Health { return b.health }

// Color returns the sketch's color override ("" ⇒ inherit); SetColor sets it.
func (b *base) Color() string     { return b.color }
func (b *base) SetColor(c string) { b.color = c }

// LineType returns the sketch's line-type override (api/types.SketchLineType value,
// "" ⇒ inherit); SetLineType sets it and drops any loaded custom definition when
// moving to a non-custom style.
func (b *base) LineType() string { return b.lineType }

func (b *base) SetLineType(t string) {
	b.lineType = t
	if t != string(types.SketchLineCustom) {
		b.customLineType, b.customLineTypeFile = nil, ""
	}
}

// SetCustomLineType installs a loaded .lin definition as the sketch's line style
// (lineType becomes "custom"); from is the source file, kept for reporting.
//
//	sk.SetCustomLineType(def, "styles.lin")
func (b *base) SetCustomLineType(d linetype.Definition, from string) {
	b.customLineType, b.customLineTypeFile = &d, from
	b.lineType = string(types.SketchLineCustom)
}

// CustomLineType returns the loaded custom definition, its source file, and
// whether one is present.
func (b *base) CustomLineType() (linetype.Definition, string, bool) {
	if b.customLineType == nil {
		return linetype.Definition{}, "", false
	}
	return *b.customLineType, b.customLineTypeFile, true
}

// LineWeight returns the sketch's line-weight override in cm (0 ⇒ inherit);
// SetLineWeight sets it.
func (b *base) LineWeight() float64     { return b.lineWeight }
func (b *base) SetLineWeight(w float64) { b.lineWeight = w }

// DeferUpdates reports whether the sketch batches edits (solving on resume);
// SetDeferUpdates toggles it (the solve gate is wired in M21-F08).
func (b *base) DeferUpdates() bool     { return b.deferUpdates }
func (b *base) SetDeferUpdates(d bool) { b.deferUpdates = d }
