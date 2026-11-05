# Implementation milestones

All milestone exits below remain **open**. Rust tooling and the first model
dependency graph are implemented; see [implementation evidence](implementation.md).
The ADR migration portion of R0 is complete; its
[map and records](../decisions/README.md) do not certify runtime behavior.
The [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563)
tracks these milestones. R0 creates the remaining detailed work items.
Do not estimate the full port from source file counts; measure representative
work first.

## Dependency order

```text
R0 baseline, workspace, and gates
 +--> R1 kernel ownership and modeling proof ----+
 +--> R2 model/session/storage foundations ------+--> R3 editable part and replay
 +--> R1V native viewport proof -----------------+-------+--> R4 GUI and C ABI
                                                       |
R3 --> R5 complete part feature families                 |
R3 + required R5 families --> R6 assemblies/drawings/sheet metal
R4 + domain modules as needed --> R7 full shell, scripts, presentation
R5 + R6 + kernel exchange readiness --> R8 interoperability
R4 + R5 + R6 + R7 + R8 --> R9 conformance and release cutover
```

R1, R1V, and R2 can advance independently after R0. R5 need not wait for all
GUI work. Each R6 family starts when its own model and kernel dependencies
pass; it need not wait for unrelated part features. R7 can port base shell
work before every R6 panel exists. Keep Martensite's visualization priority
ahead of broad new exchange work.

Every milestone also depends on its required K11 extensions from the Go
kernel. Capabilities outside OCCT must be ported and adapted into Martensite
before the application features that use them can pass acceptance. Schedule
each extension with its first consumer; do not defer all extensions to R9.

## R0 — Fix the scope and establish the workspace

**Inputs:** the revisions in [source-map](source-map.md), the current issue
tracker, and the API and kernel repository workflows.

Work:

1. Create one Rust-port epic and small issues for the milestones and kernel/API
   dependencies. Search existing work before opening a duplicate. Keep original
   Go bug links attached to relevant corpus cases.
2. Generate a behavior inventory from Go packages, registered commands, wire
   methods, feature definitions, persistence codecs, script entry points,
   assets, and all nested modules. Classify implemented behavior, unverified
   source, known failures, and future roadmap items separately.
   Trace feature-to-kernel dependencies. Separate reusable Martensite behavior,
   missing OCCT ports, and required Go capabilities outside OCCT. Create
   Martensite extension work items for the last group, with source and tests.
3. Record comparison fixtures and Go baseline receipts from `make gate`. Check
   premises against OCCT/SolveSpace where geometry is involved. Record failures
   honestly; do not label them pre-existing without the required baseline proof.
4. Create the Cargo workspace, common verified toolchain, GPL license files,
   dependency pins, lockfile, formatting/lint rules, documentation checks,
   architecture checks, and `make gate`. Create only crates needed immediately.
5. Compile `oblikovati-api` and selected Martensite crates together. Resolve the
   fact that the initial kernel checkpoint may not be fetchable remotely.
   Allow ignored local path overrides for development; verify CI from clean
   published revisions or a checksummed source artifact.
6. Use the completed [ADR migration](../decisions/legacy-map.md): 66 legacy
   dispositions and 13 consolidated Rust records. Implement and verify its
   crate/API/kernel ownership rules; attach evidence to the behavior ledger.
   Configure PRs and checks to target `rust`; keep release publication disabled
   until R9. New decisions follow the [ADR policy](../decisions/README.md).

**Exit:** clean-checkout build and tests work without sibling working-tree
assumptions. Every inventory row has a stable ID, source location, target
module, milestone, and acceptance test requirement. Missing tools or tests
make the full gate fail. Workspace creation is not feature completion.
The legacy decision map identifies applicable content and its consolidated
Rust destination; old accepted status grants no Rust acceptance.

## R1 — Prove the Martensite evaluation interface

**Depends on:** R0; K01-K04 and the relevant K11 extensions from
[kernel readiness](martensite.md).

Work:

1. Build the application-owned kernel module with unit conversion, shape
   ownership, operation options, diagnostics, and validation. Keep toolkit
   imports contained in this module.
2. Test input copy/isolation, orientation and location, unit changes, failures,
   repeat runs, and the thread model. Record operations that mutate inputs.
3. Construct wires and faces, then extrude and revolve through real Martensite
   builders. Return provenance and analytic properties with the result.
