// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"errors"
	"testing"

	"oblikovati/event"
)

func TestEventIDsAreStable(t *testing.T) {
	cases := []struct {
		e    event.Event
		want event.TypeID
	}{
		{DocumentCreated{}, 0x0401},
		{DocumentOpened{}, 0x0402},
		{DocumentSave{}, 0x0403},
		{DocumentClose{}, 0x0404},
		{DocumentActivate{}, 0x0405},
		{ApplicationQuit{}, 0x0410},
		{ModelChanged{}, 0x0420},
	}
	for _, c := range cases {
		if got := c.e.EventID(); got != c.want {
			t.Errorf("%T EventID = %#x, want %#x", c.e, got, c.want)
		}
	}
}

func TestVetoErrorMessage(t *testing.T) {
	err := &VetoError{Operation: "close", Reason: "unsaved changes"}
	if err.Error() != "doc: close vetoed: unsaved changes" {
		t.Errorf("VetoError.Error() = %q", err.Error())
	}
}

func TestLifecycleFiresBeforeAndAfterEvents(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store)
	var log []string
	event.Subscribe(ws.Events(), event.After, func(_ event.Context, e DocumentCreated) event.Outcome {
		log = append(log, "created:"+e.Document.DisplayName())
		return event.Continue()
	})
	event.Subscribe(ws.Events(), event.Before, func(_ event.Context, _ DocumentSave) event.Outcome {
		log = append(log, "saving")
		return event.Continue()
	})
	event.Subscribe(ws.Events(), event.After, func(_ event.Context, _ DocumentSave) event.Outcome {
		log = append(log, "saved")
		return event.Continue()
	})

	d, _ := ws.Add(Part, "p.obk", true)
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	want := []string{"created:p", "saving", "saved"}
	if len(log) != 3 || log[0] != want[0] || log[1] != want[1] || log[2] != want[2] {
		t.Errorf("event log = %v, want %v", log, want)
	}
}

func TestCloseCanBeVetoed(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	d, _ := ws.Add(Part, "p.obk", true)

	sub := event.Subscribe(ws.Events(), event.Before, func(_ event.Context, e DocumentClose) event.Outcome {
		if e.Document.Dirty() {
			return event.Veto("unsaved changes")
		}
		return event.Continue()
	})

	// Dirty document: the close is vetoed and the document stays open.
	err := ws.Close(d, true)
	var veto *VetoError
	if !errors.As(err, &veto) {
		t.Fatalf("Close error = %v, want a VetoError", err)
	}
	if _, ok := ws.ByName("p.obk"); !ok {
		t.Error("vetoed close still removed the document")
	}

	// Once the veto condition clears, the close proceeds.
	d.ClearDirty()
	if err := ws.Close(d, true); err != nil {
		t.Fatalf("Close after clearing dirty: %v", err)
	}
	if _, ok := ws.ByName("p.obk"); ok {
		t.Error("document not closed after veto cleared")
	}
	sub.Cancel()
}

func TestSaveAndOpenCanBeVetoed(t *testing.T) {
	store := newFakeStore()
	ws := NewWorkspace(store)
	d, _ := ws.Add(Part, "p.obk", true)

	saveSub := event.Subscribe(ws.Events(), event.Before, func(_ event.Context, _ DocumentSave) event.Outcome {
		return event.Veto("read-only project")
	})
	if err := ws.Save(d); err == nil {
		t.Error("Save was not vetoed")
	}
	saveSub.Cancel()
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save after removing veto: %v", err)
	}
	_ = ws.Close(d, true)

	event.Subscribe(ws.Events(), event.Before, func(_ event.Context, _ DocumentOpened) event.Outcome {
		return event.Veto("blocked")
	})
	if _, err := ws.Open("p.obk", true); err == nil {
		t.Error("Open was not vetoed")
	}
}

func TestQuitVetoAndClose(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	_, _ = ws.Add(Part, "a.obk", true)
	_, _ = ws.Add(Part, "b.obk", true)

	block := true
	event.Subscribe(ws.Events(), event.Before, func(_ event.Context, _ ApplicationQuit) event.Outcome {
		if block {
			return event.Veto("confirm exit")
		}
		return event.Continue()
	})
	if err := ws.Quit(true); err == nil {
		t.Fatal("Quit was not vetoed")
	}
	if ws.Count() != 2 {
		t.Errorf("documents closed despite vetoed quit: count=%d", ws.Count())
	}
	block = false
	if err := ws.Quit(true); err != nil {
		t.Fatalf("Quit: %v", err)
	}
	if ws.Count() != 0 {
		t.Errorf("Quit left %d documents open", ws.Count())
	}
}

func TestDocumentActivateEventOnSetActive(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	a, _ := ws.Add(Part, "a.obk", true)
	_, _ = ws.Add(Part, "b.obk", true)
	activated := ""
	event.Subscribe(ws.Events(), event.After, func(_ event.Context, e DocumentActivate) event.Outcome {
		activated = e.Document.DisplayName()
		return event.Continue()
	})
	if err := ws.SetActiveDocument(a); err != nil {
		t.Fatalf("SetActiveDocument: %v", err)
	}
	if activated != "a" {
		t.Errorf("activated = %q, want a", activated)
	}
}
