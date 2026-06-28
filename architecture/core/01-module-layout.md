# Core 01 — Module layout & build gating

*Applies realtime-3d skill §14 (build gating), §15 (conventions); adapts the
suggested layout (§"module layout") to a parametric CAD domain.*

## Package layout

```
oblikovati/            # the GPL-v2 application module (this repo root)
  cmd/
    oblikovati/        # main: assembles the session + windowed head
    oblikovati-cli/    # headless: batch/translate/thumbnail, CGO_ENABLED=0

  build/               # compile-time constants: build.Debug, build.Editor, build.Profile
  math/                # vec2/3/4, mat3/4, quat (float64), AABB/OBB,
                       #   ROBUST PREDICATES (exact orient/incircle) — kernel depends on it
  kernel/              # PURE GO geometry/modeling kernel (ADR-0002), cgo-free
    geom/              #   analytic curves & surfaces, NURBS, evaluators, exact-predicate shims
    topo/              #   B-rep topology: Body/Shell/Face/Loop/Edge/Vertex + adjacency + ref-keys
    brep/              #   solid builders + curved-boolean/half-space dispatch over topo
    ops/              #   boolean, fillet/chamfer, sweep/loft, tessellate, mass-props, heal
    fit/ hlr/ subd/    #   curve/surface fitting, hidden-line removal, subdivision surfaces
    exchange/          #   STEP/mesh import-export codecs
    diag/ geomapi/     #   kernel diagnostics channel; the in-proc geometry API facade

  model/               # the DOMAIN (cgo-free): "model = evaluated program"
    doc/               #   Document, Workspace (open docs), document refs
    compdef/           #   Part/Assembly component definitions (content containers)
    feature/           #   feature history engine + Definition/Feature pairs (the triangle)
    sketch/            #   sketch entities, constraints (solver lives in /solve)
    param/             #   parameters, units, expression engine, dependency DAG
    identity/          #   reference keys (topological naming), TypeID registry hooks
    attr/              #   attributes & properties (extensible metadata)
    assembly/ occurrence/ drawing/ sheetmetal/ material/ style/ … # the rest of the domain

  solve/               # the Go-native constraint solver (Gauss-Newton/Levenberg-Marquardt)
  scene/               # VIEWPORT scene graph: entity + transform hierarchy + dirty flags
  renderer/            # render backend abstraction: passes, caches, draw queue, picking
  command/             # command-pattern history (undo/redo), ADR-0006/core-06
  event/               # typed event bus (before/after, veto), core-06
  persistence/         # document container (.obk YAML package) + serialization

  addin/               # host-side API: JSON method router (addin/router, serves api/wire), op registry
  addincat/            # add-in catalogue client (addins.oblikovati.org)
  app/                 # the session/mediator: orchestrates documents, commands, environments,
                       #   the frame tick — cgo-free and headless-testable
  script/              # embedded Lua scripting (ADR-0028)

  head/                # the cgo Vulkan 1.3 + Dear ImGui windowed shell — a separate submodule so
                       #   the cgo build never touches the headless-tested core (ADR-0008)
    viewport/          #   the 3D viewport (Vulkan render loop)
    ui/                #   ImGui shell: ribbon, browser, inspector, panels
    addins/            #   the in-proc add-in host: discovery, lifecycle, C-ABI load (ADR-0016)
```

### Dependency direction (must stay acyclic)

```
math → kernel → model → {scene, api, persistence, solve}
                         scene → renderer
                         model → command, event, identity
                         app → (model, command, event, addin, scene) as the session mediator
                         head → app + renderer + scene (the cgo windowed shell on top)
```

`math/kernel/model` never import `renderer`, `scene`, or `head` (the cgo edge). The
domain does not know the GPU exists. This is what keeps the kernel cgo-free and
headless-testable. (`model/clientgraphics` is the one place this is currently bent —
Oblikovati/Oblikovati#1500 relocates it out of the pure domain.)

### The `/api` contract module (separate sibling repo, Apache-2.0)

The public API is its own module, **`oblikovati.org/api`**, in a sibling repo
(`../Oblikovati.API`, tied in for local dev by the `go.work` replace), licensed
Apache-2.0 so add-ins — including closed-source ones — can build against it
([ADR-0018](../decisions/ADR-0018-apache-api-contract-module.md)).
It has four packages: `types` (enums/value types — the canonical definitions this
GPL module aliases), `contract` (in-proc Go interfaces this module satisfies), `wire`
(method-name constants + JSON DTOs), and `client` (a `Transport` + typed client for
out-of-process add-ins). The dependency flows **only toward** `api`: this module
requires it (via the `go.work` replace, as `head` does too), and `api` never
imports this module (CI-enforced).

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