4. Prove an initial persistent face mapping from maker history and semantic
   feature roles. Identify missing history or validation in upstream work items.
   Port and adapt any required Go-only kernel behavior into Martensite before
   accepting this interface; OCCT coverage cannot replace this work.
5. Verify tolerance and ownership against
   [ADR-0007](../decisions/ADR-0007-martensite-integration.md). Record evidence
   in the K01-K11 ledger. Compare general, degenerate, and near-degenerate
   cases with OCCT, including per-face checks.

**Exit:** the headless kernel proof passes with valid topology, analytic
properties, provenance, and unchanged inputs. A missing capability blocks the
dependent feature; a stub or unconditional success cannot pass this stage.

## R1V — Prove the native viewport path early

**Depends on:** R0 and K06; K05 and R1 for the final modeled-shape check.

Work:

1. Reproduce Martensite's offscreen acceptance using the pinned source, then
   establish native window presentation with its Vulkan renderer.
2. Retain the current ImGui interaction model. Test a narrow Rust/native UI
   binding, window input, clipboard, focus, text input, DPI, and resizing.
3. Define how the viewport and UI share presentation resources and completion
   fences. Test view creation/removal and shutdown while a frame is in flight.
4. Display the R1 shape, select a face, map it to model identity, show an
   overlay, and capture the rendered result. Verify resource invalidation
   after an edit.
5. Record Linux, Windows, and macOS presentation requirements. The Linux proof
   can unblock development; unsupported target profiles remain open gates.
   Verify the single [Vulkan ADR](../decisions/ADR-0008-vulkan-renderer.md)
   and its shared UI/platform interface. Record capability evidence separately.
   If proof requires a different design, supersede the affected ADR before
   dependent implementation; do not create competing renderer records.

**Exit:** a real native window can display, pick, resize, overlay, and capture
the modeled shape. The renderer reports failures; it cannot return a stale
capture as a success. Record remaining visual-mode requirements for R7.

## R2 — Port model, session, and storage foundations

**Depends on:** R0. Use named kernel fakes only for tests of orchestration.

Work:

1. Port stable IDs, ordered collections, document ownership, typed properties,
   units, parameter expressions, dependency invalidation, and cycle errors.
2. Port the shared constraint solver and derivative/scaling behavior. Add
   2D sketches, dimensions, DOF reporting, profiles, and work geometry needed
   by the first part. Keep one solver for later assembly placement.
3. Port session commands, transactions, undo/redo, events, document generations,
   cancellation, and asynchronous result adoption. Make query and edit paths
   explicit. Use deterministic fake clocks and storage in unit tests.
4. Implement the schema-2 YAML reader/writer, project files, key envelope
   codecs, atomic save, Save As/Copy, and lossless storage of supported resource
   and add-in sections. Reject unsupported or malformed content clearly.
5. Add a headless CLI and the first typed automation operations: version,
   documents, parameters, sketches, transactions, and storage. Take names and
   DTOs only from the API crate. Inventory the unimplemented host operations.

**Exit:** tests prove parameter/constraint edits, DOF reporting, transaction
rollback, event order, and storage round trips without a GPU. Unsupported
document content cannot be saved with silent loss. Kernel fakes do not count
as feature or topology evidence.

## R3 — Complete the first editable part

**Depends on:** R1 and R2.

Work:

1. Port the feature Definition/Add/Edit lifecycle, ordered history,
   end-of-part, suppression, and dependency-based recompute for extrusion.
2. Connect solved profiles and parameter values to Martensite. Pass achieved
   solve precision through the modeling tolerance policy.
3. Attach generated/modified/deleted provenance and stable reference keys to
   feature results. Read a Go document fixture and resolve saved references
   through recipe replay. Define any necessary key migration before writing it.
4. Edit parameters and feature definitions; retain the clean history prefix,
   discard stale worker results, and preserve the last valid display result.
   A failed feature must report health and quarantine only its dependents.
5. Run the same workflow from CLI and typed automation. Save, reopen, undo,
   redo, suppress, and restore the feature. Test a deleted reference and an
   ambiguous reference as explicit failures.

**Exit:** a fully constrained, parameter-driven sketch (DOF = 0) extrudes into
a valid solid. A dependent face reference survives parameter edits and file
replay. Analytic per-face checks and properties pass. No Go kernel runs in the
Rust application. This is the first headless product milestone.

