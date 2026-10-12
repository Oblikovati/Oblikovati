# Source baseline and port map

Inspected on 2026-10-10. These are source observations, not new runtime test
results. Freeze the revisions below for the initial comparison corpus. Any
later baseline update needs a reviewed change and regenerated evidence.

## Revisions

| Input | Revision | State at inspection |
| --- | --- | --- |
| `Oblikovati` | `aa8297d1b0ac33c024f19e54821536d76e8a447a` | Clean `develop`; equals fetched `origin/develop`. |
| `Oblikovati.API` | `9b5da090a9260892f54b4762d1225b5f892cbeb4` | `feat/rust-api`, API version `0.154.0`; PR #296 open. Nested untracked checkout is outside this baseline. |
| `RustyCADKernel` | `0b4bc21d0a59a7c0cb1c41c26059623162d00d56` | Local `main` integration checkpoint. Active uncommitted work exists and is excluded. |
| Martensite remote reference | `e4144c4c3ac1bad8487367d3c51595f048a8fc5d` | Local `origin/main` reference at inspection; not a fresh remote availability check. |
| Martensite's OCCT oracle | `3d097a0328e71b826377d4814ab05ec3c3d23871` | OCCT `8.1.0-dev1`, as pinned by Martensite. |
| Initial `oblikovati-rust` | `5388046d` | Empty independent root on the named `rust` branch. |

The branch names above record the state at baseline inspection. On 2026-10-10,
the application repository's former default `develop` was renamed to
`go-legacy`, and `rust` became the default. The Go baseline commit and its
history are unchanged; this branch change does not update any source baseline.

The Martensite checkpoint is far ahead of its local remote reference. Do not
assume that CI can fetch it. R0 must establish a published, tested revision or
a reproducible source artifact before making it a CI dependency. Do not copy
the active working tree and call it a fixed dependency.

Use Rust 1.98.1, edition 2024, as the initial candidate toolchain: that is the
Martensite pin. The API declares Rust 1.95 as its minimum. R0 must compile both
dependencies together and then pin the verified common toolchain and lockfile.

## Evidence entry points

The following paths are relative to their named sibling repository. Read the
file at the revision above when recreating this plan's evidence.

| Repository | Evidence |
| --- | --- |
| `Oblikovati` | `CLAUDE.md`, `go.mod`, `head/go.mod`, `app/session.go`, `addin/router/router.go`, `addin/dispatch/queue.go` |
| `Oblikovati` | `persistence/manifest.go`, `persistence/io.go`, `model/identity/encoding.go`, `model/identity/refkey.go`, `solve/solve.go` |
| `Oblikovati` | `head/internal/native/native.go`, `head/internal/addinhost/loader.go`, `head/internal/addinhost/hostcall.go` |
| `Oblikovati` | `script/engine.go`, `architecture/decisions/ADR-0016-shared-library-addins-mcp-bridge.md`, ADR-0018, ADR-0020, ADR-0028, ADR-0034, ADR-0043, ADR-0044 |
| `Oblikovati.API` | `rust/Cargo.toml`, `rust/src/ffi.rs`, `rust/src/client/transport.rs`, `include/oblikovati_addin.h`, `Makefile`, `RELEASING.md` |
| `RustyCADKernel` | `README.md`, `Cargo.toml`, `rust-toolchain.toml`, `crates/kernel/src/lib.rs`, `docs/porting-plan.md`, `docs/vulkan-renderer.md` |
| `RustyCADKernel` | `porting/api-inventory/completeness.json`, `porting/renderer/capabilities.json`, toolkit tests, `crates/tkbrep/src/topods/shape.rs`, `crates/tkernel/src/standard/handle.rs` |

The Go architecture notes and preexisting ADRs are legacy references. Some
contain old paths, gRPC plans, or behavior that later rules changed. Use the
router, canonical C ABI, and validated tests as behavior evidence. Current
user requirements and the new accepted Rust ADRs govern the Rust design;
legacy accepted status does not carry forward. Adapt applicable decisions
and consolidate them by boundary under the [Rust ADR policy](../decisions/README.md).
The completed [legacy map](../decisions/legacy-map.md) records all 66
dispositions and the supplemental Martensite renderer inspection used for the
13 new records. That inspection does not update the baseline above.
The Go README's early status also understates the source now present in
assemblies, drawings, and other modules. Inventory code and tests directly.

