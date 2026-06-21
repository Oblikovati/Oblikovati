// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"oblikovati.org/addin/dispatch"
	"oblikovati.org/addin/events"
	"oblikovati.org/addin/opregistry"
	"oblikovati.org/addin/router"
	"oblikovati.org/addincat"
	"oblikovati.org/app"
	"oblikovati.org/event"
	"oblikovati.org/head/internal/addinhost"
	"oblikovati.org/persistence/addinstate"
	"oblikovati.org/persistence/dialogmemory"
	"oblikovati.org/script/bridge"
	"oblikovati.org/script/console"
	"oblikovati.org/script/gopherlua"
	"oblikovati.org/script/runner"
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
	dir        string              // watched add-ins directory
	watchDone  chan struct{}       // closed by stop() to end the watcher goroutine
	changed    int32               // set by the watcher when a library is replaced on disk
	script     *console.Controller // the Script Console runtime (Lua over the dispatched router)
	methods    func() []string     // host wire-method names, for the editor's autocomplete
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

	dir := addInsDir()
	h := &addInHost{dispatcher: d, dir: dir, watchDone: make(chan struct{})}
	// The Script Console runs Lua over the SAME router + dispatcher add-ins use, so its
	// host calls serialize onto the session goroutine and never freeze the UI (ADR-0028 §5).
	h.script = newScriptController(rtr, session, d)
	h.methods = rtr.Methods // the editor's autocomplete walks this method set
	useBehaviorStore(session)
	useDialogMemoryStore(session)
	h.loadAndRegister(session, dir)
	h.loadInstalledAddIns(session) // add-ins the in-app catalogue installed under the per-user dir (#1164)
	h.subs = events.Subscribe(session, func(ev []byte) { h.notifyActive(session, ev) })
	// Under a supervisor (make run-watch sets OBK_ADDIN_AUTORESTART=1), watch the
	// add-ins dir so a rebuilt library makes the app exit-and-relaunch — the safe way
	// to pick up a new add-in (a Go c-shared cannot be hot-swapped in-process). Plain
	// `make run` never auto-exits.
	if os.Getenv("OBK_ADDIN_AUTORESTART") == "1" {
		go h.watchAddIns()
	}
	return h
}

// loadAndRegister loads every shared-library add-in from dir, surfaces any refused on
// API-version grounds in the status bar, and registers (and per stored behavior,
// activates) the loadable ones. A bad add-in is logged and skipped — never fatal.
func (h *addInHost) loadAndRegister(session *app.Session, dir string) {
	libs, skipped, err := addinhost.LoadDir(dir)
	h.registerResults(session, libs, skipped, err)
}

// loadInstalledAddIns loads add-ins the in-app catalogue installed under the per-user
// directory (#1164), in addition to those beside the executable. A relocated or
// unresolvable directory is logged and skipped — never fatal.
func (h *addInHost) loadInstalledAddIns(session *app.Session) {
	dir, err := addincat.UserAddInsDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "add-ins: locate user add-ins dir: %v\n", err)
		return
	}
	libs, skipped, err := addinhost.LoadInstalledTree(dir)
	h.registerResults(session, libs, skipped, err)
}

// registerResults surfaces a load's error and version-skips, then registers (and per stored
// behavior, activates) each loadable add-in. Shared by the flat exe-adjacent directory and
// the per-user install tree so both report and register identically.
func (h *addInHost) registerResults(session *app.Session, libs []*addinhost.LoadedAddIn, skipped []addinhost.IncompatibleAddIn, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "add-ins: %v\n", err)
	}
	// A version-skipped add-in is the usual cause of "the bridge port never opened": surface each one
	// on stderr (not only the UI status bar) so it is catchable in the process log / headless runs.
	for _, sk := range skipped {
		fmt.Fprintf(os.Stderr, "add-in %q skipped (incompatible): %s\n", sk.ID, sk.Reason)
	}
	if notice := incompatibleNotice(skipped); notice != "" {
		session.SetNotice(notice)
	}
	for _, lib := range libs {
		if err := registerAndMaybeActivate(session, lib); err != nil {
			fmt.Fprintf(os.Stderr, "add-in %q: %v\n", lib.ID(), err)
			continue
		}
		h.loaded = append(h.loaded, lib)
	}
}

