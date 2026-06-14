# Oblikovati — Modern Architecture (Go · Vulkan 1.3)

This is the modernized architecture for Oblikovati: a parametric, feature-based,
history-driven MCAD application rebuilt in **Go** with a **Vulkan 1.3** renderer,
targeting **Linux, macOS, and Windows**.

It is a *modernization* of two existing artifacts in this repo, not a fresh idea:

- **[GitHub milestones](https://github.com/Oblikovati/Oblikovati/milestones)** — the M00–M25 feature roadmap
  derived from the Inventor-class `Oblikovati.Contracts` surface. That defines
  *what* to build. This folder defines *how* to build it on a modern stack.
- The **`parametric-cad-architecture`** skill (the domain/model patterns) and the
  **`realtime-3d-app-architecture`** skill (the runtime/renderer patterns). This
  architecture is the concrete fusion of both on Go + Vulkan.

The original contracts are a faithful mirror of Autodesk Inventor's **COM**
automation API (variants, `_X : X` dual interfaces, `ObjectTypeEnum` RTTI,
connection-point events, OLE structured storage, in-process COM add-ins). COM is
Windows-only and hostile to Go. The throughline of this modernization:

> **Keep the domain model (the hard, correct ideas); replace the plumbing
> (the Windows/COM/marshaling ideas) with idiomatic Go and a modern GPU stack.**

What we **keep** (these are correct regardless of language — see the
`parametric-cad-architecture` skill): the model-as-evaluated-program, the
Definition→Add→Feature triangle, dimensioned parameters + dependency DAG,
persistent topological identity (reference keys), feature-history recompute with
health/suppression, definition-vs-occurrence instancing with context proxies,
transient-vs-persistent geometry, transactions/undo, extensible attributes.

What we **replace**:

| COM / C# idiom (old) | Go / Vulkan idiom (new) | Where |
|---|---|---|
| `ObjectTypeEnum` RTTI on every object | Go static types in-proc; stable `TypeID` only for persistence/RPC | [core/02](core/02-object-model-identity.md) |
| `object`/variant parameters | concrete types + generics in-proc; `any`/`oneof` only at RPC seam | [core/02](core/02-object-model-identity.md) |
| `_X : X` dual interfaces for versioning | Go interface composition; protobuf field versioning at the seam | [core/02](core/02-object-model-identity.md) |
| `Application` singleton + `Parent`/`Application` back-pointers | one **runtime mediator**, passed explicitly | [core/00](core/00-runtime-and-frame-loop.md) |
| `IEnumerable` collections, 1-based indexing | generic `Collection[T]`, 0-based | [core/02](core/02-object-model-identity.md) |
| connection-point events + `out HandlingCode` veto | typed observer registries; `Veto(reason)` via handler return | [core/06](core/06-transactions-and-events.md) |
| `TransactionManager` | command-pattern history (Apply/Revert) | [core/06](core/06-transactions-and-events.md) |
| OLE structured storage (`.ipt`) | zip-package container + columnar binary streams | [core/05](core/05-documents-persistence-identity.md) |
| native kernel via Coral marshaling | **pure-Go geometry kernel** (no cgo) | [core/03](core/03-geometry-kernel.md) |
| in-process COM add-ins | **in-proc registries + out-of-proc gRPC add-ins** | [core/07](core/07-extensibility.md) |
| (no renderer — Inventor owns its own) | **Vulkan 1.3 renderer**, scene graph, draw-call-as-data | [core/08](core/08-renderer-vulkan.md) |
| ribbon/browser/docking (COM UI) | **Dear ImGui (cimgui-go)** shell + custom viewport | [core/09](core/09-ui-imgui.md) |

## The decisions (ADRs)

The three defining choices, plus the derived ones, are recorded as ADRs:

| ADR | Decision |
|---|---|
| [0001](decisions/ADR-0001-go-language.md) | Go as the implementation language |
| [0002](decisions/ADR-0002-go-native-kernel.md) | **Go-native geometry kernel** (no cgo in the kernel) |
| [0003](decisions/ADR-0003-extensibility-hybrid-rpc.md) | **Hybrid extensibility**: in-proc registries + out-of-proc gRPC |
| [0004](decisions/ADR-0004-ui-imgui.md) | **Dear ImGui** shell + custom Vulkan viewport |
| [0005](decisions/ADR-0005-vulkan13-renderer.md) | Vulkan 1.3 (dynamic rendering, bindless, timeline semaphores) |
| [0006](decisions/ADR-0006-no-com-object-model.md) | Drop COM RTTI/variants/dual-interfaces |
| [0007](decisions/ADR-0007-async-recompute.md) | Decouple recompute from the frame loop (async, progressive) |
| [0008](decisions/ADR-0008-cgo-boundary.md) | Confine cgo to the platform/render edge (or eliminate via purego) |
| [0009](decisions/ADR-0009-sketch-solver.md) | Sketch solver: decompose + numeric (Newton), pure Go *(it2)* |
| [0010](decisions/ADR-0010-feature-recompute-model.md) | Feature recompute: rollback-replay over reference-keyed inputs *(it2)* |
| [0011](decisions/ADR-0011-assembly-positioning-solver.md) | One geometric solver for sketch (2D) + assembly (3D) *(it3)* |
| [0012](decisions/ADR-0012-exact-hidden-line.md) | Exact (analytic) hidden-line removal for drawing views *(it4)* |
| [0013](decisions/ADR-0013-ilogic-embedded-scripting.md) | iLogic rules via embedded scripting over the public API *(it4)* |
| [0014](decisions/ADR-0014-renderer-testability.md) | Renderer testability via a differential oracle hierarchy *(testing)* |
| [0015](decisions/ADR-0015-build-ci-and-test-tooling.md) | Build, CI/CD, and test tooling foundation *(tooling)* |
| [0016](decisions/ADR-0016-shared-library-addins-mcp-bridge.md) | **In-process shared-library add-ins (C ABI) + MCP automation bridge** — amends 0003 |
| [0017](decisions/ADR-0017-release-pipeline.md) | **Channel-based release pipeline** (nightly + stable; GUI head + CLI) — supersedes 0015's CD |
| [0029](decisions/ADR-0029-user-config-location.md) | **Unified per-user config location** (`~/.oblikovati`; `%AppData%\oblikovati` on Windows) via `oblikovati.org/userconfig` |
| [0030](decisions/ADR-0030-tolerant-nurbs-meshing.md) | Tolerant NURBS surface meshing (on-surface interior nodes + shared-edge stitching) |
| [0031](decisions/ADR-0031-embedded-document-resources.md) | Imported files are embedded in the document as a root `resources` dictionary (UUID-keyed) |
| [0032](decisions/ADR-0032-blender-theme-file-format.md) | Blender theme XML as the theme file format |
| [0033](decisions/ADR-0033-icon-color-roles.md) | Icon color roles (primary / secondary / tertiary / background) |
| [0034](decisions/ADR-0034-per-document-type-file-extensions.md) | Per-document-type file extensions, and a project file |
| [0035](decisions/ADR-0035-assembly-machining-features.md) | **Assembly machining features**: occurrence-relative references + a serialized feature program |
| [0036](decisions/ADR-0036-content-agnostic-sketch-host.md) | The sketch environment hosts on a content-agnostic `sketchHost` interface (part or assembly) |

> Note: ADRs 0018–0028 exist under `decisions/` but predate the last index refresh; this
> table is being backfilled separately.

## The modern stack at a glance

```
                          ┌──────────────────────────────────────────────┐
   add-ins (link /api,  ──┤  api/  — the public contract (Apache-2.0):    │  separate module
   never /source)         │  types · contract · wire · client (ADR-0018)  │
                          └───────────────────────┬──────────────────────┘
                                                   │ C ABI in-proc today (ADR-0016) · gRPC later
 ┌────────────────────────────────────────────────▼─────────────────────────────────┐
 │ app/  — assembled application: wires the runtime mediator, picks build tags        │
 ├──────────────┬───────────────┬───────────────┬──────────────┬─────────────────────┤
 │ ui/ (ImGui)  │ renderer/      │ scene/        │ registry/    │ addins/ (rpc host)  │
 │ shell+panels │ Vulkan 1.3     │ viewport graph│ self-register│ plugin lifecycle    │
 ├──────────────┴───────────────┴───────────────┴──────────────┴─────────────────────┤
 │ runtime/  — the mediator: frame loop, ordered phases, job pool, clock, schedulers  │
 ├────────────────────────────────────────────────────────────────────────────────────┤
 │ model/  — documents · componentdef · features · parameters · sketch · identity      │
 │ kernel/ — geom (curves/surfaces/nurbs) · topo (B-rep) · ops (boolean/fillet/tess)    │  PURE GO
 │ math/   — vec/mat/quat (float64) · robust predicates                                 │  (no cgo)
 ├────────────────────────────────────────────────────────────────────────────────────┤
 │ platform/ (window,input,fs)  ·  persistence/ (zip+binary)  ·  build/ (tags)          │
 └────────────────────────────────────────────────────────────────────────────────────┘
   cgo (or purego) confined here ───────────────┘
```

## Core architecture docs (iteration 1 — the foundations)

This first pass deliberately covers only the **core** (the equivalents of plan
milestones M00–M05 plus the new renderer/frame-loop), reevaluated calmly. Later
iterations layer the modeling milestones (M06+) onto this spine.

| # | Doc | Modernizes (plan) |
|---|-----|-------------------|
| 00 | [Runtime mediator & frame loop](core/00-runtime-and-frame-loop.md) | new (realtime-3d §1–2) + M04 |
| 01 | [Module layout & build gating](core/01-module-layout.md) | new (realtime-3d §14) |
| 02 | [Object model, types & identity](core/02-object-model-identity.md) | M00 |
| 03 | [Go-native geometry kernel](core/03-geometry-kernel.md) | M01, M07 |
| 04 | [Parameters & expression engine](core/04-parameters-expressions.md) | M02 |
| 05 | [Documents, persistence & reference keys](core/05-documents-persistence-identity.md) | M03 |
| 06 | [Transactions (commands) & events](core/06-transactions-and-events.md) | M04 |
| 07 | [Extensibility: registries + gRPC add-ins](core/07-extensibility.md) | M05 |
| 08 | [Vulkan 1.3 renderer & scene graph](core/08-renderer-vulkan.md) | new (realtime-3d §3–6) |
| 09 | [UI shell: Dear ImGui](core/09-ui-imgui.md) | M05 |

Plus the [COM→Go idiom cheatsheet](mapping/com-to-go-cheatsheet.md).

## Modeling spine docs (iteration 2 — part modeling)

The part-modeling spine layered on the core. See [modeling/README.md](modeling/README.md).

| # | Doc | Modernizes (plan) |
|---|-----|-------------------|
| 00 | [Sketch & constraint solver](modeling/00-sketch-and-solver.md) | M06 |
| 01 | [Feature-history engine](modeling/01-feature-engine.md) | M07-F04, M08-F01 |
| 02 | [Sketched & work features](modeling/02-sketched-work-features.md) | M08 |
| 03 | [Dress-up & pattern features](modeling/03-dressup-patterns.md) | M09 |
| 04 | [Surfacing & freeform](modeling/04-surfacing-freeform.md) *(it3)* | M10 |
| 05 | [Sheet metal](modeling/05-sheet-metal.md) *(it3)* | M13 |

## Assembly docs (iteration 3 — multi-part)

The multi-part domain: instancing, context proxies, the unified solver, reps. See
[assembly/README.md](assembly/README.md).

| # | Doc | Modernizes (plan) |
|---|-----|-------------------|
| 00 | [Instancing & context proxies](assembly/00-instancing-and-proxies.md) | M11 |
| 01 | [Constraints, joints & drive](assembly/01-constraints-joints.md) | M12-F01/F02/F03 |
| 02 | [Representations & model states](assembly/02-representations.md) | M12-F04/F05 |

## Application & output docs (iteration 4 — drawing, automation, viz, interop, analysis)

The capability domains on the spine. See [apps/README.md](apps/README.md).

| # | Doc | Modernizes (plan) |
|---|-----|-------------------|
| 00 | [Drawing & documentation](apps/00-drawing-documentation.md) | M14 |
| 01 | [Design automation: iPart & iLogic](apps/01-design-automation.md) | M15 |
| 02 | [Visualization, appearances & presentations](apps/02-visualization-presentation.md) | M16 |
| 03 | [Interoperability & translation](apps/03-interoperability.md) | M17 |
| 04 | [Analysis, measurement & simulation](apps/04-analysis-simulation.md) | M18 |

## Status — all iterations complete ✅

- **Iteration 1:** core foundations (`core/`, ADR 0001–0008). ✅
- **Iteration 2:** modeling spine (`modeling/00–03`, ADR 0009–0010) — sketch + solver
  (M06), feature engine & the Definition→Add→Feature triangle (M07–M09). ✅
- **Iteration 3:** surfacing (`modeling/04`, M10), sheet metal (`modeling/05`, M13),
  assembly (`assembly/`, ADR 0011) — instancing + proxies (M11), constraints/joints &
  representations (M12). ✅
- **Iteration 4:** apps (`apps/`, ADR 0012–0013) — drawing & hidden-line (M14),
  automation/iLogic (M15), visualization/presentations (M16), interop (M17),
  analysis/simulation incl. contact & dynamics (M18). ✅

**All 19 roadmap milestones (M00–M18) now have a modern-architecture
treatment on the Go + Vulkan stack.** The recurring result: after the iteration-1
core was laid down, each later domain was overwhelmingly *reuse* — only six genuinely
new engines were needed across the whole modeler (geometry kernel, reference keys,
sketch/assembly solver, feature engine, hidden-line removal, rules runtime), and every
one sits behind a clean boundary the core established.

## Testability (cross-cutting — built for LLM-authored code)

A senior-test-engineer pass over the architecture, since Claude authors and debugs
most of the code. The core is already highly testable (cgo-free, GPU-free, pure
recompute, command/event seams — the testability ledger is in
[testing/README.md](testing/README.md)); the **renderer** is the hard case and gets a
deep treatment, because its bugs are visual and an LLM cannot see the output.

| Doc | Scope |
|---|---|
| [testing/README.md](testing/README.md) | strategy, testability ledger, the pyramid, CI tiers, the LLM-signal principle |
| [testing/00](testing/00-renderer-oracle-pipeline.md) | **renderer oracle hierarchy** — analytic · CPU-reference · metamorphic · Blender; software-Vulkan determinism; numeric diff reports |
| [testing/01](testing/01-core-and-integration-testing.md) | pure-Go unit/property/metamorphic tests, reference-key tests, round-trip goldens, gRPC dogfood integration |
| [testing/02](testing/02-performance-ui-and-conformance.md) | performance/allocation budgets & soak, ImGui UI-logic tests, add-in gRPC conformance/isolation |

The renderer doc [core/08](core/08-renderer-vulkan.md) was revised to make
testability structural (three backends, the GPU line, per-pass AOV capture, CPU
reference shading, determinism mode, validation-as-gate) per
[ADR-0014](decisions/ADR-0014-renderer-testability.md). **Principle:** convert "does
it look right?" (an LLM can't judge) into "does it match the oracle within tolerance,
and which pass/region diverged?" (actionable) — with the fast, exact, dependency-free
oracles (analytic/CPU-reference/metamorphic) carrying the inner loop and Blender as
the high-fidelity ground-truth backstop.
