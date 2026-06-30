<!-- SPDX-License-Identifier: GPL-2.0-only -->
# FEM Parity — Phase 0a-host (GPL host implementation) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement, in the GPL `Oblikovati` host, the three public-API methods and two push events shipped in `oblikovati.org/api` v0.101.0 — so the new `referenceList` control and the modal `TaskPanelSpec` actually render and round-trip to add-ins.

**Architecture:** Three layers. (1) **App/session** (pure Go, TDD-unit-tested): event types, the `referenceList` store mutation + selection capture, and a task-panel store with an async resolve→event path modeled on the existing file-dialog precedent. (2) **Router + event-relay** (pure Go, TDD): three `wire.Method*` handlers + two event subscriptions. (3) **head/ui** (cgo/ImGui — build + live-MCP validated, not unit-tested): render the `referenceList` widget and a true input-blocking modal task panel, which requires one new cgo binding (`igBeginPopupModal`). The new `wire` DTOs are already visible to the host via the go.work-resolved local `Oblikovati.API` sibling (on `develop`, which carries v0.101.0).

**Tech Stack:** Go; cgo + Dear ImGui (in `head/internal/native`); the project `event.Bus`; `oblikovati.org/api` v0.101.0 wire/types.

## Global Constraints

- Every new `.go`/`.cpp`/`.h` file carries `// SPDX-License-Identifier: GPL-2.0-only` (run `scripts/add-spdx-headers.py` or include it inline; the code blocks below already do).
- Never re-declare a DTO or method-name string — import from `oblikovati.org/api/wire`. The host implements the contract; it does not redefine it.
- Code style (CLAUDE.md): functions 4–20 lines, files <500 lines, explicit types (no `any`), early returns, max 2 indentation levels, exception/error messages include the offending value and expected shape.
- TDD for the app/router/event-relay layers (layers 1–2): failing test first, watched fail. The `head/ui` + cgo layer (layer 3) cannot be Go-unit-tested (ImGui immediate-mode draw calls need a live frame); those tasks are validated by `go build ./...` (compile) + the live-MCP task (H10). This is an explicit, bounded deviation — every behavioral seam that CAN be unit-tested (the store mutations, event emission, router decode, selection filtering) IS, so the head layer is thin glue over already-tested session methods.
- Run `gofmt -l` and `golangci-lint run` on touched Go packages before each commit; run the host build `go build ./...` after the cgo task.
- New host events reuse the existing relay mechanism (`addin/events`), the existing `event.Bus`, and the existing C-ABI Notify sink — no new transport.
- Branch: `feature/fem-parity-phase-0a-host` (already created off `develop`). Do NOT `git add -A` (an untracked `motorshot` throwaway exists; stage only the files each task names).

## Key existing anchors (verified)

- Router register + handler + helpers: `addin/router/ui_surfaces.go:14` (`registerUISurfaceHandlers`, `r.readOnly(wire.Method*, handler)`), `:80` (`setDockableWindowValue` template), `decode(args,&req)` + `ok()` helpers.
- Dock store + emit: `app/dockwindow_store.go:99` (`PanelValueChanged` → `setControlValue` + `event.Emit(s.bus, event.After, …)`).
- Event TypeIDs: `app/events.go:30` last M05-F03 id is `tidPanelValueChanged = 0x0513`; next free `0x0514`, `0x0515`. `PanelValueChanged` struct + `EventID()` at `:75`.
- Relay: `addin/events/events.go:302` (`relayJSON` generic union), `:78` (`subscribeSessionUI` entry mirroring).
- Modal async precedent: `app/dialog_requests.go:66` (`RequestFileDialog` queue) + `:88` (`ResolveFileDialog` → `event.Emit(... FileDialogChosen{})`).
- Selection refs: `app/selection.go:318` (`(*Selection).References() []string`), format `"face/"+base64.RawURLEncoding(key)` / `"vertex/"+…` (`model/feature/work_surface_ref.go:57`).
- head render: `head/ui/addin_panels.go:136` (`drawAddInPanelControl`, `PushIDInt(index)`), `:164` (`drawEditableControl` switch), `drawControlList` (`:110`); containers `head/ui/addin_grid.go:119` (`drawPanelContainer`).
- head frame hook: `head/ui/chrome.go:46` (`drawChromeWindows`) → `:155` (`drawAddInPanels`).
- native ImGui binding: Go wrappers `head/internal/native/imgui.go` (decls `:62–66`, wrappers `:842–870`: `BeginChild/EndChild`, `Selectable`, `BeginPopupContextItem`, `OpenPopup`, `BeginPopup`, `EndPopup`, `CloseCurrentPopup`, `Button`, `CenterNextWindow`, `SetNextWindowSize`); C++ impl `head/internal/native/imgui_wrap.cpp` (`:40` `obk_ig_begin_closable` = the `int*open` bool-out template; `:212–215` popup family). **`BeginPopupModal` is NOT wrapped — task H8 adds it.**

---

### Task H1: app event types `PanelReferencesChanged` + `TaskPanelClosed`

**Files:**
- Modify: `app/events.go` (two `tid*` consts in the M05-F03 block ~line 30; two event structs + `EventID()` after `PanelValueChanged` ~line 82)
- Test: `app/events_phase0a_test.go` (create)

**Interfaces:**
- Produces: `app.PanelReferencesChanged{WindowID, ControlID string; Refs []string; Action string}` with `EventID() == tidPanelReferencesChanged (0x0514)`; `app.TaskPanelClosed{ID string; Accepted bool}` with `EventID() == tidTaskPanelClosed (0x0515)`.

- [ ] **Step 1: Write the failing test**

