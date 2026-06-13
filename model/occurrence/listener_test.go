// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import (
	"testing"

	"oblikovati.org/math"
)

// recordingListener is a named fake OccurrenceListener that logs every notification in
// order, so a test can assert exactly which mutations fired and with what payload.
type recordingListener struct {
	added       []*Occurrence
	removed     []*Occurrence
	replaced    []replaceCall
	transformed []transformCall
	suppressed  []*Occurrence
}

type replaceCall struct {
	occ      *Occurrence
	previous Definition
}

type transformCall struct {
	occ      *Occurrence
	previous math.Matrix4
}

func (r *recordingListener) OccurrenceAdded(o *Occurrence)   { r.added = append(r.added, o) }
func (r *recordingListener) OccurrenceRemoved(o *Occurrence) { r.removed = append(r.removed, o) }
func (r *recordingListener) OccurrenceReplaced(o *Occurrence, previous Definition) {
	r.replaced = append(r.replaced, replaceCall{o, previous})
}
func (r *recordingListener) OccurrenceTransformed(o *Occurrence, previous math.Matrix4) {
	r.transformed = append(r.transformed, transformCall{o, previous})
}
func (r *recordingListener) OccurrenceSuppressionChanged(o *Occurrence) {
	r.suppressed = append(r.suppressed, o)
}

// listenIn returns a fresh collection with a recording listener installed.
func listenIn() (*Occurrences, *recordingListener) {
	occ := NewOccurrences()
	rec := &recordingListener{}
	occ.SetListener(rec)
	return occ, rec
}

func unitBox() fakeComponent {
	return fakeComponent{box: math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1))}
}

// TestListenerSeesStructuralMutations covers add/remove/replace notifications and their
// payloads — the After-phase facts the assembly turns into domain events (M11-F07).
func TestListenerSeesStructuralMutations(t *testing.T) {
	occ, rec := listenIn()
	first := unitBox()
	o := occ.AddByComponentDefinition("a:1", first, math.Identity4())
	if len(rec.added) != 1 || rec.added[0] != o {
		t.Fatalf("OccurrenceAdded = %v, want one call for the placed occurrence", rec.added)
	}

	second := unitBox()
	if !occ.Replace(o, second) {
		t.Fatal("Replace reported the occurrence as foreign")
	}
	if len(rec.replaced) != 1 || rec.replaced[0].occ != o || rec.replaced[0].previous != Definition(first) {
		t.Fatalf("OccurrenceReplaced = %+v, want one call carrying the prior definition", rec.replaced)
	}

	if !occ.Remove(o) {
		t.Fatal("Remove reported the occurrence as foreign")
	}
	if len(rec.removed) != 1 || rec.removed[0] != o {
		t.Fatalf("OccurrenceRemoved = %v, want one call for the removed occurrence", rec.removed)
	}
}

// TestListenerSeesTransformAndSuppression covers the per-occurrence placement and
// suppression notifications, including that a no-op suppression toggle stays silent.
func TestListenerSeesTransformAndSuppression(t *testing.T) {
	occ, rec := listenIn()
	o := occ.AddByComponentDefinition("a:1", unitBox(), math.Identity4())

	moved := math.Translation4(math.V3(5, 0, 0))
	o.SetTransform(moved)
	if len(rec.transformed) != 1 || rec.transformed[0].previous != math.Identity4() {
		t.Fatalf("OccurrenceTransformed = %+v, want one call carrying the prior (identity) placement", rec.transformed)
	}

	o.SetSuppressed(true)
	o.SetSuppressed(true) // no-op: already suppressed, must not re-fire
	if len(rec.suppressed) != 1 || rec.suppressed[0] != o {
		t.Fatalf("OccurrenceSuppressionChanged = %v, want exactly one call (no-op toggle stays silent)", rec.suppressed)
	}
}

