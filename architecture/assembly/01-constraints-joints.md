# Assembly 01 — Constraints, joints & drive

*Modernizes M12-F01 (constraints), F02 (joints), F03 (iMates & drive). Implements
ADR-0011 (one solver for sketch + assembly). Static positioning only — contact and
dynamics are iteration 4.*

## Constraints position occurrences through the one solver

Assembly constraints relate two occurrences' geometry (via context proxies, doc 00)
and reduce their relative DOF. They build a `solve.System` of rigid-body variables
(6 DOF/occurrence) + residual equations — the **same engine** as the sketch solver
(ADR-0011), differing only in variable kind and the residual library.

```go
package assembly
type Constraint interface{ Residuals(*solve.VarMap) []solve.Residual; Refs() []Ref }

type MateConstraint struct {     // faces/edges/points coincident (opposed or aligned)
    A, B   Ref                   // reference-keyed PROXIES (doc 00) — assembly-space geometry
    offset param.Quantity        // parameter-backed (core/04)
    flush  bool                  // mate vs flush
}
// Angle, Tangent, Insert (axis+plane), Symmetry — each a residual contribution
```

- **Inputs are reference-keyed proxies** (`Ref` resolving to a `Proxy[*topo.Face]`),
  resolved at solve time exactly like feature inputs (ADR-0010). A constraint whose
  geometry vanished goes **sick** and is re-selectable — never crashes the solve.
- **Offsets/angles are parameters** (core/04): driving an assembly is just editing a
  constraint's parameter → DAG dirty → async re-solve (ADR-0007).
- **Grounded** occurrences are fixed variables (the anchor, like a fixed sketch
  point). At least one ground per connected component or it floats (reported as DOF).
- **Diagnostics**: over/under/redundant detection and per-occurrence DOF come from
  the same graph analysis the sketch uses (ADR-0011) — "this part is under-
  constrained (3 DOF remain)" falls out for free.

## Joints = reduced-DOF parameterizations (not a second mechanism)

A joint expresses the relative placement of two occurrences with a **tight
parameterization** instead of separate constraints (parametric-cad: the simplified
joint model). Per ADR-0011, a joint contributes a small variable block + limits to
the *same* `solve.System`:

```go
type Joint struct {
    kind     JointKind          // rigid|rotational|slider|cylindrical|planar|ball
    A, B     Ref                // joint origins (proxies)
    dof      DOFParam           // e.g. rotational → {θ param} ; cylindrical → {θ, d}
    limits   []Limit            // min/max on driven vars
}
```

- A rotational joint replaces 6 relative DOF with 1 driven angle θ; the solver sees a
  variable block of size 1 plus the alignment residuals fixing the axis. No separate
  joint solver.
- **Limits** are inequality bounds on driven variables (clamped during solve/drive).
- Joints and constraints **coexist** in one assembly and one `solve.System`.

## iMates — reusable, auto-pairing constraint stubs

```go
type iMateDefinition struct{ name string; kind ConstraintKind; geom Ref } // on a component
type iMateResult struct{ a, b *iMateDefinition; constraint Constraint }    // matched on placement
```

An iMate is a **named, typed constraint half** stored on a component definition. When
a component with iMates is placed, the host **auto-pairs** matching-named iMates
between components and materializes real `Constraint`s (an `iMateResult`). Composite
iMates bundle several. This is a placement-time convenience over the same constraint
model — no new solving.

## Drive & animation

Driving sweeps a constraint's or joint's **driven variable** through a range, re-
solving (warm-started, ADR-0011) at each step — reusing the interaction/animation
`dt` from the frame loop (core/00):

```go
type Drive struct{ target DrivenVar; start, end param.Quantity; steps int; collision bool }
```

- Each step: set the driven variable → async re-solve from the previous step's
  solution (cheap warm start) → render. Smooth motion at frame rate.
- **Collision-stop** (optional) halts the drive when interference is detected — a
  *query* between steps (static interference, iteration 4), not the contact solver.
- This is purely kinematic positioning; **dynamics** (forces, mass, time
  integration) is M18, a different engine (ADR-0011 scope note).

## Why one solver was the right call (ADR-0011 payoff)

Everything above — mates, flush, angle, insert, joints, limits, grounding, drive,
DOF diagnostics — is *the sketch solver's machinery* over 3D rigid-body variables.
We wrote robustness, warm-start, decomposition, and over/under-constraint reporting
**once** (ADR-0009) and got the assembly positioner for the cost of a 3D residual
library + a quaternion parameterization. The hardest XL risk (a second constraint
solver) was avoided by construction.

## Net mapping from COM

| COM | Here |
|---|---|
| `MateConstraint`/`FlushConstraint`/`AngleConstraint`/`InsertConstraint` | `Constraint` residuals over `solve.System` (proxy inputs) |
| `ConstraintLimits` | `Limit` inequality bounds |
| `AssemblyJoint` + `AssemblyJointDefinition` | `Joint` = reduced-DOF variable block |
| `iMateDefinition`/`iMateResult` | named constraint half + placement-time auto-pair |
| `DriveConstraintSettings` | `Drive` sweeping a driven variable, warm-started re-solve |
| (separate assembly solver) | the **one** `solve/` engine (ADR-0011) |