Create `app/events_phase0a_test.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestPhase0aEventIDsDistinct(t *testing.T) {
	got := map[string]uint32{
		"refs":   uint32(PanelReferencesChanged{}.EventID()),
		"closed": uint32(TaskPanelClosed{}.EventID()),
		"value":  uint32(PanelValueChanged{}.EventID()),
	}
	if got["refs"] != 0x0514 {
		t.Fatalf("PanelReferencesChanged EventID = %#x, want 0x0514", got["refs"])
	}
	if got["closed"] != 0x0515 {
		t.Fatalf("TaskPanelClosed EventID = %#x, want 0x0515", got["closed"])
	}
	if got["refs"] == got["value"] || got["closed"] == got["value"] || got["refs"] == got["closed"] {
		t.Fatalf("event ids collide: %#v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/vmiguel/git/oblikovati-workspace/Oblikovati && go test ./app/ -run TestPhase0aEventIDsDistinct`
Expected: FAIL — `undefined: PanelReferencesChanged`.

- [ ] **Step 3: Write minimal implementation**

In `app/events.go`, add to the `const (...)` TypeID block immediately after `tidPanelValueChanged ... = 0x0513`:
```go
	tidPanelReferencesChanged event.TypeID = 0x0514 // app/dockwindow_store.go (M05-F03 referenceList)
	tidTaskPanelClosed        event.TypeID = 0x0515 // app/taskpanel_store.go (M05-F03 task panels)
```
After the `PanelValueChanged` struct + its `EventID()` method (~line 82), add:
```go
// PanelReferencesChanged reports a referenceList control's row set changing (by the user's
// Add-from-selection / per-row Remove, or by an add-in's dockableWindows.setReferences).
// Refs is the full new set; Action is "add"/"remove"/"set" for diagnostics.
type PanelReferencesChanged struct {
	WindowID  string
	ControlID string
	Refs      []string
	Action    string
}

func (PanelReferencesChanged) EventID() event.TypeID { return tidPanelReferencesChanged }

// TaskPanelClosed reports the user accepting (OK) or cancelling a modal add-in task panel.
type TaskPanelClosed struct {
	ID       string
	Accepted bool
}

func (TaskPanelClosed) EventID() event.TypeID { return tidTaskPanelClosed }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./app/ -run TestPhase0aEventIDsDistinct` → PASS.

- [ ] **Step 5: Commit**
```bash
git add app/events.go app/events_phase0a_test.go
git commit -m "feat(app): PanelReferencesChanged + TaskPanelClosed event types"
```

---

### Task H2: `Session.SetDockableWindowReferences` + `setControlRefs`

**Files:**
- Modify: `app/dockwindow_store.go` (add `SetDockableWindowReferences` mirroring `PanelValueChanged:99`; add `setControlRefs` mirroring `setControlValue`)
- Test: `app/dockwindow_references_test.go` (create)

**Interfaces:**
- Consumes: `app.PanelReferencesChanged` (H1); the existing `s.dockableWindows.windows map[string]wire.DockableWindowSpec` and `setControlValue` recursion.
- Produces: `(*Session).SetDockableWindowReferences(windowID, controlID string, refs []string)` — sets the matching `referenceList` control's `Rows` (one `wire.PanelReferenceRow{Ref: r}` per ref, Label empty for host derivation) and emits `PanelReferencesChanged{..., Action: "set"}`; `setControlRefs(controls []wire.PanelControlSpec, controlID string, refs []string) bool` (recurses Children, returns true if found+updated).

- [ ] **Step 1: Write the failing test**

Create `app/dockwindow_references_test.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

func TestSetDockableWindowReferencesUpdatesAndEmits(t *testing.T) {
	s := newTestSession(t)
	s.SetDockableWindow(wire.DockableWindowSpec{
		ID: "w", Title: "W", Visible: true,
		Controls: []wire.PanelControlSpec{{Kind: types.PanelReferenceList, ID: "faces"}},
	})
	var got app.PanelReferencesChanged
	_ = got // see assertion via captured below
	captured := captureEvent[PanelReferencesChanged](t, s)

	s.SetDockableWindowReferences("w", "faces", []string{"face/aaa", "face/bbb"})

	ev := captured()
	if ev.WindowID != "w" || ev.ControlID != "faces" || len(ev.Refs) != 2 || ev.Refs[1] != "face/bbb" {
		t.Fatalf("event = %+v, want windowID=w controlID=faces refs=[face/aaa face/bbb]", ev)
	}
	stored := s.DockableWindows().List()[0].Controls[0]
	if len(stored.Rows) != 2 || stored.Rows[0].Ref != "face/aaa" {
		t.Fatalf("stored rows = %+v, want two rows starting face/aaa", stored.Rows)
	}
}
```
NOTE for the implementer: this test uses two helpers you must locate or add in the `app` test package — `newTestSession(t)` (find the existing session test constructor used by `dockwindow_store_test.go`; reuse it) and a generic `captureEvent[E](t, s)` that subscribes to `s`'s bus for event `E` and returns a getter. If `dockwindow_store_test.go` already has a session+event-capture harness, mirror it exactly instead of these names; adjust the test to the existing harness. Remove the stray `var got` line — it is a placeholder reminder, not real code.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/ -run TestSetDockableWindowReferencesUpdatesAndEmits`
Expected: FAIL — `undefined: (*Session).SetDockableWindowReferences` (or harness-name errors you resolve per the note).

- [ ] **Step 3: Write minimal implementation**

