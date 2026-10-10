# Port the Oblikovati application to Rust

Date: 2026-10-10. Status: implementation plan; no application code is ported yet.

## Outcome

Build the desktop application and CLI in Rust, using `oblikovati-api` for the
public contract and Martensite for geometry, topology, and kernel operations.
Keep the document model, editable feature history, automation behavior, and
user workflows of the Go application. Verify these behaviors against a fixed
source revision and recorded tests.

The first useful result is a part that a Go or Rust add-in can create, edit,
save, reopen, and display. Complete the remaining application features in
separate milestones. A working extrude does not mean the port is complete.

The [source map](source-map.md) records the starting revisions. The
[milestones](milestones.md) define the work. The
[verification plan](verification.md) defines evidence required to close it.

## Scope and ownership

| Repository | Owns | Port action |
| --- | --- | --- |
| `oblikovati-rust`, branch `rust` | GPL application, feature recipes, documents, session, GUI, CLI, host | Implement the application in Rust. |
| `Oblikovati.API` | Apache public types, traits, wire declarations, clients, C ABI | Consume the Rust crate. Change the shared contract here first when required. |
| `RustyCADKernel` | Martensite geometry, topology, algorithms, and visualization | Integrate tested capabilities. Fix missing kernel behavior in this repository. |
| `Oblikovati.AddIns.*` | Separate add-ins | Keep the C ABI. Test existing Go add-ins against the Rust host. |
| `Oblikovati` | Go reference application | Read source and run comparison tests at the recorded revision. |

Do not translate `Oblikovati/kernel` into another production Rust kernel.
Keep its tests and useful fixtures as application acceptance inputs. Do not
ship the Go kernel, call it over FFI, or link native OCCT to supply a missing
Martensite operation. Native OCCT and SolveSpace remain test oracles.

All application features found in the baseline remain in the port inventory,
including incomplete features. An unfinished Go roadmap item remains separate
feature work; the language port does not silently promise its completion.
Source presence, passing unit tests, and demonstrated user behavior are separate
states in the inventory.

## Architecture

Use a Cargo workspace with focused crates. Create each crate when its first
tested behavior is needed. The names below are proposed application package
names, not existing packages.

| Proposed crate | Responsibility and interface |
| --- | --- |
| `oblikovati-model` | Document recipes, IDs, parameters, units, sketches, features, assembly and drawing models, and health. Owns dependency and naming rules. |
| `oblikovati-solve` | The shared constraint solver for sketches and assembly placement. Receives explicit variables and residuals; returns status and achieved precision. |
| `oblikovati-martensite` | Kernel adapter: construct and evaluate shapes, return provenance and diagnostics, validate results, query analytic geometry, and tessellate. |
| `oblikovati-storage` | YAML documents, projects, resources, atomic writes, migrations, and preferences behind injected storage interfaces. |
| `oblikovati-session` | Commands, transactions, undo/redo, events, selection, tools, job scheduling, and document lifecycle. |
| `oblikovati-automation` | Implements the API traits and wire router against session operations. Classifies queries and edits once. |
| `oblikovati-addin-host` | Library discovery, C ABI, callback dispatch, buffer ownership, compatibility checks, and add-in lifecycle. |
| `oblikovati-viewport` | Martensite visualization adapter, camera, presentation, pick-to-reference mapping, overlays, and capture. |
| `oblikovati-shell` | Dear ImGui UI state, ribbon, browser, panels, dialogs, and input. Uses session operations. |
| `oblikovati-platform` | Native window, UI binding, clipboard, file dialogs, and other OS integrations. |
| `oblikovati-script` | Lua execution and its limits behind an application-owned interface; routes model calls through automation. |
| `oblikovati-cli` / `oblikovati-head` | Composition roots and shipped binaries. Select and inject concrete adapters. |

Keep internal modules under 500 lines and split by responsibility. Do not
rebuild the current large `app.Session` as one Rust struct with all UI concerns.
Session modules own their state; the composition root connects them explicitly.

The dependency direction is:

```mermaid
flowchart TD
    host[Add-in host and script] --> automation
    automation --> api[Oblikovati.API]
    automation --> session
    shell[GUI shell] --> session
    cli[CLI] --> session
    session --> model
    model --> api
    model --> solver
    kernel[Martensite adapter] --> model
    storage[Storage adapter] --> model
    kernel --> martensite[RustyCADKernel]
    viewport --> martensite
    shell --> viewport
    shell --> platform
```

Arrows mean imports. The binaries inject concrete kernel and storage adapters
into model-owned interfaces. Keep model and solver independent of the window,
GPU, filesystem, and add-in loader. Only the kernel adapter and viewport import
Martensite toolkits. Keep native UI calls inside the platform module. Enforce
these import rules with workspace checks.

The kernel interface belongs to the model's evaluation needs. It hides shape
ownership, operation history, tolerance conversion, and validation. It is not
a second copy of every OCCT class. Use immutable recipe inputs and opaque,
revision-scoped shape references. Tests cross the same interface as features.
Use named fakes for external I/O; geometry correctness tests use Martensite.

## Contract and version rules

- Import `oblikovati_api::{types, contract, wire, client, ffi}`. Do not copy
  public DTOs, enum values, method strings, or ABI declarations into this repo.
- Keep contract changes in `Oblikovati.API/go` first, then regenerate the Rust
  declarations and run the two-language gate. Application CI consumes a fixed
  API revision and checks the supported method inventory against it.