## R4 — Connect the desktop and existing add-ins

**Depends on:** R3 and R1V.

Work:

1. Add document tabs, the first ribbon commands, feature browser, parameter
   editor, sketch input, tool preview/cancel, selection, undo/redo, and save
   dialogs through the shared session interface.
2. Implement add-in discovery, manifest and API version checks, activation,
   deactivation, event delivery, optional automation, and callback dispatch.
   Match the canonical C header and API Rust FFI definitions.
3. Define host response allocation/free ownership, returned string lifetimes,
   length validation, and error behavior. Do not let a Rust panic cross the
   C ABI. Keep the unavoidable callback bridge scoped to host lifetime.
4. Test activation and deactivation while the session queue is active. Prevent
   reentrant calls and waits that deadlock the session thread. Preserve the
   existing queued-request timeout semantics unless a reviewed API change
   explicitly changes them; never retry a timed-out edit blindly.
5. Load a Go test add-in, a Rust test add-in, and the existing Go MCPBridge.
   Build the same R3 part through both typed clients. Exercise notifications,
   version rejection, automation errors, shutdown, and repeated lifecycle calls.
6. Drive the foreground Rust application through MCPBridge and inspect its
   screenshots, model result, and diagnostics.

**Exit:** the GUI, CLI, and both add-in languages operate the same tested part
workflow. At least 20 activate/deactivate cycles per test add-in complete
without leaks or hangs. Do not unload a Go shared library merely because it
has deactivated; prove and retain the reference host's safe library lifetime.
This is the first integrated preview, with its supported scope stated.

## R5 — Complete part modeling by feature family

**Depends on:** R3 and each operation family's Martensite readiness, including
K11 extensions from the Go kernel where OCCT has no equivalent capability.

Port each family as a separate issue and PR:

| Family | Required work and acceptance |
| --- | --- |
| Sketch and reference completion | Remaining 2D/3D entities, constraints, projected geometry, dimensions, profiles, paths, and text. Test editing and reference preservation. |
| Base solids and surfaces | Remaining extrude/revolve modes, sweep, loft, holes, ribs, coils, work features, and construction surfaces found in the inventory. |
| Boolean and topology edits | Join, cut, intersect, split, trim, combine, and multi-body results. Validate tangent, coincident, seam, and near-degenerate cases. |
| Dress-up | Fillet, chamfer, draft, shell, thicken, and related feature options. Use the kernel's general operation path and complete history. |
| Repetition and derived geometry | Patterns, mirrors, transforms, copies, derived parts, and dependency propagation with stable identities. |
| Surface and query completion | NURBS/surface editing, continuity and analysis, healing as an explicit operation, transient B-rep, analytic measurements, and section curves. |

Each issue includes typed automation, persistence, edit/recompute, health,
undo/redo, UI exposure when available, and its corpus. Do not close a family
because its constructor passes one happy-path example. Reproduce representative
valid solids first, then broaden inputs and options.

**Exit:** every baseline part-feature row has passing tests or a separately
approved scope decision. Required unsupported operations remain open work.
Required Go-kernel extensions must be integrated in Martensite and tested
through their application features. Absence from OCCT is not an exclusion.
Report kernel rule checks and regressions with each dependency update.

## R6 — Port assemblies, sheet metal, drawings, and presentations

**Depends on:** R3, relevant R5 families, and K07/K08 as needed.

Create separate issues for each domain:

- **Assemblies:** occurrences and proxies, document resolution, mates/joints,
  solver integration, patterns, representations, motion/contact behavior,
  assembly features, BOM, and derived models. Test repeated instances and
  distinct occurrence reference paths to the same underlying part.
- **Sheet metal:** rules, thickness, bends, reliefs, unfold/refold, flat
  patterns, and their stored recipes. Compare folded and flat outputs with
  appropriate analytic and reference fixtures.
- **Drawings:** sheets, styles, projected/section/detail views, exact HLR,
  dimensions, annotations, tables, and model associativity. Reopen and
  regenerate drawings after part and assembly changes.
- **Presentations:** stored views, component transforms, explode paths,
  visibility, and playback behavior present in the baseline.

Extend API implementations and file codecs as each domain becomes usable.
Preserve missing/broken reference diagnostics and avoid infinite reference
cycles. Keep all four document kinds and projects in the storage corpus.