In `app/dockwindow_store.go`, after `PanelValueChanged` (~line 107) add:
```go
// SetDockableWindowReferences replaces a referenceList control's rows with refs (one row per ref,
// Label left empty for host derivation) and notifies the owning add-in. Mirrors PanelValueChanged.
func (s *Session) SetDockableWindowReferences(windowID, controlID string, refs []string) {
	if spec, ok := s.dockableWindows.windows[windowID]; ok {
		if setControlRefs(spec.Controls, controlID, refs) {
			s.dockableWindows.windows[windowID] = spec
		}
	}
	event.Emit(s.bus, event.After, PanelReferencesChanged{
		WindowID: windowID, ControlID: controlID, Refs: refs, Action: "set",
	})
}

// setControlRefs finds the control by id (recursing containers) and replaces its Rows. Returns
// true when the control was found and updated. Mirrors setControlValue's recursion.
func setControlRefs(controls []wire.PanelControlSpec, controlID string, refs []string) bool {
	for i := range controls {
		if controls[i].ID == controlID {
			controls[i].Rows = rowsFromRefs(refs)
			return true
		}
		if setControlRefs(controls[i].Children, controlID, refs) {
			return true
		}
	}
	return false
}

func rowsFromRefs(refs []string) []wire.PanelReferenceRow {
	rows := make([]wire.PanelReferenceRow, len(refs))
	for i, r := range refs {
		rows[i] = wire.PanelReferenceRow{Ref: r}
	}
	return rows
}
```
(If `setControlValue` mutates a copied slice rather than in place, mirror exactly whatever pointer/index discipline it uses so the stored spec actually changes — verify against `setControlValue` in the same file.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./app/ -run TestSetDockableWindowReferencesUpdatesAndEmits` → PASS. Then `go test ./app/` → PASS.

- [ ] **Step 5: Commit**
```bash
git add app/dockwindow_store.go app/dockwindow_references_test.go
git commit -m "feat(app): SetDockableWindowReferences updates referenceList rows + emits event"
```

---

### Task H3: `Session.AddReferencesFromSelection`

**Files:**
- Modify: `app/dockwindow_store.go` (add `AddReferencesFromSelection`); add a small `selectionRefKind` helper (new file `app/reference_kind.go`)
- Test: `app/reference_kind_test.go` (create)

**Interfaces:**
- Consumes: `(*Selection).References()` (`app/selection.go:318`) via the Session's selection accessor (find how `app` reads the live selection — e.g. `s.selection.References()` or `s.Selection().References()`; use whichever exists), and `SetDockableWindowReferences` (H2).
- Produces: `(*Session).AddReferencesFromSelection(windowID, controlID string, accepts []string)` — reads current selection refs, keeps those whose kind ∈ accepts (empty accepts = any), appends to the control's existing rows (dedup), and calls `SetDockableWindowReferences`; `selectionRefKind(ref string) string` returns the kind prefix (`"face"`/`"edge"`/`"vertex"`) of a `"<kind>/<b64>"` ref.

- [ ] **Step 1: Write the failing test** (kind filter is the pure, fully-testable core)

Create `app/reference_kind_test.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"reflect"
	"testing"
)

func TestSelectionRefKind(t *testing.T) {
	cases := map[string]string{
		"face/abc": "face", "edge/xy": "edge", "vertex/z": "vertex", "garbage": "",
	}
	for ref, want := range cases {
		if got := selectionRefKind(ref); got != want {
			t.Fatalf("selectionRefKind(%q) = %q, want %q", ref, got, want)
		}
	}
}