## Application mapping

Milestone IDs are defined in [milestones](milestones.md).

| Go source | Rust destination | Action and required behavior | Stage |
| --- | --- | --- | --- |
| `kernel/`, geometric parts of `math/` | `RustyCADKernel`, consumed through `oblikovati-martensite` | Reuse equivalent Martensite capabilities, complete missing OCCT ports, and port/adapt required Go capabilities outside OCCT into Martensite. Adapt to its topology model; retain Go source provenance, tests, and fixtures. | R0 inventory; R1-R8 as required |
| `solve/`, `solve/ad/` | `oblikovati-solve` | Port the shared numerical solver, rank/DOF results, derivative path, scaling, and failure behavior. Martensite is not assumed to supply an Inventor-style sketch solver. | R2, R6 |
| `model/param`, `model/depend` | `oblikovati-model` | Units, formulas, dependency invalidation, cycle errors, stable parameter IDs, and incremental recompute. | R2 |
| `model/doc`, `model/attr`, `model/identity` | `oblikovati-model` | Document lifecycle, properties, attributes, identity, key codecs, and binding. Resolve cycles with IDs and owned collections. | R2, R3 |
| `model/sketch` | `oblikovati-model::sketch` | 2D/3D entities, constraints, projections, profiles, paths, text, dimensions, and health. | R2, R3, R5 |
| `model/feature`, `model/compdef` | `oblikovati-model::feature` | Definitions, history, end-of-part, suppression, dependency replay, provenance, and feature health. Geometry goes through Martensite. | R3, R5 |
| `command`, `event`, `app` | `oblikovati-session` | Commands, transactions, undo/redo, event order, tool lifecycle, selection, and session state. Split UI state from document state. | R2-R4, R7 |
| `persistence`, `yamlcodec`, `userconfig` | `oblikovati-storage` | Schema-2 YAML, projects, resources, atomic save, Save As/Copy, old versions, preferences, and configuration paths. | R2, R3, R6 |
| `addin/router`, `addin/opregistry`, `addin/dispatch` | `oblikovati-automation` and session dispatch | Shared API types, query/edit classification, single undo recording, queue limits, errors, and trace records. | R2, R4 |
| `head/internal/addinhost` and add-in registration in `app` | `oblikovati-addin-host` | C exports, major/minor check, library lifetime, callbacks, notifications, automation export, and loading order. | R4 |
| `renderer`, `scene`, `model/facetstore`, `clientgraphics` | `oblikovati-viewport` | Martensite presentation, cache invalidation, camera, picking, stable entity references, overlays, capture, and instancing. | R1V, R4, R7 |
| `head/internal/native`, `head/ui`, GUI command | `oblikovati-platform`, `oblikovati-shell`, `oblikovati-head` | Native window, ImGui binding, ribbon/browser, tools, dialogs, focus, themes, and GPU lifecycle. | R1V, R4, R7 |
| `model/assembly`, `occurrence`, `proxy`, `bom`, `bomapi`, `contentset` | Model assembly modules | Occurrence identity, referenced documents, transforms, mates/joints, representations, patterns, BOM, and missing-reference health. | R6 |
| `model/sheetmetal` and sheet-metal feature definitions | Model sheet-metal modules and kernel adapter | Thickness, bends, unfold/refold, flat pattern, and feature history. | R6 |
| `model/drawing`, `linetype`, `text` | Model drawing modules and viewport | Views, dimensions, annotations, styles, HLR, sheets, and regeneration after model edits. | R6 |
| `model/material`, `style`, `colorscheme`, `display`, `theme`, `osfont` | Model presentation modules, shell, platform | Materials, appearances, lighting/display modes, fonts, themes, and persisted settings. | R7 |
| `model/analysis`, `bodyapi`, `pointcloud` | Model/query modules and viewport | Analytic measurement, transient bodies, point-cloud resources, display, and selection. | R5-R8 |
| `model/exchange` and translator modules | RustyCADKernel format implementations; application exchange adapter and workflows | Inventory each format and both directions; kernel owns translators and DWG/acadrust. Application preserves kernel results, diagnostics, and source resources. ADR-0018. | R8 |
| `script`, script console | `oblikovati-script` and shell | Lua 5.1 surface, typed API access, quotas, cancellation, diagnostics, and editor behavior. | R7 |
| `cmd/oblikovati-cli`, command/manual generators | `oblikovati-cli` and tools | Commands, exit codes, batch workflows, help, manuals, script execution, and headless behavior. | R2, R7-R9 |
| `addincat`, `update`, `usagestats`, `report`, `crcpost` | Host/platform integration modules | Catalogue, installation/update lifecycle, opt-in telemetry, reporting, and existing transport settings. Inventory each behavior before replacement. | R7, R9 |
| `perf`, `archguard`, `scripts`, `.github`, release tooling | Rust checks, benchmarks, and release workflow | Preserve architecture rules, complete gate discovery, packaging, artifacts, and performance evidence. | R0, R9 |

