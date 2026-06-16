# Viewport & Mouse-Interaction Behaviours — Inventor / AutoCAD Reference

Status: research / audit reference (no implementation).
Last researched: 2026-06-16.

This document consolidates **mouse-click and viewport-interaction behaviours** of
Autodesk Inventor (with AutoCAD cross-references where the convention is shared)
so the Oblikovati team can audit our implementation against the behaviour
Oblikovati intends to mimic. Each behaviour is a discrete, checkable row with a
trigger, expected result, preconditions, and a source citation.

## How to audit

Each row below should be checkable against Oblikovati's `head/ui/` viewport code.
The most relevant files are:

- `head/ui/chrome_viewport.go` — the viewport widget + input routing.
- `head/ui/navigate.go`, `head/ui/navigate_test.go` — orbit / pan / zoom camera control.
- `head/ui/highlight.go`, `edge_highlight.go`, `tool_highlight.go`, `revolve_highlight.go` — pre-highlight / rollover / pick highlight.
- `head/ui/marking_menu_view.go`, `minitoolbar_view.go` — right-click marking menu + mini-toolbar.
- `head/ui/work_axis_selection.go`, `viewport_visibility.go` — selectable-set / selection-priority surface.
- `head/ui/viewport_environment.go` — environment (part/sketch/assembly) gating of what is selectable.

For each row: confirm the **trigger** maps to the same gesture, the **result** matches,
and the **precondition** (environment / state) is enforced. Rows marked
`AutoCAD-only` are convention cross-references, not strict Inventor parity targets —
flag in review whether Oblikovati follows the Inventor or the AutoCAD convention.

### Source key

- `CORPUS:inv:<file>` → `/home/vmiguel/git/oblikovati-workspace/extracted-behaviour/areas/<file>`
- `CORPUS:acad:<guid>` → `/home/vmiguel/git/oblikovati-workspace/Oblikovati/experiments/autocad-docs-scraper/out/md/cloudhelp/2027/ENU/AutoCAD-Core/files/<guid>.md`
- `WEB:<url>` → official Autodesk Help (preferred for Inventor specifics not captured in the local corpus).

---

## 1. Selection

