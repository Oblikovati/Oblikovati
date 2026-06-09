# Core 01 — Module layout & build gating

*Applies realtime-3d skill §14 (build gating), §15 (conventions); adapts the
suggested layout (§"module layout") to a parametric CAD domain.*

## Package layout

```
oblikovati/
  cmd/
    oblikovati/        # main: assembles a Runtime (window) — build tag `app`
    oblikovati-cli/    # headless: batch/translate/thumbnail — build tag `headless`
  runtime/             # the mediator, frame loop, schedulers, clock, worker pool
  build/               # compile-time constants: build.Debug, build.Editor, build.Profile

  math/                # vec2/3/4, mat3/4, quat (float64), AABB/OBB,
                       #   ROBUST PREDICATES (exact orient/incircle) — kernel depends on it
  kernel/              # PURE GO geometry/modeling kernel (ADR-0002), cgo-free
    geom/              #   analytic curves & surfaces, NURBS, evaluators
    topo/              #   B-rep: Body/Shell/Face/Loop/Edge/Vertex + adjacency + ref-key hooks
    ops/               #   boolean, fillet/chamfer, sweep/loft, tessellate, heal
    predicate/         #   exact arithmetic shims used by ops

  model/               # the DOMAIN (cgo-free): "model = evaluated program"
    doc/               #   Document, Workspace (open docs), document refs
    compdef/           #   Part/Assembly component definitions (content containers)
    feature/           #   feature history engine + Definition/Feature pairs (the triangle)
    sketch/            #   sketch entities, constraints, the Go-native solver
    param/             #   parameters, units, expression engine, dependency DAG
    identity/          #   reference keys (topological naming), TypeID registry hooks
    attr/              #   attributes & properties (extensible metadata)

  scene/               # VIEWPORT scene graph: entity + transform hierarchy + dirty flags
  renderer/            # Vulkan 1.3 backend (cgo/purego edge), passes, caches, draw queue, picking
    backend/           #   *_vulkan.go ; null/offscreen backend for tests & thumbnails
  platform/            # window/input/fs/threading/profiler — the cgo (or purego) edge (ADR-0008)
  ui/                  # ImGui shell (ADR-0004): panels, browser, inspector (reflection), tables

  addin/               # host-side API: JSON method router, op registry, dispatch (serves api/wire)
  addins/              # out-of-proc add-in host: discovery, lifecycle, supervision, sandbox
  registry/            # self-registration: features, workspaces, commands, types, translators
    types/             #   stable TypeID registry (persistence/RPC identity, ADR-0006)
  command/             # command-pattern history (undo/redo), ADR-0006/core-06
  event/               # typed event bus (before/after, veto), core-06
  persistence/         # document container (zip package) + columnar binary serialization

  internal/...         # non-public helpers
```

### Dependency direction (must stay acyclic)

```
math → kernel → model → {scene, api, persistence}
                         scene → renderer → platform
                         model → command, event, identity, registry
                         ui → runtime → (everything, as the mediator)
```

`math/kernel/model` never import `renderer/ui/platform`. The domain does not know
the GPU exists. This is what keeps the kernel cgo-free and headless-testable.

### The `/api` contract module (separate, Apache-2.0)

The public API is its own module, **`oblikovati.org/api`** at the repo root
(`/api`, sibling to `/source`), licensed Apache-2.0 so add-ins — including
closed-source ones — can build against it ([ADR-0018](../decisions/ADR-0018-apache-api-contract-module.md)).
It has four packages: `types` (enums/value types — the canonical definitions
`/source` aliases), `contract` (in-proc Go interfaces `/source` satisfies), `wire`
(method-name constants + JSON DTOs), and `client` (a `Transport` + typed client for
out-of-runtime add-ins). The dependency flows **only toward** `/api`: `/source`
requires it (via a `replace` directive, like `/source/head` does), and `/api` never
imports `/source` (CI-enforced).

## Build-time gating (realtime-3d §14)

- **Feature/mode flags as build tags surfaced through `build/`**:
  `build.Debug` (validation layers, tracing, asserts), `build.Profile` (CPU/GPU
  capture), `build.Editor` (extra tooling). Code branches on these *constants*;
  dead branches compile out.
- **One entry point, two faces**: `cmd/oblikovati` (windowed Runtime) vs
  `cmd/oblikovati-cli` (headless Runtime, null renderer) — they share the runtime
  and differ only in the top layer + a tag. The headless build is `CGO_ENABLED=0`.
- **Platform code via filename suffix**: `window_linux.go`, `window_darwin.go`,
  `window_windows.go`, and `*_cgo.go` / `*_purego.go` for the renderer edge
  (ADR-0008). No runtime `if runtime.GOOS == …` in hot code.
- **On/off pairs** for optional subsystems (`addins_on.go` / `addins_off.go` gated
  by a tag) exporting the same symbols, so a build without the add-in host still
  compiles callers unchanged.

## Conventions (realtime-3d §15)

- **Errors as typed values**; resolve/fall back near the source. Modeling failures
  are **health state on the feature**, never panics (parametric-cad skill §2).
- **Interfaces only when justified** — the kernel `ops` boundary (so an OCCT
  fallback stays possible, ADR-0002), the renderer backend, the platform edge, the
  registry contracts. Most domain code is concrete structs (clearer, faster).
- **`NotYetImplemented(issueID)`** stubs instead of `TODO`/`FIXME`, so gaps are
  runtime-visible and grep-able (the kernel will have many during phasing).
- **One structured logging facade** (`slog`) on the runtime; no `fmt.Println`.
- **Document every exported symbol**, std-lib style; comment *why*.
- **Configurable scalar** (realtime-3d §13): the kernel/math use `float64`
  (CAD precision); the renderer narrows to `float32` only at the tessellation/GPU
  boundary. A single `type Scalar = float64` alias makes the precision explicit and
  flippable.
