# Application & output domains (iteration 4)

The closing iteration. These are the capability domains built *on* the modeling +
assembly spine — documentation, automation, visualization, interoperability, and
analysis. They complete the modernization of all 19 implementation-plan milestones.

The recurring theme of iteration 4: **almost everything here is a client of the core
spine, not new plumbing.** Drawing is a DAG-dependent projection of the model;
automation is an automation client of the public API; visualization feeds the
renderer caches; interop is the registry + the kernel's import/tessellate; analysis
is queries over the kernel + out-of-process compute. Only two genuinely new engines
appear — exact hidden-line removal (ADR-0012) and the embedded rules runtime
(ADR-0013) — and both are bounded applications of existing machinery.

## What it covers (plan milestones)

| # | Doc | Modernizes (plan) | New ADRs |
|---|-----|-------------------|----------|
| 00 | [Drawing & documentation](00-drawing-documentation.md) | M14 | [0012](../decisions/ADR-0012-exact-hidden-line.md) |
| 01 | [Design automation: iPart & iLogic](01-design-automation.md) | M15 | [0013](../decisions/ADR-0013-ilogic-embedded-scripting.md) |
| 02 | [Visualization, appearances & presentations](02-visualization-presentation.md) | M16 | — |
| 03 | [Interoperability & translation](03-interoperability.md) | M17 | — |
| 04 | [Analysis, measurement & simulation](04-analysis-simulation.md) | M18 | — |

## How each reuses the spine

| Domain | Built from |
|---|---|
| **Drawing** | drawing = document (core/05); views = DAG-dependent (core/04) HLR projections (ADR-0012) with reference-keyed curves; dims = params (core/04) on model refkeys; tables = assembly BOM (assembly/00) |
| **Automation** | iPart = data-driven member generation over the feature engine; iLogic = embedded scripting (ADR-0013) over the public API + event triggers (core/06) |
| **Visualization** | appearances/materials feed renderer material caches (core/08); presentations = occurrence-transform animations (assembly/00) over the frame-loop clock (core/00) |
| **Interop** | translator framework = registry (core/07) / gRPC add-ins (ADR-0003); import = kernel heal→base feature (modeling/02); export = serialize topology/tessellation (core/03) |
| **Analysis** | measure/mass = kernel queries + material density (M16); interference = kernel booleans (core/03); FEA/dynamics/contact = out-of-process compute services (ADR-0003) |

## A deliberate build-vs-integrate stance (analysis)

FEA and multibody dynamics are major subsystems in their own right. Rather than build
them into the cgo-free core (ADR-0002), iteration 4 runs them as **out-of-process
compute services over the gRPC boundary** (ADR-0003) — they may use whatever solver
technology is best (including native libraries) without contaminating the pure-Go
core or its cross-compilation. Measurement, mass properties, and interference *are*
pure-Go core (kernel queries). See [apps/04](04-analysis-simulation.md).

## Completion

With this iteration, every milestone M00–M18 of the implementation plan has a
modern-architecture treatment on the Go + Vulkan stack. The [root README](../README.md)
status table reflects all four iterations complete.