| # | Trigger | Expected result | Precondition / env | Source |
|---|---------|-----------------|--------------------|--------|
| S1 | Hover cursor over geometry | Object **pre-highlights** (rollover highlight) to show what a click would select. On by default. In assembly/weldment, when turned off, prehighlight does not show during component placement. | Any 3D env; "Prehighlight" colour option (Tools > Options > Colors). | `CORPUS:inv:work-environment.md` L287 |
| S2 | Single left-click on pre-highlighted object | Object becomes selected (added to a single-item selection set). | Select tool active. | `WEB:https://help.autodesk.com/cloudhelp/2016/ENU/Inventor-Help/files/GUID-B8F6E805-2E4B-4366-8FF6-33F0ED1C1C4A.htm` |
| S3 | **Shift+click** or **Ctrl+click** an unselected object | Adds it to the selection set. | A selection set exists. | `WEB:GUID-B8F6E805` |
| S4 | **Shift+click** or **Ctrl+click** an already-selected object | Removes that object from the set; all others unaffected. | Object already selected. | `WEB:GUID-B8F6E805` |
| S5 | Left-click empty space (not on an object) | Clears the entire selection set. (Shift+click empty space also clears.) | — | `WEB:GUID-B8F6E805` |
| S6 | **Window select** — click-drag from upper-left toward lower-right | Selects only objects **fully enclosed** by the box. | Drag started on empty space. | `WEB:GUID-B8F6E805`; AutoCAD parity `CORPUS:acad:GUID-F754A1B4-3DC2-4F4F-94EB-98D9591A5F32` |
| S7 | **Crossing select** — click-drag from lower-right toward upper-left | Selects objects **enclosed OR crossed/intersected** by the box. | Drag started on empty space. | `WEB:GUID-B8F6E805`; AutoCAD parity `CORPUS:acad:GUID-11AB8C73-D35A-45DF-A7AE-4D03313DDACA` |
| S8 | **Shift + click-drag** a box | Adds all previously-unselected objects in the box to the set. | — | `WEB:GUID-B8F6E805` |
| S9 | **Ctrl + click-drag** a box | **Inverts** the selection status of every object inside the box. | — | `WEB:GUID-B8F6E805` |
| S10 | Right-click occluded geometry → **Select Other**, then use the arrows | Cycles through overlapping/occluded candidates under the cursor; click to commit the highlighted one. | Multiple objects under cursor. | `WEB:GUID-B8F6E805`; `WEB:https://help.autodesk.com/view/INVNTOR/2022/ENU/?guid=GUID-3E3537A9-565E-4F2E-B69D-069EDCDF1AE1` |
| S11 | Set **selection priority** via the arrow next to the Select command | Switches what a click prefers: **Edge Priority** (edges), **Feature Priority** (features), **Part Priority** (whole parts / components). | Part/assembly env; QAT Select dropdown. | `WEB:GUID-3E3537A9` (Select Command Reference); `CORPUS:inv:inventor-basics.md` L233 |
| S12 | Double-click a feature / component in graphics or browser | Enters edit for that object (e.g. activates sketch / edit feature / edit component in place). | Object is editable in current env. | `CORPUS:inv:inventor-basics.md` L133 (double-click view to activate) |
| S13 | (AutoCAD-only) Pickbox over stacked objects, hold **Shift+Spacebar** and click repeatedly | Cycles selection through objects lying on top of one another; Esc turns cycling off. | Selection preview on/off both supported. | `CORPUS:acad:GUID-0BDFD1F5-90EA-4224-B338-9E6961AA9596` |
| S14 | Selection filters (assembly) | Selection filters restrict which component types can be picked (perf + precision). | Assembly env. | `WEB:https://help.autodesk.com/view/INVNTOR/2024/ENU/?guid=GUID-CDE42DCC-6EB8-4CB0-B242-B4B9BC8DC84A` |
| S15 | What is selectable by environment | **Part:** bodies, faces, edges, vertices, features, work geometry, sketch entities. **Sketch:** sketch points/curves, constraints, dimensions. **Assembly:** components, constraints, work geometry; part sub-entities only under the right priority/edit scope. **Drawing:** views, dimensions, annotations, sheet objects. | Driven by active environment. | `CORPUS:inv:inventor-basics.md` L321 (sketch activate); environment gating general |

---

## 2. Context (right-click)

| # | Trigger | Expected result | Precondition / env | Source |
|---|---------|-----------------|--------------------|--------|
| C1 | Right-click in graphics window | A **marking menu** (radial menu) displays centred on the cursor position. | Marking menus enabled (default). | `WEB:https://help.autodesk.com/cloudhelp/2016/ENU/Inventor-Help/files/GUID-4B081B74-45EA-451E-84BE-9FFE0E2ACAE3.htm` |
| C2 | Right-press, drag toward an item, quick-release (a **mark/gesture**) | Selects that radial item by gesture without showing the full menu (muscle-memory mode). | Marking menu mode. | `WEB:GUID-4B081B74` |
| C3 | Right-click; read the shortened context list above/below the wheel | An **overflow menu** (shortened classic context menu) appears above or below the marking menu depending on cursor position. | Marking menu mode. | `WEB:GUID-4B081B74` |
| C4 | Marking-menu contents vary by environment | Part env exposes Hole/Extrude/Fillet/Work Plane; Sketch env exposes Line/Circle/Rectangle/Finish Sketch; Assembly env exposes Constrain/Place/Move/Rotate Component. | Active environment. | `WEB:GUID-4B081B74` |
| C5 | Right-click with no command active | Context offers **Repeat <last command>** to re-invoke the previous command. | Idle (no active command). | `WEB:https://help.autodesk.com/view/INVNTOR/2022/ENU/?guid=GUID-3E3537A9-565E-4F2E-B69D-069EDCDF1AE1` (Select context); convention — verify exact label |
| C6 | Right-click during an active command | Context offers in-command actions plus **OK / Done / Cancel** (Apply / Finish where relevant). | A command is in progress. | Convention (Inventor in-command marking menu); verify against `marking_menu_view.go` |
| C7 | Press **Esc** | Aborts / cancels the operation in progress. Some operations are not instantly interruptible; a message displays when cancellation completes. | A command/operation is running. | `CORPUS:inv:inventor-basics.md` L154–155, L183–184 |
| C8 | Toggle marking-menu vs classic context menu | The right-click menu style is a setting (marking menu vs. classic). Audit which Oblikovati ships and whether it is switchable. | Tools > Customize. | `WEB:GUID-4B081B74` (Open question — exact toggle path; see §6) |

