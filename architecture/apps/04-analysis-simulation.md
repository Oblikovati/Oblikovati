# Apps 04 — Analysis, measurement & simulation

*Modernizes M18 (measure/mass, interference, FEA, dynamics, tolerance) **plus** the
contact solver deferred from iteration 3 (M12-F05). The split that organizes this
doc: **lightweight queries are pure-Go core; heavy solvers are out-of-process compute
services** (ADR-0003).*

## The build-vs-integrate line

| Capability | Where it lives | Why |
|---|---|---|
| Measure, mass properties, interference | **pure-Go core** (kernel queries) | cheap, exact, no external solver; needed everywhere |
| Contact solving (interactive) | **core** (a focused collision/impulse module) | must run in the interaction loop, low-latency |
| FEA, multibody dynamics | **out-of-process compute services** (ADR-0003) | major subsystems; may use native/established solvers without contaminating the cgo-free core |

This keeps the geometry/model core pure Go and cross-compilable (ADR-0002/0008) while
letting the genuinely heavy numerical solvers be whatever is best — reached over the
**gRPC boundary we already built** (ADR-0003). Analysis is, architecturally, mostly an
add-in domain.

## Measurement & mass properties — pure-Go kernel queries

```go
package analyze
func Measure(a, b Entity) MeasureResult           // distance/angle/area/min-distance
func MassProps(body *topo.Body, m asset.Material) MassProperties // volume·density → mass, COM, inertia
```

- **Measurement** (distance/angle/area/min-distance) uses the kernel evaluators
  (core/03) — exact, interactive, available from kernel Phase A.
- **Mass properties** integrate geometry × **material density** (apps/02, M16) →
  volume, mass, center of mass, inertia tensor, principal axes. Pure Go over the
  B-rep. Feeds dynamics (below).

## Interference & model health

- **Static interference** between occurrences is a **kernel boolean intersection**
  (core/03, Phase C) over assembly-space proxies (assembly/00) → overlap volume +
  location. Used by the drive collision-stop (assembly/01).
- **Model health aggregation** (a "design doctor") is a *query*, not an engine: walk
  the document collecting sick features / lost references (the `Health` already on
  every feature, modeling/01) → a fix-it list. Free, because health is first-class
  state (parametric-cad §2).

## Contact solver (deferred from iteration 3)

Real-time contact (dragging a component stops at contact instead of interpenetrating)
is a **different problem from static positioning** (ADR-0011 scope note) — it is
collision detection + inequality response, not equality constraints. A focused module
in the core:

```go
package contact
type ContactSet struct{ members []OccurrencePath }
func (s *Solver) Resolve(drag DragState) []Correction   // collision detection → push-out, per interaction frame
```

- Runs in the **interaction phase** (core/00) during drag, low-latency, on the worker
  pool — broad-phase (BVH over occurrence bounds) + narrow-phase (kernel proximity).
- It biases the assembly positioning re-solve (assembly/01) with inequality
  corrections; it does **not** replace it. Kinematic, not dynamic.

## FEA & dynamics — out-of-process compute services (ADR-0003)

```
host (Go core) ──gRPC──▶ analysis service (own process, any solver tech)
   sends: tessellated/meshed geometry + materials + loads/constraints (or joints+forces)
   streams back: results (stress/displacement/SF  |  motion/reaction graphs)
```

- **FEA**: geometry simplification + **meshing reuses kernel tessellation** (core/03);
  loads/constraints/materials are defined in-app (typed, reflection-inspector UI,
  core/09) and shipped to the service; results stream back and are visualized via the
  renderer (core/08, result-field coloring). The **solver** is an integrated/established
  FE engine in the service — not built into the core.
- **Multibody dynamics**: the mechanism is extracted from **joints/constraints**
  (assembly/01) + **mass properties** (above) + forces; the service integrates motion
  over time and streams position/velocity/reaction graphs (which can feed FEA loads).
  Distinct from the static positioning solver (ADR-0011).
- Both are **clients of the public API** (ADR-0003) for geometry/topology access —
  another dogfood of the contract (core/07). Running them out-of-process also means a
  long solve never blocks the editor (the async principle, ADR-0007, taken to the
  process level).

## Tolerance & GD&T analysis

Model-based GD&T (tolerance features, datum reference frames — the annotation objects
from apps/00) plus **tolerance-stack analysis**: propagate dimensional/geometric
tolerances through the assembly to compute worst-case and statistical (RSS) variation,
with contributor breakdown. A computation over the parameter/dimension graph (core/04)
and assembly structure (assembly/00) — core, not a service.

## Why this completes the picture cleanly

Analysis splits exactly along the architecture's grain: **queries over the model**
(measure/mass/interference/tolerance/health) are pure-Go core because they are cheap
and ubiquitous; **heavy solvers** (FEA/dynamics) are out-of-process compute services
because they are major, may want native solver tech, and must never block the editor —
and the **gRPC boundary that makes that clean already existed** for add-ins (ADR-0003).
Contact is the one new in-core module, scoped tightly to interactive dragging. No part
of M18 forces a compromise of the cgo-free core (ADR-0002) — the boundaries laid down
in iteration 1 absorb even the heaviest domain.

## Net mapping from COM

| COM | Here |
|---|---|
| `MeasureTools` / `MassProperties` | pure-Go kernel queries (`analyze`) |
| `InterferenceResults` | kernel boolean intersection over proxies |
| `ContactSolver` (M12-F05) | focused in-core collision module (interaction loop) |
| stress analysis / `DynamicSimulation` | out-of-process compute services over gRPC (ADR-0003) |
| `ModelToleranceFeature` / `ModelDatumReferenceFrame` | tolerance-stack computation over the param/assembly graph |