- Implement the applicable Rust contract traits and typed wire operations.
  Preserve numbers, omitted fields, nulls, errors, transaction effects, and
  event order. Serialization parity alone is not host behavior parity.
- Retain existing add-in version exports and compatibility behavior. Do not
  claim full add-in compatibility while only part of the host is implemented.
  Early builds are development previews with a tested method inventory.
- API PR [#296](https://github.com/Oblikovati/Oblikovati.API/pull/296) is open
  at this plan's baseline. Pin its exact tested revision for early integration.
  Move to a released revision after its merge and release checks pass.
- The website change in
  [PR #7](https://github.com/Oblikovati/website-oblikovati.org/pull/7) must be
  deployed before the API release gate can pass. This is an upstream release
  dependency, not a reason to block local integration with a fixed revision.

## Runtime and ownership

One session owner commits model edits in order. GUI commands, CLI commands,
scripts, and add-ins use the same transaction path. The GUI drains a bounded
request queue and remains responsive while computation runs.

Workers receive immutable recipe snapshots and a document generation. They
return operation results, provenance, health, and presentation updates. The
session adopts a result only if its document and generation are still current.
Cancel work when possible; always discard stale results. Close, undo, and redo
must also invalidate stale jobs.

A cloned Martensite shape can share mutable storage. `Clone`, `Arc`, and even
`Send + Sync` do not prove that two evaluations can safely mutate the same
shape. Prove copy and mutation behavior in the early kernel check. Keep mutable
kernel state confined to one evaluation owner until that check passes. Do not
add unsafe thread traits to make a scheduling design compile.

Keep UI and viewer objects on their required thread. Define render-resource
retirement and shutdown order before moving work off that thread. No caller
may block the session thread while waiting for a callback that requires it.

## Documents and persistent identity

Keep the existing YAML recipe format and document kinds as the first storage
target: `.opd`, `.oad`, `.odd`, `.ord`, and project `.opj`. The current document
schema is version 2. Recompute geometry from recipes; do not make a serialized
Martensite shape the primary document representation.

Read the existing versioned `OKRK` reference-key envelope and supported legacy
keys. Replacing the kernel changes topology creation, so decoding the bytes
alone is insufficient. Prove that old references bind to the intended face,
edge, or vertex after recipe replay. Use operation provenance and semantic
feature roles. Never use pointer values, face iteration order, or nearest
geometry as persistent identity.

If an existing key payload cannot retain its meaning, define a versioned key
migration with an explicit old-to-new mapping. Preserve the original file and
report unresolved references. Do not silently rebind them or overwrite a file
that the port cannot fully represent. Full document compatibility remains an
exit gate, including projected geometry, attributes, mates, and dimensions.

## Viewport and UI direction

Use Martensite's Vulkan and visualization work as the target viewport. Retain
the current Dear ImGui workflow and UI assets while porting application logic
to Rust. A narrow native UI binding can remain; replacing the UI toolkit is a
separate decision. Reuse the existing window/UI stack only where it does not
bring back the Go application or its rendering engine.

Test native presentation, selection, overlays, resize, and ImGui composition
early. Martensite's current offscreen triangle rendering does not satisfy these
requirements. Record gaps in its capability ledger. Some Oblikovati features,
such as its OpenPBR workflows, may exceed the kernel's current renderer scope;
agree their ownership and acceptance before claiming visual parity. See the
[kernel integration plan](martensite.md).

## Decisions to record before dependent implementation

Create new Rust ADRs; link the Go ADRs they supersede and keep the old records.
Use a Rust-specific ADR namespace to avoid collisions with ongoing Go work.

| Decision | Due | Required evidence |
| --- | --- | --- |
| Rust application and Martensite ownership | R0 | Supersede Go-language and Go-kernel decisions; list the old production modules that this port replaces. |
| Kernel ownership, units, naming, and evaluation interface | R1 | Shape mutation tests, operation history, reference-key replay, and tolerance policy. |
| Native UI binding and Martensite presentation | R1V | Working native window, renderer composition design, OS requirements, and retained native dependency list. |
| Document and key migration | R3 | Old fixtures replayed through the new kernel without incorrect rebinding. |
| Lua runtime | R7 | Lua 5.1 behavior, instruction/time/memory limits, cancellation, and host containment tests. |
| Rust packaging and release transition | R9 | Reproducible platform packages, dependency pins, upgrade and rollback tests. |

Do not select a new YAML, Lua, window, or UI wrapper only from its name. Check
its current support, license notices, MSRV, and the required behavior in a small
test. The plan fixes the interfaces and acceptance criteria; those dependency
choices need evidence during their assigned milestone.

## Branch and work policy

Keep `rust` as the integration branch for this new application history. Fetch
upstream for reference. Do not merge `develop` with `--allow-unrelated-histories`;
that would restore the Go tree that this branch was created to replace.

Use small feature branches from `rust` and PRs back to `rust` once the branch
is published. Fetch and merge the current `rust` integration branch before
each feature PR. Keep granular commits. Kernel changes follow the separate
Martensite integration workflow; API changes follow the API workflow.

R0 creates the application tracking epic and milestone issues from this plan.
Use R0-R9 as stable planning IDs until real issue numbers exist. These IDs are
not claims that GitHub milestones or implementation issues already exist.
The tracker owns work status; these documents own design and acceptance.