---

## 3. Navigation

Mouse-button conventions are the Inventor defaults; rows marked `AutoCAD` are the
shared cross-product convention.

| # | Trigger | Expected result | Precondition / env | Source |
|---|---------|-----------------|--------------------|--------|
| N1 | Hold **middle mouse button** + drag | **Pan** the view (parallel to screen). | Any 3D env. | `WEB:`engineering/Autodesk shortcut guides; mirrors AutoCAD Pan `CORPUS:acad:GUID-17225A0F-7A50-41B9-8404-7D415518DEFB` |
| N2 | **Scroll the mouse wheel** | **Zoom** in/out. Default zooms toward the cursor location (zoom-to-cursor). Wheel direction is configurable (Application Options). | Any 3D env. | Autodesk shortcut guidance; AutoCAD Zoom `CORPUS:acad:GUID-072D3942-A308-455C-8A75-8E63FB62FA4C`; verify zoom-direction default |
| N3 | Hold **Shift + middle mouse button** + drag | **Free Orbit** (default 3-button binding). | Any 3D env. | `WEB:https://help.autodesk.com/cloudhelp/2025/ENU/Inventor-Help/files/GUID-08AFE9FE-E8BD-4648-B52C-F25CEE881FB6.htm` |
| N4 | Free Orbit pivot rule | Pivot = model geometry centre when the full model is in view; snaps to nearest edge/face/vertex when partially in view; = cursor location when the model is outside the view. | Free Orbit active. | `WEB:GUID-08AFE9FE` |
| N5 | Free Orbit ring — drag **inside** the orbit circle | 3D spatial rotation about the circle centre. | Free Orbit tool (F4) showing the ring. | `WEB:GUID-08AFE9FE` |
| N6 | Free Orbit ring — drag from left/right rim | Rotate about the **vertical** screen axis. | Free Orbit tool. | `WEB:GUID-08AFE9FE` |
| N7 | Free Orbit ring — drag from top/bottom rim | Rotate about the **horizontal** screen axis. | Free Orbit tool. | `WEB:GUID-08AFE9FE` |
| N8 | Free Orbit ring — drag **around the circle perimeter** | Roll: orbit **normal to the screen** (about the view axis). | Free Orbit tool. | `WEB:GUID-08AFE9FE` |
| N9 | Left-click when an orbit cursor is shown | Defines a **new pivot centre** for orbit. | Free Orbit tool. | `WEB:GUID-08AFE9FE` |
| N10 | **Constrained Orbit** (Navigation Bar) | Rotates about the model-space vertical axis through the TOP/BOTTOM ViewCube faces; horizontal mouse = turntable, vertical mouse = tilt. | Navigation Bar / Orbit flyout. | `WEB:GUID-08AFE9FE`; AutoCAD `CORPUS:acad:GUID-072D3942` |
| N11 | **F2** (hold) | Pan the graphics window. | Any 3D env. | `WEB:`Autodesk Inventor shortcuts; ARKANCE F-key tip |
| N12 | **F3** (hold) | Realtime zoom in/out. | Any 3D env. | `WEB:`Autodesk Inventor shortcuts |
| N13 | **F4** (hold) | Orbit (rotate) the model. | Any 3D env. | `WEB:`Autodesk Inventor shortcuts; `WEB:GUID-08AFE9FE` (F4 = Free Orbit) |
| N14 | **F5** | Return to the **previous view** (last display). | History of views exists. | `WEB:`Autodesk Inventor shortcuts |
| N15 | **F6** | Restore **Home / isometric** view orientation. | — | `WEB:`Autodesk Inventor shortcuts |
| N16 | **Zoom Window / Zoom Area** | Drag a box; view zooms to fit that box. | Navigation Bar / Zoom flyout. | AutoCAD Zoom tools `CORPUS:acad:GUID-17225A0F`; Inventor "Zoom Area" |
| N17 | **Fit / Zoom All** | Frames the whole model/scene to fit the window. (Often "End" key or Navigation Bar.) | — | `CORPUS:inv:work-environment.md` L312 (Zoom All transition); verify keybinding |
| N18 | **Look At** (face/edge/sketch) | Reorients the view normal to a selected planar face, work plane, or sketch. | A planar reference selected. | `WEB:GUID-08AFE9FE` (view reorient); Inventor "Look At" command |
| N19 | ViewCube — click a **face / edge / corner** (26 zones) | Snaps to that preset standard/iso orientation (animated transition if enabled). | ViewCube visible. | `WEB:https://help.autodesk.com/cloudhelp/2024/ENU/Inventor-Help/files/GUID-94F44C80-7313-4529-85D1-E64ABF732700.htm`; AutoCAD `CORPUS:acad:GUID-E6D3896C-AF39-4F5C-A57C-CACE2A1117F9` |
| N20 | ViewCube — click an **adjacent-face triangle/arrow** | Rolls to the neighbouring face view. | A face view is current. | `WEB:GUID-94F44C80` |
| N21 | ViewCube — click a **roll arrow** (upper-right) | Rotates the current view 90° CCW (left arrow) / CW (right arrow). | A face view is current. | `WEB:GUID-94F44C80` |
| N22 | ViewCube — **click-drag** the cube | Free-orbits the model interactively. | ViewCube visible. | `WEB:GUID-94F44C80`; `CORPUS:acad:GUID-E6D3896C` |
| N23 | ViewCube — click the **Home** icon | Resets to the model's Home view. | — | `WEB:GUID-94F44C80` |
| N24 | ViewCube hover/inactive state | Inactive ViewCube is partially transparent so it doesn't obscure the model; becomes opaque/active when the cursor is over it. | ViewCube visible. | `CORPUS:acad:GUID-E6D3896C` |
| N25 | **Navigation Bar** | Hosts ViewCube, SteeringWheels, Pan, Zoom tools, Orbit tools, and 3Dconnexion; floats along an edge of the drawing area. | — | `CORPUS:acad:GUID-17225A0F` |
| N26 | **SteeringWheels** | Collection of wheels offering rapid switching between specialized navigation tools (zoom/orbit/pan/rewind etc.). | — | `CORPUS:acad:GUID-072D3942`, `GUID-17225A0F` |
| N27 | Continuous Orbit (click-drag-release) | View keeps orbiting in the released direction until interrupted. | Continuous Orbit tool. | `CORPUS:acad:GUID-072D3942` (AutoCAD; verify Inventor parity) |
| N28 | View-transition smoothing | Transition time between viewing commands (Isometric, Zoom All, Zoom Area, View Face) is configurable; 0 = abrupt. | Tools > Options > Display. | `CORPUS:inv:work-environment.md` L312 |