// incompatibleNotice is the status-bar message for add-ins LoadDir refused on version
// grounds — so a skipped add-in is never silent. Empty when nothing was skipped.
func incompatibleNotice(skipped []addinhost.IncompatibleAddIn) string {
	if len(skipped) == 0 {
		return ""
	}
	if len(skipped) == 1 {
		return fmt.Sprintf("Add-in %q skipped: incompatible API version (%s)", skipped[0].ID, skipped[0].Reason)
	}
	ids := make([]string, len(skipped))
	for i, s := range skipped {
		ids[i] = s.ID
	}
	return fmt.Sprintf("%d add-ins skipped: incompatible API version (%s)", len(skipped), strings.Join(ids, ", "))
}

// newScriptController builds the Script Console runtime: a gopher-lua engine whose host
// calls are marshalled onto the session goroutine via the dispatched caller (the same
// dispatcher the frame loop drains), under the GUI resource limits. oblikovati.methods()
// is backed by the router's registered method list for discoverability (ADR-0028).
func newScriptController(rtr *router.Router, session *app.Session, d *dispatch.Dispatcher) *console.Controller {
	caller := bridge.NewDispatchedCaller(rtr.Handle, session, d, context.Background())
	run := runner.New(gopherlua.New(), caller, rtr.Methods)
	return console.NewController(run, runner.DefaultGUILimits)
}

// useBehaviorStore wires the per-user load-behavior preferences into the registry,
// so a demand/disabled add-in registers (it lists in addins.list) but is not
// activated at startup (M05-F01, #251). A store failure costs only persistence —
// the session still runs with the default behaviors.
func useBehaviorStore(session *app.Session) {
	path, err := addinstate.DefaultPath()
	if err == nil {
		err = session.AddIns().UseBehaviorStore(addinstate.NewFileStore(path))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "add-in behaviors: %v\n", err)
	}
}

// useDialogMemoryStore wires the per-user remembered dialog choices (suppressed
// balloon tips, remembered prompt answers — M05-F09). A failure costs only
// persistence.
func useDialogMemoryStore(session *app.Session) {
	path, err := dialogmemory.DefaultPath()
	if err == nil {
		err = session.UseDialogMemoryStore(dialogmemory.NewFileStore(path))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "dialog memory: %v\n", err)
	}
}

// registerAndMaybeActivate registers one loaded add-in, activating it only when its
// stored load behavior says so; demand/disabled entries wait for addins.activate.
func registerAndMaybeActivate(session *app.Session, lib *addinhost.LoadedAddIn) error {
	if err := session.AddIns().Register(lib); err != nil {
		return err
	}
	if session.AddIns().LoadBehavior(lib.ID()) != app.LoadOnStartup {
		return nil
	}
	return session.AddIns().Activate(session, lib.ID())
}

// notifyActive forwards a serialized event to every ACTIVE add-in — a registered
// but deactivated add-in must not keep observing the session.
func (h *addInHost) notifyActive(session *app.Session, ev []byte) {
	for _, lib := range h.loaded {
		if session.AddIns().IsActive(lib.ID()) {
			_ = lib.Notify(ev)
		}
	}
}

// drain runs pending add-in calls on the (session) goroutine. Call once per frame.
func (h *addInHost) drain() { h.dispatcher.Drain(addInDrainPerFrame) }

// stop deactivates the add-ins and closes the dispatcher. It does NOT dlclose the
// libraries: a Go c-shared keeps runtime threads (sysmon/GC) alive, and unmapping its
// code with dlclose crashes the host. The process is exiting, so the OS reclaims them.
func (h *addInHost) stop(session *app.Session) {
	close(h.watchDone)
	for _, s := range h.subs {
		s.Cancel()
	}
	for _, lib := range h.loaded {
		_ = session.AddIns().Deactivate(session, lib.ID())
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