**Exit:** representative multi-document projects reopen with the same recipes,
relationships, properties, references, and supported behavior. UI panels can
finish in R7, but the domain and automation tests must already pass.

## R7 — Complete UI, scripting, and application behavior

**Depends on:** R4; relevant R5/R6 domains and K09 for their UI.

Work:

1. Complete the ribbon/browser inventory, editors, command window, navigation,
   selection filters, snapping, manipulation, context/marking menus, shortcuts,
   dialogs, themes, messages, and persisted preferences.
2. Complete display modes, materials and appearances, lighting, environment
   maps, point clouds, client graphics, overlays, view capture, and OpenPBR
   workflows. Track required renderer extensions explicitly with Martensite.
3. Select and prove the Lua engine behind the script interface. Preserve Lua
   5.1 behavior, typed API groups, console/editor behavior, limits, cancellation,
   and diagnostics. An in-process native VM is not a substitute for the old
   containment requirement without new proof or an isolation design.
4. Complete add-in UI contributions, lifecycle notifications, catalogue and
   installation flows, application options, reports, telemetry preferences,
   help, and external resources. Test network integrations with named fakes.
5. Finish CLI commands, batch modes, exit codes, and generated help/manuals.
   Ensure headless operation does not create a window or require a GPU.

**Exit:** every baseline UI/script/application row has behavior tests and the
required live evidence. Visual parity is checked per workflow, including
text, selection, overlays, and errors, not by one whole-window screenshot.

## R8 — Complete exchange and external workflows

**Depends on:** relevant R5/R6 behavior and K10. Keep the kernel's current
visualization priority; do not demand all exchange toolkits for early parts.

Work:

1. Split the exchange inventory by format, direction, document kind, metadata,
   assembly handling, platform, and required external tools.
2. Consume released Martensite exchange capabilities. DWG uses
   [acadrust](https://github.com/hakanaktt/acadrust) inside the kernel. All
   exchange parsers, writers, and format-specific translators belong to
   RustyCADKernel, including required Go behavior outside OCCT. Do not port
   exchange implementations in the application scope.
3. Port application commands, dialogs, API routing, resource storage, and
   transactional adoption of kernel results. Include the external exporter
   and add-in workflows from the inventory; Inventor, OLE, and SolidWorks
   format implementations remain kernel-owned. Follow
   [ADR-0018](../decisions/ADR-0018-exchange-format-ownership.md).
4. Test import, edit, save, reopen, and export with units, analytic face types,
   topology, names, colors, references, and assembly structure. Include the
   elliptical-arc STEP regression linked in the source map.
5. Document formats that were already unavailable at the fixed baseline as
   roadmap work. Required baseline formats cannot be silently removed.

**Exit:** each required format row links to kernel release and round-trip/oracle
evidence, plus passing application integration tests. Kernel tests and package
releases remain the kernel repository's responsibility. Clear errors preserve
source files on failure. Current testing is local Linux; Windows and macOS
remain deferred.

## R9 — Prove conformance and switch releases

**Depends on:** all required rows from R0-R8 and upstream release readiness.

Work:

1. Reconcile all behavior rows, Go test mappings, public traits, wire methods,
   command inventory, document/key fixtures, and add-in workflows. Resolve
   every unclassified difference and required missing feature.
   Require passing evidence for every Go-kernel extension outside OCCT as
   well as every OCCT-derived capability used by the application.
2. Run the full platform, coverage, duplication, numeric, performance, live,
   and packaging gates in [verification](verification.md). Repeat required
   checks against the exact dependency revisions to be released.
3. Produce Linux, Windows, and macOS GUI/CLI packages with native dependencies,
   resources, licenses/notices, file associations, and version metadata.
   Test installation, update, removal, and operation outside the source tree.
4. Test Go-to-Rust project opening on copies, failure recovery, side-by-side
   preview installation, and return to the previous Go build. If a new schema
   is needed, the backup remains readable by that Go build.
5. Update application documentation and release channels. Retire the Go
   production release path only through the tracked cutover task after all
   gates pass. Retain the fixed Go checkout and fixtures as test references.

**Exit:** the Rust application satisfies the required baseline behavior and
stronger kernel rules, with no Go kernel or Go host in its production path.
Existing Go add-ins still work through the C ABI. The release manifest names
the application, API, Martensite, schema, key, and ABI versions. Release
publication and replacement of the current channel are explicit cutover work.