---

## 4. Direct manipulation / drag

| # | Trigger | Expected result | Precondition / env | Source |
|---|---------|-----------------|--------------------|--------|
| D1 | Click-drag an under-constrained sketch entity | Entity moves; the sketch solver updates within remaining DOF. | Sketch env, DOF > 0. | Convention; verify against sketch tooling |
| D2 | Click-drag a component (assembly) | Component moves within its unconstrained DOF; grounded components do not move. | Assembly env, unconstrained DOF. | `CORPUS:inv:work-environment.md` L278 (ground at origin) |
| D3 | Click-drag a face/edge with a triad / manipulator handle | Direct-edit move/rotate along the picked handle axis (e.g. Press/Pull, body move triad). | Direct-edit tool active. | `CORPUS:inv:editing-part-bodies-and-faces.md`; convention |
| D4 | Hover feedback during a tool | Candidate geometry pre-highlights; picked geometry shows a distinct pick/selected highlight (hover + picks). | Any tool with selectable inputs. | `CORPUS:inv:work-environment.md` L287; Oblikovati `tool_highlight.go` |
| D5 | (AutoCAD-only) Grip-edit / nudge | Selected objects expose grips; dragging a grip or arrow-key nudging moves them. | AutoCAD object selected. | `CORPUS:acad:GUID-11AB8C73` |
| D6 | Drag to constrain (snap) | Dragging near a coincident point/edge snaps and can infer a constraint on release. | Sketch / assembly snap on. | Convention; verify |

