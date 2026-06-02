# Apps 00 — Drawing & documentation

*Modernizes M14 (drawing documents, views, dimensions/GD&T, tables, output).
Implements ADR-0012 (exact hidden-line). The one substantial new engine is HLR;
everything else reuses the spine.*

## Drawing is a document type producing associative views

A drawing is a `Document` (core/05) whose content is sheets of **associative views**
of 3D models. The defining property: a view is a **DAG dependent of the model** —
when the model changes, the view recomputes (async, ADR-0007), and dimensions
attached to it follow.

```go
package drawing
type Content struct{ sheets Collection[*Sheet] }
type Sheet   struct{ size SheetSize; border *Border; titleBlock *TitleBlock; views Collection[*View]; tables []Table }
type View struct {
    model    Ref               // the part/assembly being drawn (document reference, core/05)
    camera   ViewOrientation    // base/projected/iso; section/detail derive from a parent view
    scale    param.Quantity
    style    RenderStyle        // wireframe | hidden-line | shaded
    curves   []*DrawingCurve    // OUTPUT of HLR — each tagged with a model-edge RefKey
}
```

- **Sheets/borders/title blocks** are parametric (a sketch + prompted fields sourced
  from iProperties, core/05) — the COM `BorderDefinition`/`TitleBlockDefinition` as
  parameter-backed templates.
- **Standards/styles** (dimension/text/line) are the style system shared with
  visualization (apps/02, M16) — one style manager.

## Drawing views = exact HLR projection (ADR-0012)

The view's `Recompute` runs the **exact hidden-line engine** (ADR-0012): silhouette +
sharp edges, projected, analytically occlusion-tested against the B-rep, classified
visible/hidden, emitted as **vector `DrawingCurve`s carrying the source edge's
reference key**.

```go
func (v *View) recompute(ctx) {
    body := resolve(v.model)                       // current model topology
    v.curves = hlr.Project(ctx, body, v.camera, v.scale)  // analytic HLR (ADR-0012), reuses kernel
    // each DrawingCurve.SourceKey = the model edge's RefKey → associativity
}
```

- **Associativity for free**: because each curve knows its model edge (reference key,
  core/05), a dimension attached to the curve re-attaches after the model rebuilds —
  the curve is the edge's projection, not a dumb line.
- **View types** — base/projected (orthographic), section (cutting plane → analytic
  section + hatch), detail (boundary + scale), auxiliary (fold line), break/crop — are
  all the same engine with different cameras/clipping; section/detail are children of
  a parent view in the DAG.
- **Recompute is async + cached** (ADR-0007, ADR-0012): pan/zoom the sheet is pure 2D
  redraw; HLR re-runs only on model/camera/scale change, on the worker pool.

## Dimensions & annotations reuse parameters + reference keys

```go
type Dimension struct {
    refs  []Ref                // attach to DrawingCurves (→ model edges) — reference-keyed
    value param.Quantity        // retrieved from the model OR a driving param
    kind  DimKind; tol Tolerance
}
```

- A **model dimension** retrieved onto a view reads the model's parameter (core/04);
  a **drawing dimension** measures the curves. Either way the value is a `Quantity`
  and updates with the model — the same unit/tolerance system as sketches (modeling/00).
- **GD&T** (feature control frames, datum reference frames, surface texture),
  centerlines/center marks, and leader notes are typed annotation objects on
  **annotation planes**, attached to topology via reference keys → they track the
  feature they annotate.

## Tables come from the model

- **Parts list** is a view onto the assembly **BOM** (assembly/00) — same data, drawn;
  **balloons** reference list items by the BOM's stable item numbers.
- **Hole table** is driven by **hole features' `TapInfo`** (modeling/03) + a datum
  origin → X/Y/spec rows.
- **Revision table** and general/custom tables round it out. All are queries over the
  model/assembly, rebuilt when their source changes.

## Output

Print/plot and export to **PDF and DWG/DXF** flow through the translator framework
(apps/03, M17): the drawing is vector curves + annotations + tables, serialized with
style/layer mapping. The null renderer (core/08) produces **view thumbnails** headless.

## What is genuinely new vs. reuse

| New (this milestone) | Reused from the spine |
|---|---|
| Exact HLR engine (ADR-0012) — the one hard new thing | document model (core/05), DAG recompute (core/04), async (ADR-0007) |
| Drawing view/sheet/annotation object model | reference keys for associativity (core/05) |
| Section/detail analytic derivation | parameters/units/tolerance (core/04), BOM (assembly/00) |
| | kernel evaluators/topology for HLR (core/03), translators for output (M17) |

Drawing is the clearest demonstration that the reference-key investment (core/05,
ADR-0010) pays compounding dividends: model→view→dimension associativity, the feature
that makes drawings *drawings* and not screenshots, is "free" once identity is right.