// TestDragBatchCoalescesTransforms proves the solver-drag batch: many per-step moves of
// one occurrence collapse to a single notification carrying the pre-batch placement,
// while a second moved occurrence flushes once in first-moved order (M11-F07).
func TestDragBatchCoalescesTransforms(t *testing.T) {
	occ, rec := listenIn()
	a := occ.AddByComponentDefinition("a:1", unitBox(), math.Identity4())
	b := occ.AddByComponentDefinition("b:1", unitBox(), math.Identity4())
	rec.transformed = nil // ignore the adds

	occ.SuspendNotifications()
	for i := 1; i <= 100; i++ {
		a.SetTransform(math.Translation4(math.V3(float64(i), 0, 0)))
	}
	b.SetTransform(math.Translation4(math.V3(0, 7, 0)))
	if len(rec.transformed) != 0 {
		t.Fatalf("got %d notifications during the batch, want 0 (coalesced)", len(rec.transformed))
	}
	occ.ResumeNotifications()

	if len(rec.transformed) != 2 {
		t.Fatalf("after resume got %d notifications, want 2 (one per moved occurrence)", len(rec.transformed))
	}
	if rec.transformed[0].occ != a || rec.transformed[0].previous != math.Identity4() {
		t.Errorf("first flush = %+v, want occurrence a with its pre-batch (identity) placement", rec.transformed[0])
	}
	if rec.transformed[1].occ != b {
		t.Errorf("second flush occurrence = %v, want b (first-moved order)", rec.transformed[1].occ)
	}
	// Revision still advanced per step (geometry version is not batched).
	if occ.Revision() == 0 {
		t.Error("revision did not advance during the batch")
	}
}

// TestDragBatchSkipsNetZeroMove: an occurrence dragged out and back to its starting
// placement emits no event at resume (nothing net changed).
func TestDragBatchSkipsNetZeroMove(t *testing.T) {
	occ, rec := listenIn()
	o := occ.AddByComponentDefinition("a:1", unitBox(), math.Identity4())
	rec.transformed = nil

	occ.SuspendNotifications()
	o.SetTransform(math.Translation4(math.V3(9, 0, 0)))
	o.SetTransform(math.Identity4()) // back to start
	occ.ResumeNotifications()

	if len(rec.transformed) != 0 {
		t.Fatalf("net-zero drag emitted %d notifications, want 0", len(rec.transformed))
	}
}

// TestNestedBatchFlushesAtOutermostResume: only the outermost ResumeNotifications ends
// the batch; an inner resume must not flush early.
func TestNestedBatchFlushesAtOutermostResume(t *testing.T) {
	occ, rec := listenIn()
	o := occ.AddByComponentDefinition("a:1", unitBox(), math.Identity4())
	rec.transformed = nil

	occ.SuspendNotifications()
	occ.SuspendNotifications()
	o.SetTransform(math.Translation4(math.V3(3, 0, 0)))
	occ.ResumeNotifications() // inner — still batched
	if len(rec.transformed) != 0 {
		t.Fatalf("inner resume flushed %d notifications, want 0 (batch still open)", len(rec.transformed))
	}
	occ.ResumeNotifications() // outer — flush now
	if len(rec.transformed) != 1 {
		t.Fatalf("outer resume flushed %d notifications, want 1", len(rec.transformed))
	}
}

// TestSetListenerNilDetaches: passing nil restores the silent listener so later
// mutations don't panic and nothing is recorded.
func TestSetListenerNilDetaches(t *testing.T) {
	occ, rec := listenIn()
	occ.SetListener(nil)
	o := occ.AddByComponentDefinition("a:1", unitBox(), math.Identity4())
	o.SetTransform(math.Translation4(math.V3(1, 0, 0)))
	if len(rec.added) != 0 || len(rec.transformed) != 0 {
		t.Errorf("detached listener still received calls: added=%d transformed=%d", len(rec.added), len(rec.transformed))
	}
}