R0 expands this module map into individual behavior rows. The map is a starting
inventory, not a claim that every exported function was audited in this task.
Assets, templates, fonts, shaders, material libraries, and document fixtures
also need entries and retained provenance.

For each application feature, trace the Go-kernel operations it calls and
classify the complete behavior against OCCT and Martensite. Record required
behavior outside OCCT as a Martensite extension, with its Go source revision,
tests, consumers, target kernel module, and owning work item. The inventory
must cover these extensions as well as the OCCT-derived kernel. A missing
same-named OCCT method alone does not establish a missing capability.

## Nested Go modules

The baseline tracks seven `go.mod` files: the root, `head`, two add-in test
libraries under `head/internal/addinhost/testdata`, and translators
`model/exchange/translators/{inventor,olecf,solidworks}`. A root `go test ./...`
does not cover all seven. Use the Go reference's full gate when establishing
baseline evidence; preserve its module discovery when building comparison jobs.

Do not infer translator support from directory names. Record successful import
and export cases, diagnostics, unavailable formats, and platform restrictions
from the actual implementations and fixtures.

## Existing tracker items that affect acceptance

The issue tracker was checked for Rust/Martensite application-port work and
open application defects. No matching application-port epic was found in that
search. R0 must check again before creating one. The Martensite repository
returned no open issues in this inspection; its checked-in porting ledger is
also required evidence.

| Existing item | Effect on this port |
| --- | --- |
| [API #295](https://github.com/Oblikovati/Oblikovati.API/issues/295) and [PR #296](https://github.com/Oblikovati/Oblikovati.API/pull/296) | Track the shared Go/Rust API and release prerequisites. |
| [API #187](https://github.com/Oblikovati/Oblikovati.API/issues/187) | Parameter expressions in sketch entity requests remain separate API work; do not assume every scalar request accepts an expression. |
| [#3510](https://github.com/Oblikovati/Oblikovati/issues/3510) | Add the oblique-cylinder/elliptical-arc STEP case to exchange acceptance. |
| [#3531](https://github.com/Oblikovati/Oblikovati/issues/3531) | Section operations must not use display meshes to replace missing analytic intersections. |
| [#3544](https://github.com/Oblikovati/Oblikovati/issues/3544) | Add the reported crossing-cylinder case to deterministic tessellation checks. |
| [#3552](https://github.com/Oblikovati/Oblikovati/issues/3552) | Include tangent-contact Boolean cases; do not treat Go output as the geometry oracle. |
| [#3559](https://github.com/Oblikovati/Oblikovati/issues/3559) | Keep native UI/GPU lifecycle stress tests in the new gate. The reported Go crash is not a verified Rust defect. |
| [#3526](https://github.com/Oblikovati/Oblikovati/issues/3526), closed | Preserve the requirement that the full gate includes the windowed application and every nested module. |

Open M11-M19 work and M48 kernel remediation are not replaced or closed by this
plan. Link relevant issues to new acceptance rows. Reproduce a reported defect
before using it as a result comparison; do not copy an incorrect expected result
just because it came from the Go implementation.
