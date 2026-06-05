// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Oblikovati/oblikovati/addin/dispatch"
	"github.com/Oblikovati/oblikovati/addin/events"
	"github.com/Oblikovati/oblikovati/addin/opregistry"
	"github.com/Oblikovati/oblikovati/addin/router"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/event"
	"github.com/Oblikovati/oblikovati/head/internal/addinhost"
)

// addInDrainPerFrame bounds how many queued add-in calls run per frame, so a burst
// of MCP requests can't starve rendering.
const addInDrainPerFrame = 32

// hostCallTimeout caps how long an add-in's host call waits to be drained. The frame
// loop drains every frame, so normal latency is ~one frame; the timeout only guards
// a stalled loop.
const hostCallTimeout = 10 * time.Second

// addInHost owns the runtime wiring for shared-library add-ins: the dispatch queue
// the frame loop drains, the loaded libraries, and the event subscriptions.
type addInHost struct {
	dispatcher *dispatch.Dispatcher
	loaded     []*addinhost.LoadedAddIn
	subs       []event.Subscription
}

// startAddIns wires the add-in subsystem: it installs the router as the host call
// handler, loads every shared-library add-in from the add-ins directory, activates
// them, and forwards session events to them. It never fails the app — a bad or
// missing add-in is logged and skipped — and always returns a host whose dispatcher
// the frame loop can drain.
func startAddIns(session *app.Session) *addInHost {
	d := dispatch.New(64)
	rtr := router.New(opregistry.Default())
	// Route kernel logs into the router's operation trace so a driver can read both the
	// op-trace and any slog records over the bridge (logs.tail / tail_logs).
	slog.SetDefault(slog.New(rtr.Trace().SlogHandler(slog.LevelInfo)))
	addinhost.SetHost(d, func(method string, req []byte) ([]byte, error) {
		return rtr.Handle(session, method, req)
	}, hostCallTimeout)

	h := &addInHost{dispatcher: d}
	libs, err := addinhost.LoadDir(addInsDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "add-ins: %v\n", err)
	}
	for _, lib := range libs {
		if err := activate(session, lib); err != nil {
			fmt.Fprintf(os.Stderr, "add-in %q: %v\n", lib.ID(), err)
			continue
		}
		h.loaded = append(h.loaded, lib)
	}
	if len(h.loaded) > 0 {
		h.subs = events.Subscribe(session, h.notifyAll)
	}
	return h
}

// activate registers and activates one loaded add-in.
func activate(session *app.Session, lib *addinhost.LoadedAddIn) error {
	if err := session.AddIns().Register(lib); err != nil {
		return err
	}
	return session.AddIns().Activate(session, lib.ID())
}

// notifyAll forwards a serialized event to every active add-in.
func (h *addInHost) notifyAll(ev []byte) {
	for _, lib := range h.loaded {
		_ = lib.Notify(ev)
	}
}

// drain runs pending add-in calls on the (session) goroutine. Call once per frame.
func (h *addInHost) drain() { h.dispatcher.Drain(addInDrainPerFrame) }

// stop deactivates and unloads the add-ins and closes the dispatcher.
func (h *addInHost) stop(session *app.Session) {
	for _, s := range h.subs {
		s.Cancel()
	}
	for _, lib := range h.loaded {
		_ = session.AddIns().Deactivate(session, lib.ID())
		_ = lib.Close()
	}
	h.dispatcher.Close()
}

// addInsDir is the directory scanned for shared-library add-ins: OBK_ADDINS_DIR if
// set, else an "addins" folder beside the executable.
func addInsDir() string {
	if dir := os.Getenv("OBK_ADDINS_DIR"); dir != "" {
		return dir
	}
	exe, err := os.Executable()
	if err != nil {
		return "addins"
	}
	return filepath.Join(filepath.Dir(exe), "addins")
}