---

## 5. Modifier & function keys (consolidated)

| Gesture | Action | Source |
|---------|--------|--------|
| Left-click | Pick / pre-highlighted select | `WEB:GUID-B8F6E805` |
| Left-click empty | Clear selection | `WEB:GUID-B8F6E805` |
| Shift+click / Ctrl+click | Add to / toggle selection set | `WEB:GUID-B8F6E805` |
| Drag L→R | Window select (enclosed only) | `WEB:GUID-B8F6E805` |
| Drag R→L | Crossing select (enclosed + crossed) | `WEB:GUID-B8F6E805` |
| Shift + drag box | Add box contents to set | `WEB:GUID-B8F6E805` |
| Ctrl + drag box | Invert box contents in set | `WEB:GUID-B8F6E805` |
| Double-click | Edit / activate (sketch, feature, component) | `CORPUS:inv:inventor-basics.md` L133 |
| Right-click | Marking menu + overflow context menu | `WEB:GUID-4B081B74` |
| Esc | Cancel / abort current operation | `CORPUS:inv:inventor-basics.md` L154 |
| MMB drag | Pan | Autodesk shortcuts |
| Wheel scroll | Zoom (to cursor; direction configurable) | Autodesk shortcuts |
| Shift + MMB drag | Free Orbit | `WEB:GUID-08AFE9FE` |
| F2 (hold) | Pan | Autodesk shortcuts |
| F3 (hold) | Zoom | Autodesk shortcuts |
| F4 (hold) | Orbit | Autodesk shortcuts / `WEB:GUID-08AFE9FE` |
| F5 | Previous view | Autodesk shortcuts |
| F6 | Home / isometric view | Autodesk shortcuts |
| Shift+Spacebar (+click) | (AutoCAD) cycle stacked objects | `CORPUS:acad:GUID-0BDFD1F5` |

Note: Inventor lets users **customize** mouse shortcuts and command aliases /
shortcut keys (Tools > Options > Customize > Keyboard). Default bindings above are
the audit target; Oblikovati should treat them as defaults, not hard-codes.
Source: `CORPUS:inv:work-environment.md` L64, L94, L547.

---

## 6. Open questions / ambiguities

1. **Wheel-zoom direction default.** Inventor's default scroll direction (scroll-up =
   zoom-in vs zoom-out) and zoom-to-cursor behaviour are user-configurable; the local
   corpus does not state the factory default explicitly. Sources: Autodesk shortcut
   guides assert zoom-to-cursor; confirm Oblikovati's default in `navigate.go` and
   match Inventor's out-of-box setting. (N2)
2. **Marking menu vs. classic context menu toggle.** The marking-menu page (GUID-4B081B74)
   describes the radial menu and overflow but the exact UI path to switch to the classic
   context menu was not captured. Verify the toggle exists and where. (C8)
3. **Repeat-last-command label and availability.** Confirmed as a general convention but
   not pinned to an Inventor Help quote in the gathered sources. Verify exact menu label
   and whether it appears in graphics vs. browser. (C5)
4. **Fit / Zoom-All keybinding.** Corpus confirms a Zoom All *command* and its transition
   behaviour but not the default key (commonly `End` in Inventor; `Home` is ViewCube Home).
   Confirm the binding. (N17)
5. **Continuous Orbit parity.** Documented for AutoCAD (GUID-072D3942); not confirmed as an
   Inventor feature in the gathered sources — treat as AutoCAD-only until verified. (N27)
6. **Selection priority defaults per environment.** Edge/Feature/Part priority is confirmed,
   but which is the default in each environment (and whether sketch has its own priority)
   was not captured verbatim — verify. (S11)
