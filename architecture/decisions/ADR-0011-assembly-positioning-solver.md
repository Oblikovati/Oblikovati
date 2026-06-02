# ADR-0011 — One geometric constraint solver for sketch (2D) and assembly (3D)

**Status:** accepted · **Context:** assembly constraints (mate/flush/angle/insert)
and joints (M12-F01/F02) must position rigid components. We already built a 2D
constraint solver for sketches (ADR-0009). Do we build a second, unrelated solver?

## Decision

**No second solver.** Generalize the sketch solver's **decompose + numeric core**
(ADR-0009) into one reusable engine, `solve/`, parameterized by the *variable kind*:

- **Sketch** → variables are 2D points + scalars (2 DOF/point).
- **Assembly** → variables are **rigid-body placements**, 6 DOF each (position +
  orientation, orientation carried as a unit quaternion with a normalization
  constraint).

Both reduce to the same shape: a vector of DOF variables, a set of residual
equations `f(vars)=0`, graph decomposition into clusters, Newton–Raphson + LM
damping per cluster, warm-started from the current configuration.

```go
package solve
type System struct{ vars []Var; residuals []Residual; graph Graph }
func (s *System) Solve(ctx) Status      // DOF analysis + decompose + numeric (ADR-0009)
// sketch and assembly each build a System from their own constraints; the core is shared
```

## Why one solver

- **The math is the same family.** A "mate" (two faces coincident) and a sketch
  "coincident" are both residual equations; an "angle" constraint is an angle
  residual in 2D or 3D. The decomposition, DOF analysis, and damped-Newton loop are
  identical — only the variable parameterization and the residual library differ.
- **Diagnostics for free, again.** Over/under/redundant-constraint detection and DOF
  reporting (which assemblies need badly — "this component is under-constrained")
  fall out of the same graph analysis the sketch uses.
- **One hard thing to get right, not two.** Robustness, determinism, warm-start,
  branch stability are solved once. Halves the XL risk.

## Joints as DOF-reduction priors (not a separate mechanism)

The newer **joint** model (rotational/slider/cylindrical/ball/planar) is *not* a
parallel solver. A joint is expressed as a **reduced-DOF parameterization** of the
relative placement of two components: a rotational joint says "the relative
transform is a rotation about this axis by angle θ" — i.e. it replaces 6 free DOF
with 1 driven variable θ. The same `solve.System` consumes it; the joint just
contributes a tightly-parameterized variable block plus limit inequalities.

So constraints and joints unify: both shape the variable/residual set the one solver
consumes. Drive/animation simply sweeps a joint/constraint's driven variable through
a range and re-solves (warm-started) per step.

## Costs / mitigations

- **3D orientation is trickier than 2D** (quaternion normalization, gimbal-free
  Jacobians) → carry orientation as quaternions with a normalization residual;
  derive analytic Jacobians for the common mate/flush/angle/insert residuals,
  finite-difference the rest.
- **Grounded components** anchor the system (fixed variables) exactly like a fixed
  sketch point — reuses the same "fixed var" path.
- **Large assemblies** → decomposition keeps Newton systems small (per cluster of
  mutually-constrained components); whole-assembly solves are avoided. Cluster
  solves run on the worker pool (core/00).

## Consequences

- `model/sketch` (ADR-0009) and `model/assembly` both depend on `solve/`; neither
  depends on the other.
- Assembly positioning is **async/cancellable** like part recompute (ADR-0007):
  dragging a component or editing a constraint enqueues a re-solve; the view shows
  the last-good positions until it completes.
- Contact and dynamic simulation (M12-F05, M18) are *different* problems (inequality/
  time-integration) and get their own treatment later — this ADR is about *static
  positioning* only.

See [assembly/01](../assembly/01-constraints-joints.md).
