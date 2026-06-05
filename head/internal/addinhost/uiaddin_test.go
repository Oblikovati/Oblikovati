//go:build linux || darwin

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Oblikovati/oblikovati/addin/dispatch"
	"github.com/Oblikovati/oblikovati/addin/events"
	"github.com/Oblikovati/oblikovati/addin/opregistry"
	"github.com/Oblikovati/oblikovati/addin/router"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/doc"
)

// TestAddInExtendsRibbonAndActsOnClick is the end-to-end proof of add-in UI
// extension over the API. A real c-shared add-in (testdata/uiaddin), wired through the
// exact production glue (SetHost→router, events.Subscribe→Notify, a drained
// Dispatcher), (1) registers a ribbon button on Activate via commands.create, then
// (2) when that button is "clicked" (session.Execute), reacts to the forwarded
// command.ended event by creating a document through the host API — Inventor's
// ButtonDefinition + OnExecute, across the two-Go-runtime boundary.
func TestAddInExtendsRibbonAndActsOnClick(t *testing.T) {
	so := buildFixture(t, "uiaddin")

	s := app.NewSession()
	// The fixture creates its button with no explicit ribbon, which defaults to the
	// Part ribbon (CommandDefinition.appearsOnRibbon). Open an active part document so
	// BuildRibbon yields the Part ribbon — this mirrors how an add-in button is used in
	// practice and the core router test's seededSession setup. Without it the session is
	// on the ZeroDoc ribbon and the button is correctly excluded.
	pd, err := s.Workspace().Add(doc.Part, "test.obk", true)
	if err != nil {
		t.Fatalf("add part document: %v", err)
	}
	pd.SetContent(compdef.NewPartComponentDefinition())

	d := dispatch.New(8)
	defer d.Close()
	rtr := router.New(opregistry.Default())
	SetHost(d, func(method string, req []byte) ([]byte, error) {
		return rtr.Handle(s, method, req)
	}, 2*time.Second)

	libs, err := LoadDir(filepath.Dir(so))
	if err != nil || len(libs) != 1 {
		t.Fatalf("LoadDir: libs=%d err=%v", len(libs), err)
	}
	addin := libs[0]
	defer addin.Close()

	// Forward session events to the add-in over the C ABI (the production wiring).
	subs := events.Subscribe(s, func(ev []byte) { _ = addin.Notify(ev) })
	defer func() {
		for _, sub := range subs {
			sub.Cancel()
		}
	}()

	// Activate blocks on the add-in's commands.create host call, so drain concurrently.
	activated := make(chan error, 1)
	go func() { activated <- addin.Activate(s) }()
	if err := drainUntil(t, d, activated); err != nil {
		t.Fatalf("Activate (the add-in's commands.create host call): %v", err)
	}

	// (1) The UI is extended: the add-in's button is on the ribbon, styled as asked.
	panel, ok := app.BuildRibbon(s).Panel("Demo")
	if !ok || len(panel.Buttons) != 1 || panel.Buttons[0].Command.ID() != "AddIn.Ping" {
		t.Fatalf("add-in did not add its ribbon button: panel=%+v ok=%v", panel, ok)
	}

	// (2) "Click" the button. Execute fires command.ended → the add-in's Notify → it
	// queues a documents.create host call we drain until the new document appears.
	if err := s.Execute("AddIn.Ping"); err != nil {
		t.Fatalf("execute add-in command: %v", err)
	}
	if !drainUntilDocument(t, d, s, "FromAddIn") {
		t.Fatal("add-in did not create its document in response to the button click")
	}
}

// drainUntil drains d on this goroutine until done yields (returning its value) or a
// deadline passes. Draining here keeps all model access on one goroutine.
func drainUntil(t *testing.T, d *dispatch.Dispatcher, done <-chan error) error {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		d.Drain(0)
		select {
		case err := <-done:
			return err
		case <-deadline:
			t.Fatal("timed out draining the dispatcher")
			return nil
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

// drainUntilDocument drains d until a document named name is open, or a deadline passes.
func drainUntilDocument(t *testing.T, d *dispatch.Dispatcher, s *app.Session, name string) bool {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		d.Drain(0)
		for _, doc := range s.Workspace().Documents() {
			if doc.DisplayName() == name {
				return true
			}
		}
		select {
		case <-deadline:
			return false
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
