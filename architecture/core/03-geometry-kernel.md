# Core 03 — Go-native geometry kernel

*Modernizes M01 (transient geometry) + M07 (B-rep kernel). Implements ADR-0002.
This is the project's long pole — phased deliberately.*

## Shape of the kernel

Three layers, all pure Go, cgo-free:

```
kernel/geom/   value geometry: curves, surfaces, NURBS, evaluators   (≈ M01)
kernel/topo/   B-rep topology: Body/Shell/Face/Loop/Edge/Vertex      (≈ M07-F01)
kernel/ops/    operations: boolean, tessellate, fillet, sweep, heal  (≈ M07-F03)
math/predicate/exact arithmetic predicates (orientation, in-circle)
```

`geom` is the modern form of the COM `TransientGeometry` factory: ownerless,
immutable value types (parametric-cad skill §6). But Go gives us **value semantics
for free** — a `geom.Point` is a struct passed by value, not a heap COM object —
so the "transient" category is just plain structs:

```go
package geom
type Point struct{ X, Y, Z float64 }       // value type, no identity, no factory needed
type Line struct{ Origin Point; Dir Vec3 }
type Arc struct{ Center Point; Normal, Ref Vec3; Radius, Start, Sweep float64 }
type NURBSCurve struct{ Degree int; Ctrl []Point4; Knots []float64 }
type Plane struct{ Origin Point; Normal Vec3 }
// …cylinder, cone, sphere, torus, NURBSSurface
```

No `CreatePoint2d` factory, no marshaling — that whole apparatus existed to cross
the COM boundary, which no longer exists. Evaluators are methods/functions:
`crv.PointAt(t)`, `srf.NormalAt(u,v)`, `srf.ClosestPoint(p)`.

## B-rep topology with identity baked in

The topology is the standard Body→Shell→Face→Loop→Edge→Vertex graph with full
adjacency. The **non-negotiable design rule** (parametric-cad skill §7): every
topological entity records *how it was generated*, because reference keys derive
from generative history, not pointers.

```go
package topo
type Face struct {
    Surface geom.Surface
    Loops   []*Loop
    body    *Body
    lineage Lineage   // ← "end cap of feature F", "side wall of extrude E, profile edge p"
}
```

`Lineage` is the seed for the reference key (core/05). Owning the kernel is what
makes this possible — an opaque third-party kernel would force us to reverse-
engineer identity. This is the core justification for ADR-0002.

## Robust predicates — booleans live or die here

Boolean robustness is the classic kernel killer (PBI-082, flagged XL). The
mitigation is **exact geometric predicates** with adaptive precision (Shewchuk-
style): orientation and in-circle/in-sphere tests that never return a wrong sign
near degeneracies. `math/predicate` provides these in pure Go (they are integer/
expansion arithmetic — no cgo needed). `ops` uses them for all branching decisions
(which side of a plane, do these intersect) so topology stays consistent.

## Concurrency

The kernel is **pure functions over immutable inputs** (ADR-0007): `Extrude(profile,
extent) (Body, error)` allocates and returns a new body; it never mutates shared
state. This makes:
- recompute of independent DAG branches **parallel** (worker pool),
- tessellation of multiple bodies **parallel**,
- the whole kernel **trivially unit-testable** headless in CI.

Tessellation emits `float32` triangle meshes + edge polylines for the renderer
(core/08) at a chordal tolerance — the one place `float64`→`float32` narrowing
happens.

## Phasing (the risk-management plan — ADR-0002)

| Phase | Geometry | Unlocks (plan features) | Notes |
|---|---|---|---|
| **A** | analytic surfaces/curves only; exact intersections | extrude, revolve, hole, chamfer-on-analytic, planar sketch | a useful analytic solid modeler; demoable early |
| **B** | NURBS curves & surfaces; quality tessellation | sweep, loft, splines, fillet (rolling-ball) | |
| **C** | robust general booleans & blends; exact predicates everywhere | combine/split, complex fillets, shell | the hardest phase |
| **D** | healing, tolerant modeling, advanced surfacing | import healing (M17), surfacing (M10) | |

Each phase ships independently. The renderer, parameters, documents, sketch UI,
and the whole app shell are built against **Phase A** geometry in parallel — the
kernel does not block the rest of the program, only the *kinds of features* that
work.

## Escape hatch (restated)

`ops` is a Go **interface** (`ops.Booleaner`, `ops.Filleter`). The default
implementation is pure Go. If Phase C slips, a single alternative implementation
backed by OCCT-via-cgo could satisfy the same interface **without touching a single
caller** (ADR-0002/0008). We do not plan for this; the interface seam simply keeps
the option open and costs nothing.

## What disappears vs. the COM plan

- The **interop/marshaling** of M00 — gone (value types are Go structs).
- The `TransientGeometry`/`TransientObjects` **factories** — gone (just structs and
  constructors); their *purpose* (a discoverable construction point) is served by
  the `geom` package surface.
- `ref byte[]` plumbing for reference keys — replaced by Go `[]byte` + typed codecs.