func TestFilterRefsByAccepts(t *testing.T) {
	refs := []string{"face/a", "edge/b", "vertex/c", "face/d"}
	got := filterRefsByAccepts(refs, []string{"face"})
	if !reflect.DeepEqual(got, []string{"face/a", "face/d"}) {
		t.Fatalf("filter = %v, want [face/a face/d]", got)
	}
	if all := filterRefsByAccepts(refs, nil); !reflect.DeepEqual(all, refs) {
		t.Fatalf("empty accepts should keep all, got %v", all)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/ -run 'TestSelectionRefKind|TestFilterRefsByAccepts'`
Expected: FAIL — `undefined: selectionRefKind`.

- [ ] **Step 3: Write minimal implementation**

Create `app/reference_kind.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"slices"
	"strings"
)

// selectionRefKind returns the kind prefix ("face"/"edge"/"vertex") of a selection reference
// string formatted "<kind>/<base64key>" (see model/feature/work_surface_ref.go); "" if malformed.
func selectionRefKind(ref string) string {
	i := strings.IndexByte(ref, '/')
	if i <= 0 {
		return ""
	}
	return ref[:i]
}

// filterRefsByAccepts keeps refs whose kind is in accepts; an empty accepts keeps all.
func filterRefsByAccepts(refs, accepts []string) []string {
	if len(accepts) == 0 {
		return refs
	}
	kept := make([]string, 0, len(refs))
	for _, r := range refs {
		if slices.Contains(accepts, selectionRefKind(r)) {
			kept = append(kept, r)
		}
	}
	return kept
}
```
In `app/dockwindow_store.go`, add (locate the live-selection accessor first — grep `func (s *Session)` for a selection getter, e.g. `s.selection`):
```go
// AddReferencesFromSelection appends the current viewport selection (filtered by accepts) to a
// referenceList control's existing rows, de-duplicated, then notifies the add-in. Empty accepts = any.
func (s *Session) AddReferencesFromSelection(windowID, controlID string, accepts []string) {
	picked := filterRefsByAccepts(s.selection.References(), accepts)
	merged := mergeControlRefs(s.dockableWindows.windows[windowID].Controls, controlID, picked)
	s.SetDockableWindowReferences(windowID, controlID, merged)
}

// mergeControlRefs returns the control's existing row refs plus the new picks, order-preserving,
// without duplicates.
func mergeControlRefs(controls []wire.PanelControlSpec, controlID string, picks []string) []string {
	out := controlRefs(controls, controlID)
	for _, p := range picks {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// controlRefs returns the current row refs of the control (recursing containers); nil if absent.
func controlRefs(controls []wire.PanelControlSpec, controlID string) []string {
	for i := range controls {
		if controls[i].ID == controlID {
			refs := make([]string, len(controls[i].Rows))
			for j, row := range controls[i].Rows {
				refs[j] = row.Ref
			}
			return refs
		}
		if r := controlRefs(controls[i].Children, controlID); r != nil {
			return r
		}
	}
	return nil
}
```
(Add `"slices"` to the `dockwindow_store.go` imports. If the Session's selection field/getter differs from `s.selection`, use the real one — confirm by reading the struct.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./app/ -run 'TestSelectionRefKind|TestFilterRefsByAccepts'` → PASS. Then `go test ./app/` → PASS.

- [ ] **Step 5: Commit**
```bash
git add app/reference_kind.go app/reference_kind_test.go app/dockwindow_store.go
git commit -m "feat(app): AddReferencesFromSelection (kind-filtered, deduped) for referenceList"
```

---

### Task H4: task-panel store (`ShowTaskPanel` / `CloseTaskPanel` / `ResolveTaskPanel` / `List`)

**Files:**
- Create: `app/taskpanel_store.go`
- Modify: the `Session` struct + its constructor (wherever `s.dockableWindows` is initialized) to add and init a `taskPanels *AddInTaskPanels`
- Test: `app/taskpanel_store_test.go` (create)

**Interfaces:**
- Consumes: `app.TaskPanelClosed` (H1); `event.Emit`; the modal-async precedent (`RequestFileDialog`/`ResolveFileDialog`, `app/dialog_requests.go`); `validateControlTree` if it exists in the dock store (reuse for control validation; else skip).
- Produces: `(*Session).ShowTaskPanel(spec wire.TaskPanelSpec) error` (validates non-empty ID/Title, stores; replaces on same ID), `(*Session).CloseTaskPanel(id string) error` (programmatic remove, NO event — the add-in initiated it), `(*Session).ResolveTaskPanel(id string, accepted bool) error` (user OK/Cancel: remove + emit `TaskPanelClosed`), `(*Session).TaskPanels() *AddInTaskPanels` with `List() []wire.TaskPanelSpec` (creation order).

- [ ] **Step 1: Write the failing test**

Create `app/taskpanel_store_test.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestShowListResolveTaskPanel(t *testing.T) {
	s := newTestSession(t)
	if err := s.ShowTaskPanel(wire.TaskPanelSpec{ID: "fix", Title: "Fixed"}); err != nil {
		t.Fatalf("ShowTaskPanel: %v", err)
	}
	if l := s.TaskPanels().List(); len(l) != 1 || l[0].ID != "fix" {
		t.Fatalf("List = %+v, want one panel fix", l)
	}
	captured := captureEvent[TaskPanelClosed](t, s)
	if err := s.ResolveTaskPanel("fix", true); err != nil {
		t.Fatalf("ResolveTaskPanel: %v", err)
	}
	if ev := captured(); ev.ID != "fix" || !ev.Accepted {
		t.Fatalf("closed event = %+v, want fix accepted=true", ev)
	}
	if l := s.TaskPanels().List(); len(l) != 0 {
		t.Fatalf("panel not removed after resolve: %+v", l)
	}
}

func TestShowTaskPanelRejectsEmptyID(t *testing.T) {
	s := newTestSession(t)
	if err := s.ShowTaskPanel(wire.TaskPanelSpec{Title: "x"}); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestCloseTaskPanelIsSilent(t *testing.T) {
	s := newTestSession(t)
	_ = s.ShowTaskPanel(wire.TaskPanelSpec{ID: "p", Title: "P"})
	if err := s.CloseTaskPanel("p"); err != nil {
		t.Fatalf("CloseTaskPanel: %v", err)
	}
	if err := s.CloseTaskPanel("missing"); err == nil {
		t.Fatal("expected error closing unknown panel")
	}
}
```
(Use the same `newTestSession`/`captureEvent` harness as H2; align names to the real harness.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/ -run TaskPanel`
Expected: FAIL — `undefined: (*Session).ShowTaskPanel`.

- [ ] **Step 3: Write minimal implementation**

Create `app/taskpanel_store.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/event" // adjust to the project's event package import path
	"oblikovati.org/api/wire"
)

// AddInTaskPanels holds the open modal task panels an add-in asked the host to show. The head
// renders each modally (BeginPopupModal) and calls ResolveTaskPanel on OK/Cancel.
type AddInTaskPanels struct {
	order  []string
	panels map[string]wire.TaskPanelSpec
}

func newAddInTaskPanels() *AddInTaskPanels {
	return &AddInTaskPanels{panels: map[string]wire.TaskPanelSpec{}}
}

// List returns the open task panels in creation order.
func (p *AddInTaskPanels) List() []wire.TaskPanelSpec {
	out := make([]wire.TaskPanelSpec, 0, len(p.order))
	for _, id := range p.order {
		out = append(out, p.panels[id])
	}
	return out
}

// TaskPanels returns the add-in modal-task-panel store.
func (s *Session) TaskPanels() *AddInTaskPanels { return s.taskPanels }

// ShowTaskPanel stores a modal task panel for the head to display; replaces one with the same ID.
func (s *Session) ShowTaskPanel(spec wire.TaskPanelSpec) error {
	if spec.ID == "" || spec.Title == "" {
		return fmt.Errorf("app: task panel needs id and title, got id=%q title=%q", spec.ID, spec.Title)
	}
	if _, exists := s.taskPanels.panels[spec.ID]; !exists {
		s.taskPanels.order = append(s.taskPanels.order, spec.ID)
	}
	s.taskPanels.panels[spec.ID] = spec
	return nil
}

// CloseTaskPanel removes a task panel programmatically (add-in-initiated); emits no event.
func (s *Session) CloseTaskPanel(id string) error {
	if _, ok := s.taskPanels.panels[id]; !ok {
		return fmt.Errorf("app: no task panel %q to close", id)
	}
	s.removeTaskPanel(id)
	return nil
}

// ResolveTaskPanel records the user's OK/Cancel: removes the panel and notifies the add-in.
func (s *Session) ResolveTaskPanel(id string, accepted bool) error {
	if _, ok := s.taskPanels.panels[id]; !ok {
		return fmt.Errorf("app: no task panel %q to resolve", id)
	}
	s.removeTaskPanel(id)
	event.Emit(s.bus, event.After, TaskPanelClosed{ID: id, Accepted: accepted})
	return nil
}

func (s *Session) removeTaskPanel(id string) {
	delete(s.taskPanels.panels, id)
	for i, oid := range s.taskPanels.order {
		if oid == id {
			s.taskPanels.order = append(s.taskPanels.order[:i], s.taskPanels.order[i+1:]...)
			break
		}
	}
}
```
IMPORTANT: fix the two imports to the project's real paths — find how `app/dockwindow_store.go` imports the `event` package (it is NOT `oblikovati.org/api/event`; copy the exact import from that file) and keep `wire`. Then add the field to the `Session` struct and initialize it in the constructor next to `dockableWindows`:
```go
	taskPanels *AddInTaskPanels   // struct field, beside dockableWindows
```
```go
	taskPanels: newAddInTaskPanels(),   // in the Session constructor, beside the dockableWindows init
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./app/ -run TaskPanel` → PASS. Then `go test ./app/` → PASS.

- [ ] **Step 5: Commit**
```bash
git add app/taskpanel_store.go app/taskpanel_store_test.go app/<session-struct-file>.go
git commit -m "feat(app): modal task-panel store (Show/Close/Resolve/List) with async close event"
```

---

### Task H5: relay the two events to add-ins

**Files:**
- Modify: `addin/events/events.go` (extend the `relayJSON` union ~line 302; add two `subscribeSessionUI` entries ~line 78)
- Test: `addin/events/events_phase0a_test.go` (create)

**Interfaces:**
- Consumes: `app.PanelReferencesChanged`, `app.TaskPanelClosed` (H1); `wire.PanelReferencesChangedEvent`, `wire.TaskPanelClosedEvent`, `wire.EventPanelReferencesChanged`, `wire.EventTaskPanelClosed` (api v0.101.0); the `relayJSON` + `Sink` machinery.
- Produces: emitting `app.PanelReferencesChanged` on the session bus relays `wire.PanelReferencesChangedEvent` JSON to the add-in sink; same for `TaskPanelClosed`.

- [ ] **Step 1: Write the failing test**

Create `addin/events/events_phase0a_test.go`. Mirror the existing relay test for `PanelValueChanged` (find it in `events_test.go` and copy its harness — a fake `Sink` that captures JSON, a session/bus, emit, assert the JSON `type` + fields). The two assertions:
```go
// after wiring a capturing sink + session bus exactly as the existing PanelValueChanged relay test:
//   emit app.PanelReferencesChanged{WindowID:"w", ControlID:"faces", Refs:[]string{"face/a"}, Action:"set"}
//   -> sink receives JSON with "type":"panel.referencesChanged","windowId":"w","controlId":"faces","refs":["face/a"]
//   emit app.TaskPanelClosed{ID:"fix", Accepted:true}
//   -> sink receives JSON with "type":"taskPanel.closed","id":"fix","accepted":true
```
Write it concretely against the existing test's harness names (do not invent a new harness).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./addin/events/ -run Phase0a`
Expected: FAIL — the events are not relayed (sink receives nothing) / compile error until the union is extended.

- [ ] **Step 3: Write minimal implementation**

In `relayJSON`'s type union (~line 302), add (before the closing `]`):
```go
	wire.PanelReferencesChangedEvent | wire.TaskPanelClosedEvent |
```
In `subscribeSessionUI` (~line 78, after the `PanelValueChanged` subscription), add:
```go
		event.Subscribe(bus, event.After, func(_ event.Context, e app.PanelReferencesChanged) event.Outcome {
			return relayJSON(sink, wire.PanelReferencesChangedEvent{
				Type: wire.EventPanelReferencesChanged, WindowId: e.WindowID,
				ControlId: e.ControlID, Refs: e.Refs, Action: e.Action,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TaskPanelClosed) event.Outcome {
			return relayJSON(sink, wire.TaskPanelClosedEvent{
				Type: wire.EventTaskPanelClosed, ID: e.ID, Accepted: e.Accepted,
			})
		}),
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./addin/events/ -run Phase0a` → PASS. Then `go test ./addin/events/` → PASS.

- [ ] **Step 5: Commit**
```bash
git add addin/events/events.go addin/events/events_phase0a_test.go
git commit -m "feat(addin/events): relay panel.referencesChanged + taskPanel.closed to add-ins"
```

---

### Task H6: router handlers + registration

**Files:**
- Modify: `addin/router/ui_surfaces.go` (register 3 methods ~line 20; add 3 handlers ~after line 91)
- Test: `addin/router/ui_surfaces_test.go` (extend — mirror the existing `TestDockableWindowSetValueOverWire`)

**Interfaces:**
- Consumes: `wire.MethodDockableWindowsSetReferences`, `wire.MethodTaskPanelShow`, `wire.MethodTaskPanelClose`; the args DTOs (`SetDockableWindowReferencesArgs`, `ShowTaskPanelArgs`, `CloseTaskPanelArgs`); `decode`/`ok`; `r.readOnly`; the Session methods from H2/H4.
- Produces: wire methods routed to `s.SetDockableWindowReferences` / `s.ShowTaskPanel` / `s.CloseTaskPanel`.

- [ ] **Step 1: Write the failing test**

In `addin/router/ui_surfaces_test.go`, add a test mirroring `TestDockableWindowSetValueOverWire`: build a session + router, register handlers, set a dockable window holding a `referenceList`, call the `dockableWindows.setReferences` method through the router with `SetDockableWindowReferencesArgs{WindowId:"w", ControlId:"faces", Refs:[]string{"face/a"}}`, and assert the stored control's `Rows[0].Ref == "face/a"`. Add a second test routing `taskPanel.show` with a `ShowTaskPanelArgs{Panel: wire.TaskPanelSpec{ID:"fix",Title:"Fixed"}}` and asserting `s.TaskPanels().List()` has one panel; then `taskPanel.close` and assert it's empty. Use the existing test's router-invocation helper verbatim.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./addin/router/ -run 'Reference|TaskPanel'`
Expected: FAIL — methods not registered / handlers undefined.

- [ ] **Step 3: Write minimal implementation**

In `registerUISurfaceHandlers` (~line 20, after `setDockableWindowValue`):
```go
	r.readOnly(wire.MethodDockableWindowsSetReferences, setDockableWindowReferences)
	r.readOnly(wire.MethodTaskPanelShow, showTaskPanel)
	r.readOnly(wire.MethodTaskPanelClose, closeTaskPanel)
```
After `setDockableWindowValue` (~line 91) add:
```go
func setDockableWindowReferences(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetDockableWindowReferencesArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	s.SetDockableWindowReferences(req.WindowId, req.ControlId, req.Refs)
	return ok()
}

func showTaskPanel(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ShowTaskPanelArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.ShowTaskPanel(req.Panel); err != nil {
		return nil, err
	}
	return ok()
}

func closeTaskPanel(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.CloseTaskPanelArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.CloseTaskPanel(req.ID); err != nil {
		return nil, err
	}
	return ok()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./addin/router/ -run 'Reference|TaskPanel'` → PASS. Then `go test ./addin/...` → PASS.

- [ ] **Step 5: Commit**
```bash
git add addin/router/ui_surfaces.go addin/router/ui_surfaces_test.go
git commit -m "feat(addin/router): route setReferences + taskPanel.show/close to the session"
```

---

### Task H7: head/ui — render the `referenceList` control

**Files:**
- Modify: `head/ui/addin_panels.go` (add `case types.PanelReferenceList:` to `drawEditableControl`; add `drawPanelReferenceList`)

**Interfaces:**
- Consumes: `s.SetDockableWindowReferences`, `s.AddReferencesFromSelection` (H2/H3); `native.{BeginChild,EndChild,Selectable,BeginPopupContextItem,MenuItem,EndPopup,Button,SameLine,PushIDInt,PopID}` (all wrapped — verify `MenuItem` exists; if not use `Selectable` inside the context popup).
- Produces: a rendered scrollable list of the control's `Rows` (deriving label from `Ref` when `Label` empty), per-row right-click **Remove**, and **Add from selection** / **Clear** buttons.

This task is **build + live-validated** (ImGui draw calls; no Go unit test). TDD does not apply — verify by `go build ./...` then the live test (H10).

- [ ] **Step 1: Add the switch case**

In `drawEditableControl` (`head/ui/addin_panels.go:164` switch), add:
```go
	case types.PanelReferenceList:
		drawPanelReferenceList(s, windowID, control)
		return true
```
(Match the file's return convention — the existing editable cases return `true` after handling; mirror exactly.)

- [ ] **Step 2: Implement the renderer**

Add to `head/ui/addin_panels.go` (keep helpers ≤20 lines; split if needed):
```go
const (
	refListRowHeight = 22
	refListMaxRows   = 6
)

// drawPanelReferenceList renders a referenceList control: a scrollable list of picked refs with
// per-row Remove, plus Add-from-selection and Clear. Edits go through the session (which emits
// panel.referencesChanged to the owning add-in).
func drawPanelReferenceList(s *app.Session, windowID string, control wire.PanelControlSpec) {
	panelFieldLabel(control.Text)
	if remove := drawRefRows(control.Rows); remove >= 0 {
		s.SetDockableWindowReferences(windowID, control.ID, refsWithout(control.Rows, remove))
	}
	if native.Button("Add from selection") {
		s.AddReferencesFromSelection(windowID, control.ID, control.Accepts)
	}
	native.SameLine()
	if native.Button("Clear") {
		s.SetDockableWindowReferences(windowID, control.ID, []string{})
	}
}

// drawRefRows draws the scrollable rows and returns the index a Remove was clicked on, else -1.
func drawRefRows(rows []wire.PanelReferenceRow) int {
	height := float32(min(len(rows), refListMaxRows)*refListRowHeight + 8)
	remove := -1
	if native.BeginChild("##reflist", -1, height, true) {
		for i, row := range rows {
			native.PushIDInt(i)
			native.Selectable(refRowLabel(row), false)
			if native.BeginPopupContextItem("##refmenu") {
				if native.Button("Remove") {
					remove = i
				}
				native.EndPopup()
			}
			native.PopID()
		}
	}
	native.EndChild()
	return remove
}

func refRowLabel(row wire.PanelReferenceRow) string {
	if row.Label != "" {
		return row.Label
	}
	return row.Ref
}

func refsWithout(rows []wire.PanelReferenceRow, drop int) []string {
	refs := make([]string, 0, len(rows))
	for i, row := range rows {
		if i != drop {
			refs = append(refs, row.Ref)
		}
	}
	return refs
}
```
NOTE: confirm `native.BeginChild` returns a bool you should branch on and that `EndChild` must always run (mirror an existing `BeginChild` call site in the head). If `native.MenuItem` exists, prefer it over `native.Button` inside the context popup for idiomatic menu styling. `min` is the Go 1.21 builtin.

- [ ] **Step 3: Build**

Run: `go build ./...` (from the repo root, with the cgo head). Expected: compiles. Then `gofmt -l head/ui/addin_panels.go` (empty) and `golangci-lint run ./head/ui/` (clean).

- [ ] **Step 4: Commit**
```bash
git add head/ui/addin_panels.go
git commit -m "feat(head): render referenceList control (list + remove + add-from-selection)"
```

---

### Task H8: cgo binding — wrap `igBeginPopupModal`

**Files:**
- Modify: `head/internal/native/imgui.go` (C decl ~line 66; Go wrapper ~line 870)
- Modify: `head/internal/native/imgui_wrap.cpp` (impl ~line 215, beside the popup family)

**Interfaces:**
- Produces: `native.BeginPopupModal(name string, open *bool) bool` — opens a modal popup armed by a prior `native.OpenPopup(id)`; returns true while visible; writes the close state back through `open` (false when the user dismisses via the title-bar X). Paired with the already-wrapped `native.EndPopup()` and `native.CloseCurrentPopup()`.

Build-validated (no Go unit test for a cgo binding).

- [ ] **Step 1: Add the C declaration**

In `head/internal/native/imgui.go` cgo preamble, beside the popup decls (~line 66, after `void obk_ig_end_popup(void);`):
```c
int  obk_ig_begin_popup_modal(const char* name, int* open);
```

- [ ] **Step 2: Add the C++ implementation**

In `head/internal/native/imgui_wrap.cpp`, beside the popup family (~line 215), mirroring `obk_ig_begin_closable` (the `int* open` bool-out template at line 40):
```cpp
int  obk_ig_begin_popup_modal(const char* name, int* open) {
    bool p_open = (*open != 0);
    int visible = ImGui::BeginPopupModal(name, &p_open) ? 1 : 0;
    *open = p_open ? 1 : 0;
    return visible;
}
```

- [ ] **Step 3: Add the Go wrapper**

In `head/internal/native/imgui.go` (~line 870, after `CloseCurrentPopup`):
```go
// BeginPopupModal opens a modal popup (arm it first with OpenPopup(id)); returns true while
// visible. open is set to false when the user dismisses via the window's close control. Pair with
// EndPopup() and dismiss programmatically with CloseCurrentPopup().
func BeginPopupModal(name string, open *bool) bool {
	c, free := cstr(name)
	defer free()
	o := C.int(0)
	if *open {
		o = 1
	}
	visible := C.obk_ig_begin_popup_modal(c, &o) != 0
	*open = o != 0
	return visible
}
```

- [ ] **Step 4: Build**

Run: `go build ./head/...` (compiles the cgo unity build). Expected: links cleanly (ImGui's `BeginPopupModal` is in the vendored imgui already used by `imgui_unity.cpp`).

- [ ] **Step 5: Commit**
```bash
git add head/internal/native/imgui.go head/internal/native/imgui_wrap.cpp
git commit -m "feat(head/native): wrap ImGui BeginPopupModal for modal task panels"
```

---

### Task H9: head/ui — render the modal task panel

**Files:**
- Create: `head/ui/task_panels.go`
- Modify: `head/ui/chrome.go` (call `drawTaskPanels(s)` inside `drawChromeWindows`, after `drawAddInPanels(s)` ~line 155)

**Interfaces:**
- Consumes: `s.TaskPanels().List()`, `s.ResolveTaskPanel` (H4); `native.{OpenPopup,BeginPopupModal,EndPopup,CloseCurrentPopup,Button,SameLine,Separator,CenterNextWindow,SetNextWindowSize}`; `drawControlList` (`head/ui/addin_panels.go:110`).
- Produces: each open task panel rendered as a true modal (OK/Cancel); OK→`ResolveTaskPanel(id,true)`, Cancel/X→`ResolveTaskPanel(id,false)`.

Build + live-validated.

- [ ] **Step 1: Implement the renderer**

Create `head/ui/task_panels.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/wire"
	// + the head's app + native import paths used by addin_panels.go (copy them verbatim)
)

// armedTaskPanels tracks which task-panel popups have been OpenPopup'd, so each is armed exactly
// once when it first appears (ImGui requires OpenPopup before BeginPopupModal takes effect).
var armedTaskPanels = map[string]bool{}

// drawTaskPanels renders every open add-in modal task panel. Called each frame from drawChromeWindows.
func drawTaskPanels(s *app.Session) {
	open := map[string]bool{}
	for _, spec := range s.TaskPanels().List() {
		open[spec.ID] = true
		drawTaskPanel(s, spec)
	}
	for id := range armedTaskPanels { // forget panels that are gone
		if !open[id] {
			delete(armedTaskPanels, id)
		}
	}
}

func drawTaskPanel(s *app.Session, spec wire.TaskPanelSpec) {
	popupID := "##taskpanel-" + spec.ID
	if !armedTaskPanels[spec.ID] {
		native.OpenPopup(popupID)
		armedTaskPanels[spec.ID] = true
	}
	native.CenterNextWindow()
	native.SetNextWindowSize(480, 320)
	stillOpen := true
	if native.BeginPopupModal(spec.Title+"###"+popupID, &stillOpen) {
		drawControlList(s, spec.ID, spec.Controls)
		native.Separator()
		drawTaskPanelButtons(s, spec)
		native.EndPopup()
	}
	if !stillOpen { // user clicked the window close control → treat as Cancel
		native.CloseCurrentPopup()
		_ = s.ResolveTaskPanel(spec.ID, false)
	}
}

func drawTaskPanelButtons(s *app.Session, spec wire.TaskPanelSpec) {
	if native.Button(orDefault(spec.OKLabel, "OK")) {
		native.CloseCurrentPopup()
		_ = s.ResolveTaskPanel(spec.ID, true)
	}
	native.SameLine()
	if native.Button(orDefault(spec.CancelLabel, "Cancel")) {
		native.CloseCurrentPopup()
		_ = s.ResolveTaskPanel(spec.ID, false)
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
```
NOTE: set the imports to exactly what `addin_panels.go` uses for `app` and `native`. Confirm `native.SetNextWindowSize` (one-shot) vs `SetNextWindowSizeOnce` — for a modal a fixed size each frame is fine; use whichever the file's siblings use. The `title###id` trick gives a stable ImGui id independent of the visible title.

- [ ] **Step 2: Hook into the frame loop**

In `head/ui/chrome.go`, inside `drawChromeWindows(s)` immediately after `drawAddInPanels(s)` (~line 155):
```go
	drawTaskPanels(s)
```

- [ ] **Step 3: Build**

Run: `go build ./...`. Expected: compiles. `gofmt -l head/ui/` empty; `golangci-lint run ./head/ui/` clean.

- [ ] **Step 4: Commit**
```bash
git add head/ui/task_panels.go head/ui/chrome.go
git commit -m "feat(head): render add-in modal task panels (BeginPopupModal, OK/Cancel→resolve)"
```

---

### Task H10: live MCP validation + host regression

**Files:**
- Create (throwaway, DO NOT COMMIT): a small MCP driver under `head/cmd/` or a scratch script that drives the running head.

This task proves the whole path on a real frame (the project's **Live tests** discipline).

- [ ] **Step 1: Full host test + build**

Run: `go test ./app/... ./addin/...` (all green) and `go build ./...` (cgo head links).

- [ ] **Step 2: Live drive over MCP**

Build + launch the head with the MCPBridge add-in (the established recipe: `make install` the MCPBridge, run the head with `DISPLAY=:1` + lavapipe `VK_ICD_FILENAMES`, MCP at `http://127.0.0.1:7800/mcp`). Then, via an MCP driver:
- Have a test add-in (or MCPBridge directly) create a dockable window containing a `referenceList` control and a button that opens a `taskPanel.show` with a `referenceList` inside; OR drive the host APIs directly through the bridge tools `set_panel_references`, `task_panel_show`, `task_panel_close`.
- Select a face in the viewport, click **Add from selection**, and `capture_window` — confirm the picked ref renders as a row.
- Open the task panel; confirm it renders as a centered modal that blocks the background; click OK; confirm a `taskPanel.closed{accepted:true}` reaches the add-in (assert via the bridge/log) and the modal closes.
- `capture_window` at each step and visually confirm (screenshot): the list shows rows, the modal is centered and input-blocking, OK/Cancel work.

- [ ] **Step 3: Record evidence**

Save the screenshots to the scratchpad and note results in the PR description. Do NOT commit the throwaway driver or screenshots (`git status` must show only the source changes).

- [ ] **Step 4: No commit** (validation only). If a defect is found, fix via a focused TDD cycle on the relevant app/router task and re-run.

---

## Self-Review (completed by plan author)

- **Spec coverage:** the design's "Phase 0a-host" notes (router handlers for the 3 methods; `head/ui` rendering of `referenceList` with Add-from-selection gated by `Accepts` + per-row Remove emitting `PanelReferencesChangedEvent` NOT the scalar event; async modal task panel delivering `TaskPanelClosedEvent` without echoing control values; reuse the `PanelControlSpec` renderer inside the modal) are all covered: H2/H3 (referenceList store + selection, emits references event), H6 (router), H7 (referenceList render), H4 (async resolve→closed event, no value echo), H8+H9 (modal reusing `drawControlList`). H1/H5 wire the event type+relay.
- **Placeholder scan:** the only non-literal instructions are "match the existing harness/helper/import names" (H2/H4/H5/H6/H9) — these are precise *locate-and-mirror* directives against named existing code, not vague TODOs; each step names the exact file+symbol to copy. The stray `var got` reminder line in H2's test is explicitly flagged for removal.
- **Type consistency:** `SetDockableWindowReferences(windowID, controlID string, refs []string)`, `AddReferencesFromSelection(windowID, controlID string, accepts []string)`, `ShowTaskPanel(spec) error`/`CloseTaskPanel(id) error`/`ResolveTaskPanel(id, accepted) error`, `BeginPopupModal(name string, open *bool) bool` are used identically across the tasks that produce and consume them. Event field names (`WindowID`/`ControlID`/`Refs`/`Action`; `ID`/`Accepted`) match between H1 (app structs) and H5 (wire mapping).
- **Modality decision (recorded):** true input-blocking modal via a new `BeginPopupModal` cgo binding (H8), per the user's choice — not the floating-window fallback.

## Out of scope / follow-ups

- **MCPBridge regeneration:** the new client methods (`set_panel_references`, `task_panel_show`, `task_panel_close`) already carry `mcp:` annotations in `api/client`; regenerating the MCPBridge tool set is a separate change in the `Oblikovati.AddIns.MCPBridge` repo (it pins the api version; bump to v0.101.0 there when those tools are wanted).
- **api pin bump:** if host CI pins a specific `oblikovati.org/api` version (vs. the go.work local resolve), bump the host's pin to `v0.101.0` before pushing so CI sees the new wire symbols (`ci-bot-pin-approval-gate` discipline).
- **CalculiX add-in** consuming these (referenceList constraint pickers, task panels) is **Phase 3**, after the Phase-2 browser tree + engine flip.
