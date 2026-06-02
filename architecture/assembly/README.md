# Assembly (iteration 3)

The multi-part domain, built on the iteration-1 core and the iteration-2 modeling
spine. Assemblies are where the parametric-cad skill's **prototype/flyweight
instancing** (§5) and **context proxies** (§5a) become concrete — and where the
realtime-3d skill's **scene-graph instancing** (one mesh, many transforms) pays off
directly on the GPU.

The central idea (parametric-cad §5): an assembly does **not** copy components. One
shared `ComponentDefinition`, many `ComponentOccurrence`s — each a lightweight
instance with its own placement transform, nested by occurrence path. Memory and
recompute scale with *unique* parts, not part *count*; and on the GPU, N occurrences
of a part are N transforms over one cached mesh (core/08) — GPU instancing for free.

## What it covers (plan milestones)

| # | Doc | Modernizes (plan) | New ADRs |
|---|-----|-------------------|----------|
| 00 | [Instancing & context proxies](00-instancing-and-proxies.md) | M11 | — (uses core/02 `Proxy[E]`) |
| 01 | [Constraints, joints & drive](01-constraints-joints.md) | M12-F01/F02/F03 | [0011](../decisions/ADR-0011-assembly-positioning-solver.md) |
| 02 | [Representations & model states](02-representations.md) | M12-F04/F05 | — |

## How it reuses the spine (nothing here is new plumbing)

- **The generic `Proxy[E Entity]`** (core/02) — already decided; assembly is its
  first real consumer. Part-space geometry viewed through an occurrence context.
- **The unified solver** (ADR-0011) — assembly positioning is the 3D instance of the
  sketch solver (ADR-0009), not a new engine.
- **Async/cancellable recompute** (ADR-0007) — assembly position solve and the
  propagation of part edits up into assemblies run off the frame loop.
- **Commands + events** (core/06) — place/move/constrain/suppress are commands;
  reps fire events.
- **Reference keys** (core/05) — constraint/joint inputs are reference-keyed proxies
  resolved at solve time, exactly like feature inputs (ADR-0010).
- **Scene-graph instancing** (core/08, realtime-3d §3, §6) — one body mesh, many
  occurrence transforms; the drawing references the live transform pointer.
- **Document references** (core/05) — an occurrence references another *document*'s
  content; the reference graph already exists.

## The assembly object model in one diagram

```
   AssemblyContent
     ├── occurrences []*Occurrence ───┐
     │      Occurrence {              │  (flyweight — parametric-cad §5)
     │        Definition  ───────────────▶ shared ComponentDefinition (a part/subassembly doc)
     │        Transform   math.Mat4   │      (one definition, N occurrences)
     │        Grounded/Suppressed     │
     │        Path/SubOccurrences     │  (nesting → OccurrencePath)
     │      }                         │
     ├── constraints/joints ──▶ solve.System (ADR-0011) ──▶ occurrence transforms
     ├── representations (design-view / positional / LOD) ── overrides
     └── bom (derived from occurrence structure)

   geometry in assembly space:  Proxy[*topo.Face]{ Native: defFace, Context: occPath } (core/02)
```

Iteration 3 stops at static assemblies (positioning, reps, BOM). **Contact solving
and dynamic simulation** (M12-F05, M18) are deferred to iteration 4 — they are a
different problem class (inequalities / time integration), not static positioning
(ADR-0011).