7. **Pre-highlight colour/style.** Confirmed that prehighlight exists and is a colour option;
   exact default colour and whether edges vs. faces pre-highlight differently is not pinned —
   audit against `highlight.go` / `edge_highlight.go`. (S1, D4)
8. **In-command marking menu (OK/Done/Cancel) contents.** Asserted by convention (C6); not
   quoted from a single Inventor Help page in the gathered sources — verify per-tool.

---

## 7. Conformance audit — Oblikovati `head/ui` + `app` (2026-06-16)

Legend: ✅ matches · 🟡 partial / divergent · ❌ not implemented · ➖ AutoCAD-only (not a parity target).
Evidence cites the implementing symbol. Rows marked **FIXED** were addressed in the audit commit.

### Selection
| # | Status | Evidence / note |
|---|---|---|
| S1 | 🟡 | Rollover prehighlight exists for sketch entities & active-tool candidates (`tool_highlight.go`, `hoverCandidate`) and origin work planes (`hoveredPlane`); general model-env face/edge/body rollover not confirmed. |
| S2 | ✅ | `handleViewportClick` → `Session.Pointer` → `picker.Pick`. |
| S3 | ✅ | `applyPickToSelection`; Shift/Ctrl adds a new object via `Selection.Toggle`. |
| S4 | ✅ **FIXED** | Shift/Ctrl+click on a selected object now removes just it (`Selection.Toggle`/`Remove`). Was a blind `append` that never removed **and duplicated** the entry (contradicting the SelectSet "de-duplicated" docstring). |
| S5 | ✅ **FIXED** | Click on empty space now clears (`clearSelectionOnEmptyClick`). The pick-miss path previously `return`ed without clearing. |
| S6 | ✅ **DONE** | Window select (drag L→R, fully-enclosed) — `Session.BeginBoxSelect`/`CommitBoxSelect` + `RayPicker.PickRegion` + `head/ui/box_select_view.go` rubber-band. Whole-body granularity (per-face/edge + sketch-entity box are #909 follow-ups). |
| S7 | ✅ **DONE** | Crossing select (drag R→L, enclosed-or-intersected) — same path, direction sets the mode. |
| S8 | ✅ **DONE** | Shift+box adds (`applyRegionToSelection`). |
| S9 | ✅ **DONE** | Ctrl+box inverts (`applyRegionToSelection`). |
| S10 | ❌ | No Select Other occluded-geometry cycling. Tracked. |
| S11 | 🟡 | `SelectionFilter` restricts *kinds* but there is no Edge/Feature/Part **priority** cycling or QAT dropdown. Tracked. |
| S12 | 🟡 | Browser double-click (`browser_view.go`) and dimension double-click (`chrome_viewport.go:316`) edit; viewport double-click-to-edit-feature is not wired. |
| S14 | 🟡 | `SelectionFilter` (kinds) exists; no assembly selection-filter UI surface. |
| S15 | ✅ | Environment gating via `viewport_environment.go` + filter kinds. |

### Context (right-click)
| # | Status | Evidence / note |
|---|---|---|
| C1 | ✅ | `openMarkingMenu` + `marking_menu_view.go` (radial ring). |
| C2 | 🟡 | Ring renders; quick-flick gesture-select (no menu shown) not confirmed. |
| C3 | ✅ | `drawMarkingOverflow` (linear overflow rows). |
| C4 | ✅ | Quadrants are per-environment `wire.MarkingMenuItem`s. |
| C5 | ❌ | No "Repeat <last command>". Tracked. |
| C6 | 🟡 | In-command OK/Done/Cancel marking content — verify per tool. |
| C7 | ✅ | `handleKeyboard` → `EscapePressed` → `PressKey{Escape}`. |
| C8 | ❌ | Marking menu only; no classic-context-menu toggle. Tracked. |

### Navigation
| # | Status | Evidence / note |
|---|---|---|
| N1 | ✅ | `ApplyNavigation` middle-drag → `cam.Pan`. |
| N2 | 🟡 | Wheel `cam.Dolly` zooms; **not** zoom-to-cursor (zoom is view-centred). Tracked. |
| N3 | ✅ | Shift+middle → `cam.Orbit`. |
| N4 | 🟡 | Orbits about camera target; no adaptive pivot (geometry-centre / edge-snap / cursor). |
| N5–N10 | ❌ | No Free-Orbit ring tool or Constrained Orbit. Tracked. |
| N11–N13 | ❌ | F2/F3/F4 hold-to-pan/zoom/orbit not bound (only F1=help is wired in `handleKeyboard`). Tracked. |
| N14 | ❌ | F5 Previous View — no view-history stack. Tracked. |
| N15 | 🟡 | `GoHome` exists (ViewCube Home + View ribbon) but is not bound to **F6**. Tracked. |
| N16 | ❌ | No Zoom Window/Area. Tracked. |
| N17 | ✅ | `View.ZoomAll`/`Session.FitView`; default **End** keybinding not bound. |
| N18 | 🟡 | M16 Orient split-button gives orientation presets, but no Look-At-selected-face. Tracked. |
| N19–N22 | 🟡 | ViewCube faces + Home + drag implemented (`viewcube*.go`); full 26-zone edges/corners and adjacent/roll arrows partial. Tracked. |
| N23 | ✅ | `GoHome` from ViewCube home icon (`viewcube_draw.go`). |
| N24 | ✅ | `InactiveOpacity` (transparent when not hovered). |
| N25 | ❌ | No floating Navigation Bar (functions live on the View ribbon). Tracked. |
| N26 | ❌ | No SteeringWheels. Tracked. |
| N27 | ➖ | Continuous Orbit — AutoCAD-only. |
| N28 | ✅ | `camera_anim.go` smooths transitions; configurable time — verify. |

### Direct manipulation / drag
| # | Status | Evidence / note |
|---|---|---|
| D1 | 🟡 | Sketch entity drag — verify against sketch solver. |
| D2 | 🟡 | Assembly grip/snap shipped (feat/assembly-grip-snap). |
| D3 | ✅ | `TriadDragging`/`ManipulatorDragging` direct-edit handles. |
| D4 | ✅ | `tool_highlight.go` hover + pick highlight. |
| D5 | ➖ | Grips/nudge — AutoCAD. |
| D6 | 🟡 | Drag-to-constrain snap — verify. |

### Resolved divergence (was flagged)
- **Left-drag orbit retired.** Left-drag previously orbited the camera "for discoverability", which
  collided with the sketch editor's left-click select/drag. `ApplyNavigation` no longer reads the left
  button — orbit is Shift+middle, pan is middle, and left-drag is selection (box-select on empty space).
  This is the Inventor-accurate model and unblocked box-select (S6–S9).

### Fixed/landed in the audit work (#916)
- **S4** Shift/Ctrl+click toggles per-object (and the SelectSet is now genuinely de-duplicated).
- **S5** Click on empty space clears the selection.
- **Left-drag-orbit retired** (`navigate.go`), unblocking left-drag selection.
- **S6–S9** Window/crossing box-select with Shift-add / Ctrl-invert — app state machine
  (`app/box_select.go`), body region pick (`app/region_pick.go`), head rubber-band + drag
  (`head/ui/box_select_view.go`); live-verified via Vulkan capture.
  Covered by `app/interaction_test.go`, `app/box_select_test.go`, `app/region_pick_test.go`,
  and the in-window `TestInWindowBoxSelectDragSelects`.

### Tracked follow-up issues (substantial unbuilt features)
- **#909** — Box-select **core landed** in #916 (window/crossing, Shift/Ctrl, left-drag-orbit retired).
  Remaining in #909: per-face/edge granularity + **sketch-entity** box-select + drag-to-move-on-entity.
- **#910** — Select Other: cycle occluded geometry (S10).
- **#911** — Function-key navigation F2–F6 + Previous View history (N11–N15).
- **#912** — Selection priority: Edge / Feature / Part (S11).
- **#913** — Navigation gaps: zoom-to-cursor (N2), Free/Constrained Orbit (N5–N10), Zoom Window (N16), Look At (N18), Navigation Bar (N25), SteeringWheels (N26).
- **#914** — ViewCube full 26-zone hit-testing + adjacent/roll arrows (N19–N22).
- **#915** — Marking menu: Repeat-last-command (C5) + classic-context toggle (C8).
