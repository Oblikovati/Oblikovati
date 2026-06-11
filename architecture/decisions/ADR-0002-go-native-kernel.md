# ADR-0002 — Go-native geometry kernel (no cgo in the kernel)

**Status:** accepted (user decision) · **Supersedes:** the original plan's
"thin managed API over a native kernel" assumption (M00/M07).

## Decision

Build the **B-rep geometry/modeling kernel in pure Go** — topology, analytic and
NURBS geometry, tessellation, and boolean/fillet/sweep operations — with **no cgo
in `kernel/` or `model/`**. The original plan assumed wrapping a native kernel
(Parasolid/ACIS/OCCT); we are not doing that.

## Why (per the user's choice)

- **"Built in Go" literally.** No native kernel dependency, no marshaling layer
  (the entire M00 interop milestone evaporates — see ADR-0006).
- **Trivial cross-compilation & testing.** A pure-Go kernel cross-compiles to all
  three platforms with `go build`, runs in CI without native toolchains, and is
  unit-testable without a GPU or OS.
- **Full control over identity.** Reference keys (topological naming) require the
  kernel to record each entity's *generative history*. Owning the kernel lets us
  bake identity in from day one instead of fighting an opaque third-party one
  (the single hardest, most schedule-critical problem — parametric-cad skill §7).

## Costs — acknowledged honestly

A production B-rep kernel is a **multi-year, XL effort**. The implementation plan
already flags the worst offenders: robust booleans (PBI-082), B-rep topology
(PBI-079), the constraint solver (PBI-075). Building these in Go from scratch is
the dominant risk of the whole project. **This ADR is accepted with eyes open and
a strict phasing discipline** (see [core/03](../core/03-geometry-kernel.md)):

1. **Phase A — analytic only.** Planes/cylinders/cones/spheres/tori + lines/arcs/
   circles/ellipses. Exact intersections. Enables extrude/revolve/hole on analytic
   geometry — covers a large fraction of real parts.
2. **Phase B — NURBS curves & surfaces** + tessellation quality. Enables sweep/loft.
3. **Phase C — robust general booleans & blends** (fillet/chamfer) with exact
   predicates for orientation robustness.
4. **Phase D — healing, tolerant modeling, advanced surfacing.**

Each phase is independently shippable (a part modeler that only does analytic
solids is still useful and demoable).

## Escape hatch

ADR-0008 confines cgo to the platform/render edge. *If* a kernel phase (likely
Phase C booleans) proves infeasible to ship in pure Go on schedule, the cgo
boundary already exists at the edge and a single `kernel/ops` package could be
backed by OCCT behind the same Go interface **without changing any caller** — the
kernel API is defined as Go interfaces precisely so this remains possible. We do
not plan for it, but we do not architecturally preclude it.

## Consequences

- `math/` must include **robust geometric predicates** (exact orientation/incircle
  via adaptive precision) — boolean robustness depends on it.
- `float64` throughout the kernel; `float32` only at the GPU tessellation boundary.
- The kernel is the long pole; the renderer, UI, parameters, and documents can all
  proceed in parallel against analytic-phase geometry.
