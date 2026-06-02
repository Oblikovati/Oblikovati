// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/doc"
)

// recorder collects forwarded events (thread-safe; events may fire from any goroutine).
type recorder struct {
	mu  sync.Mutex
	got []wireEvent
}

func (r *recorder) sink(b []byte) {
	var ev wireEvent
	if err := json.Unmarshal(b, &ev); err != nil {
		return
	}
	r.mu.Lock()
	r.got = append(r.got, ev)
	r.mu.Unlock()
}

func (r *recorder) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.got))
	for i, e := range r.got {
		out[i] = e.Type
	}
	return out
}

func TestForwardsDocumentAndCommandEvents(t *testing.T) {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("commands: %v", err)
	}
	if err := s.Commands().Add(app.NewCommand("test.noop", "Noop", "Test", func(*app.Session) error { return nil })); err != nil {
		t.Fatalf("add command: %v", err)
	}
	var rec recorder
	subs := Subscribe(s, rec.sink)
	if len(subs) == 0 {
		t.Fatal("Subscribe returned no subscriptions")
	}

	if _, err := s.Workspace().Add(doc.Part, "ev.obk", true); err != nil {
		t.Fatalf("add document: %v", err)
	}
	if err := s.Execute("test.noop"); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := rec.types()
	if !has(got, "document.created") {
		t.Errorf("missing document.created in %v", got)
	}
	if !has(got, "command.ended") {
		t.Errorf("missing command.ended in %v", got)
	}
}

func TestDocumentCreatedCarriesName(t *testing.T) {
	s := app.NewSession()
	var rec recorder
	Subscribe(s, rec.sink)
	if _, err := s.Workspace().Add(doc.Part, "bracket.obk", true); err != nil {
		t.Fatalf("add document: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.got) != 1 || rec.got[0].Document != "bracket" || rec.got[0].ID == 0 {
		t.Fatalf("event = %+v, want document=bracket with nonzero id", rec.got)
	}
}

func has(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
