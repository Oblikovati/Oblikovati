# Modeling 00 — Sketch & constraint solver

*Modernizes M06 (2D/3D sketching, constraints, solver, profiles/paths). Implements
ADR-0009. Cross-ref: parameters (core/04), reference keys (core/05), overlay
graphics (core/08), reflection inspector (core/09).*

A sketch is a **constrained 2D program** that resolves to curves and emits
`Profile`/`Path` — the clean boundary features consume (parametric-cad §10). The
solver lives entirely behind that boundary; the kernel never sees it.

## Sketch container & entities

```go
package sketch
type Sketch struct {
    id       ID
    plane    Ref            // host plane/face/work-plane (reference-keyed, core/05)
    ents     Collection[Entity]
    geomCons []GeometricConstraint
    dimCons  []DimensionalConstraint
    solved   bool
    health   Health
}
```

- Entities are structs with **constrainable points**: `Line{A,B *Point}`,
  `Arc{Center *Point; ...}`, `Circle{Center *Point; Radius}`, `Spline{Pts []*Point}`.
  A `*Point` is the solver variable carrier; entities share points (a shared
  endpoint *is* a coincidence, structurally).
- Each entity has a **session `ID`** (for constraint references and selection) and is
  persisted; construction vs. normal geometry is a flag.
- **2D↔3D mapping**: the sketch carries its plane transform; `ToModel(p2d)` /
  `ToSketch(p3d)`. Projected/reference geometry from the model resolves through
  reference keys and updates associatively.

## Constraints

```go
type GeometricConstraint interface{ Residuals(v *VarMap) []float64; Refs() []ID }
// Coincident, Collinear, Parallel, Perpendicular, Tangent, Concentric, Equal,
// Symmetry, Horizontal, Vertical, Fix, Smooth — each a residual contribution.

type DimensionalConstraint struct {
    kind   DimKind          // distance | angle | radius | diameter | arcLength
    refs   []ID
    param  param.ParamID    // ← OWNS a parameter (core/04): the dimension's value is an expression
    driven bool             // driven (reports) vs driving (constrains)
}
```

- **Dimensional constraints own parameters** (parametric-cad §4): the dimension's
  value is an editable expression in the parameter DAG. Editing it flows: param edit
  → param DAG dirty → sketch re-solve → owning feature dirty → async recompute.
- **Driven** dimensions report the solved value without adding a constraint.
- **Inference** while sketching (coincident/tangent/parallel/horizontal/vertical)
  proposes constraints with glyphs drawn in the **overlay pass** (core/08); applied
  on commit. Inference is a ranked heuristic, not part of the solver.

## The solver (ADR-0009)

`sketch.Solve()` turns entities + constraints into a system of residual equations
over point/scalar variables and solves it:

```
Solve(s *Sketch):
  vars        := collect DOF (entity points/radii) → VarMap
  residuals   := concat(geomCons, dimCons) residual functions
  graph       := constraint graph over vars
  dof, status := analyzeDOF(graph)           # 0 / under / over / redundant
  clusters    := decompose(graph)            # solvable sub-systems in dependency order
  for c in clusters (warm-started from current positions):
      newton(c.residuals, c.vars)            # NR + LM damping, capped iters
  s.health = healthFrom(status, convergence) # never NaN/hang: "couldn't solve" → sick
```

- **Decompose + numeric** (ADR-0009): small clusters, warm-started, solved in order.
- **DOF analysis is a byproduct of the graph** → reports 0-DOF (fully constrained),
  under-constrained free vars (visualized), and over/redundant constraints (flagged,
  the offending constraint rejectable).
- **Interactive drag**: dragging a point fixes it temporarily and re-solves only the
  affected clusters, warm-started — cheap enough for 60fps in the interaction phase
  (core/00). No branch jumping mid-drag (bias to current configuration).
- **Pure Go**, deterministic; a whole-system Newton fallback ships first, with
  decomposition added behind the same `Solve()` API later (de-risking, ADR-0009).

## Profiles & paths — the feature boundary

The only thing the sketch exports to features:

```go
type Profile struct{ Loops []Loop }       // outer + inner loops, from region detection
type Path    struct{ Chain []geom.Curve } // connected curve chain for sweep/loft rails
```

- **Region detection** finds closed loops from solved geometry, classifies
  outer/inner, and yields `Profile`s (multi-region supported). Open profiles are
  allowed for surfaces, rejected for solids — enforced at the feature, not here.
- A `Profile`/`Path` handed to a feature is captured as a **`Ref`** (reference key
  into the sketch) so the feature re-resolves it every recompute (ADR-0010). Edit
  the sketch → the consuming feature dirties and rebuilds.

## 3D sketches

Same model with 3D points and the 3D constraint variants (`*Constraint3D`); the
solver generalizes (3 vars/point). Used for sweep/pipe paths. The decomposition and
Newton core are dimension-agnostic.

## Why this is cleaner than the COM original

| COM | Here |
|---|---|
| `SketchLine`/`SketchArc` COM objects + `Add*` | structs with shared `*Point`s; coincidence is structural |
| `DimensionConstraint` variant value | dimension **owns** a `param.Quantity` (typed, in the DAG) |
| solver hidden in the monolith | `model/sketch` behind a `Profile`/`Path` boundary, unit-testable headless |
| `Profile` as COM object graph | `Profile{Loops}` value + a `Ref` for associativity |
| inference baked into UI | ranked heuristic → overlay glyphs (core/08), separate from solve |

The solver and parameter DAG meet only through `Quantity` values; the solver and
kernel never meet at all — exactly the decoupling the parametric-cad skill demands.
