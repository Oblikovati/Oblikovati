# ADR-0009 — Sketch constraint solver: decompose + numeric

**Status:** accepted · **Context:** the 2D sketch solver (plan M06-F05, PBI-075) is
flagged XL — second only to the kernel and reference keys in difficulty. It must be
pure Go (ADR-0002), deterministic, and fast enough to re-solve while dragging.

## Decision

Build a **decomposition + numerical** 2D constraint solver:

1. **Model the sketch as a system of equations.** Each entity contributes DOF
   variables (a point = 2 vars; a line = its two endpoints; a circle = center +
   radius). Each geometric/dimensional constraint contributes residual equations
   (`f(vars) = 0`): coincidence, distance, parallelism, tangency, angle, etc.
2. **Decompose before solving.** Analyze the constraint graph to (a) compute the
   global DOF and detect over/under/redundant constraints, and (b) **partition into
   solvable clusters** (graph-based / generalized maximum-matching decomposition).
   Solving many small clusters in dependency order beats one huge Newton system —
   it is faster, better-conditioned, and localizes failures.
3. **Solve each cluster numerically** with **Newton–Raphson** on the residual
   vector (Jacobian by analytic derivatives where cheap, finite-difference
   fallback), with line search / Levenberg–Marquardt damping for robustness near
   singularities.
4. **Warm-start from current positions** so an interactive drag re-solves in a few
   iterations from the last solution — essential for 60fps dragging.

## Why this approach

- **Pure Go, no dependencies** — it is linear algebra + graph theory, no native
  solver to wrap. Honors ADR-0002 and cross-compiles trivially.
- **Decomposition gives the diagnostics users need** (DOF count, *which* constraints
  are redundant/conflicting) almost for free — these fall out of the graph analysis,
  not the numeric solve.
- **Deterministic & debuggable** — small clusters with warm starts converge
  predictably; we can log per-cluster residuals.
- **Interactive-friendly** — warm-started incremental re-solve is cheap; only the
  affected clusters re-solve on a drag.

## Costs / mitigations

- **Robustness near degeneracies** (tangency, near-parallel) → damping (LM),
  witness configurations to disambiguate solution branches, and capped iterations
  with a "could not solve" health result rather than a hang/NaN.
- **Branch selection** (a circle tangent to a line has two sides) → bias toward the
  warm-start/current configuration; never jump branches mid-drag.
- **Building decomposition is real work** → ship a **pure-Newton whole-system
  fallback** first (correct but slower for big sketches), then add decomposition as
  a performance/diagnostics layer behind the same `Solve()` API. Most early sketches
  are small enough that the fallback is fine — this de-risks the schedule.

## Boundary discipline (parametric-cad skill §10)

The solver lives entirely inside `model/sketch` behind a **profile boundary**: it
consumes sketch entities + constraints and emits solved positions; the kernel
(`kernel/`) never sees the solver, and the solver never sees the B-rep. They evolve
independently. The sketch's output to features is `Profile`/`Path`, nothing else.

## Consequences

See [modeling/00](../modeling/00-sketch-and-solver.md). Dimensional constraints own
parameters (core/04), so the solver and the parameter DAG meet only through
`Quantity` values — a dimension change flows: parameter edit → param DAG dirty →
sketch re-solve → owning feature dirty → async recompute (ADR-0007).
