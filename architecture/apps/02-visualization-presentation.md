# Apps 02 — Visualization, appearances & presentations

*Modernizes M16 (appearances/materials, styles, cameras/lights/views, presentations/
exploded views). This is the **model-side** of visualization that feeds the renderer
(core/08); the GPU work was decided there. Mostly data + animation, little new
engine.*

This doc connects the **model's** appearance/material/style data to the **renderer's**
caches and passes (core/08), and adds presentation documents (exploded views) as
occurrence-transform animations.

## Appearances & materials — the asset system

```go
package asset
type Material struct {            // physical + visual
    Density param.Quantity         // → mass properties (apps/04, M18)
    Modulus param.Quantity         // → FEA (apps/04)
    Visual  Appearance
}
type Appearance struct{ Color RGBA; Roughness, Metallic float64; Texture *AssetTexture }
```

- **Materials carry physical properties** (density/modulus) consumed by analysis
  (apps/04) — the same object serves rendering and engineering. Visual appearances
  feed the renderer's **material instance cache** (core/08, realtime-3d §5): a
  material maps to a pipeline + bindless texture indices.
- **Appearance source resolution** (precedence: face override > feature > body >
  occurrence override > material default) is one rule, shared with assembly
  design-view reps (assembly/02) — resolved once, consumed by the render queue
  (core/08). Editing an appearance swaps a material instance; no model rebuild.
- Assets live in **libraries** (loadable/shareable), the same library mechanism as
  styles and content center (apps/01).

## Styles & standards

The style manager centralizes visual + drafting styles (color, lighting, material,
dimension, layer) with libraries and cascade/override, firing **typed style events**
(core/06) so consumers refresh. It is the same manager the drawing standards use
(apps/00) — one style system across model, assembly reps, and drawings.

## Cameras, lights & views — mostly already in the renderer

The camera model, lights, named views, and orbit/pan/zoom were defined in the
renderer (core/08, realtime-3d §3/§8). This milestone adds the **model-side** glue:
named-view persistence on the document, lighting styles (a style flavor), and view
events (core/06) — the heavy lifting is the renderer's.

## Presentations & exploded views — occurrence-transform animation

A presentation document instructs assembly: components **tweaked** (translated/
rotated) from their assembled positions, with trails, sequenced into storyboards for
animation/publishing. Architecturally it is **animation over occurrence transforms**
(assembly/00) using the **frame-loop clock** (core/00):

```go
package presentation
type Content struct{ source Ref; views []*ExplodedView }   // references an assembly
type ExplodedView struct{ tweaks []Tweak; trails []Trail; storyboard *Storyboard }
type Tweak struct{ occ OccurrencePath; delta math.Mat4; ... }   // offset from assembled pose
```

- A **tweak** is a per-occurrence transform offset, keyed by **occurrence path**
  (assembly/00) — stable identity, survives reload. Exploding interpolates each
  occurrence from assembled → tweaked over the storyboard timeline using `dt` from
  the frame loop (core/00), driving the **scene transforms** (core/08, realtime-3d §3)
  — the same dirty-flag transform machinery that moves anything else in the viewport.
- **Trails** are overlay-graphics polylines (core/08) tracing a component's path.
- **Storyboards/snapshots** sequence tweaks + camera into animations; **publishing**
  to video/raster reuses the renderer (offscreen for non-realtime capture).
- The base assembly is **never mutated** — a presentation is a non-destructive
  override layer, exactly like representations (assembly/02).

## Why almost nothing new

| Reused | From |
|---|---|
| material/appearance → GPU | renderer material cache, draw-call-as-data (core/08) |
| appearance precedence | shared with assembly reps (assembly/02) |
| camera/lights/views | renderer (core/08) |
| exploded-view animation | occurrence paths (assembly/00) + scene transforms (core/08) + frame clock (core/00) |
| styles/libraries | the one style system (also apps/00 drawing standards) |
| physical material props | feed analysis (apps/04) |

Visualization is where the **renderer (realtime-3d skill)** and the **model
(parametric-cad skill)** finish meeting: the model supplies appearances, materials,
and occurrence transforms; the renderer's caches, scene graph, and draw queue consume
them. The presentation is just the scene graph animated over the frame clock — no new
subsystem, the realtime-3d patterns doing exactly what they were built for.
