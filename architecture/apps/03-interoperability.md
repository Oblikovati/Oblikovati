# Apps 03 — Interoperability & translation

*Modernizes M17 (translator framework, neutral CAD, DWG/DXF, mesh formats,
shrinkwrap). The framework is the **registry** (core/07) + the **kernel's** import-
heal/export-tessellate (core/03). Pure Go where feasible (ADR-0002/0008).*

## Translator framework — pluggable via the registry

A translator is a registered capability (in-proc via the registry, core/07, or a
third-party **gRPC add-in**, ADR-0003) that imports/exports a format under a context
with `NameValueMap`-style options.

```go
package translate
type Translator interface {
    Formats() []Format                 // extensions, has-import, has-export
    Import(ctx, src DataMedium, opts Options) (*doc.Document, error)
    Export(ctx, d *doc.Document, dst DataMedium, opts Options) error
}
func init() { registry.Translators.Register(stepTranslator{}) }   // self-registration
```

- Adding a format = a package + one blank import (core/07) — the COM `TranslatorAddIn`
  zoo becomes uniform registration.
- Third-party/proprietary translators can run **out-of-process** over gRPC (ADR-0003)
  — useful where a format needs a native library we won't link into the cgo-free core.

## Neutral CAD exchange (STEP / IGES / SAT / Parasolid)

B-rep exchange is the heavy lift. The pipeline reuses the kernel:

```
import:  parse → build topology (kernel/topo) → HEAL (kernel/ops, Phase D) →
         wrap as NonParametricBaseFeature (modeling/02) → editable downstream
export:  serialize the model's B-rep topology + geometry to the format
```

- **Import healing** (gap-sew, orientation fix, validity — core/03 Phase D) is what
  turns a foreign, tolerant body into a valid one; imported solids arrive as **base
  features** (modeling/02) the feature tree can build on.
- **STEP (AP203/214/242)** and **IGES** are pure-Go parsers/writers (text/STEP-format
  — no cgo, honors ADR-0002). **SAT/Parasolid** are kernel-level B-rep interchange.
- Gated by **kernel Phase D** (healing/tolerant modeling) for robustness — the
  framework and the format parsers can be built earlier; fidelity grows with the
  kernel.

## AutoCAD DWG / DXF

- **DXF** (documented text/binary) → pure-Go read/write of entities (line/arc/
  polyline/spline) and blocks, mapping to **sketch geometry** (modeling/00) and
  **drawing** curves/layers (apps/00) — also the sheet-metal flat export target
  (modeling/05).
- **DWG** (proprietary binary) is the one honest exception: a fully pure-Go DWG
  reader is costly. Options, in preference order: (1) a maturing pure-Go DWG library;
  (2) an **out-of-process** translator (ADR-0003) wrapping a native DWG lib, keeping
  the core cgo-free. Flagged as the place the "no native deps" stance is most
  pressured — but the gRPC boundary already gives us a clean way to isolate it.

## Mesh & visualization formats

Tessellation-based exchange is the **easy** part — it reuses kernel faceting (core/03)
and appearances (apps/02):

| Format | Direction | Built from |
|---|---|---|
| STL / OBJ / 3MF | export (import → mesh features, modeling/04) | kernel tessellation (core/03) |
| glTF / 3D-PDF / JT | export (visualization) | tessellation + appearances/materials (apps/02) |

All pure Go, no kernel-phase dependency (faceting exists from Phase A) — these ship
early and are the first useful export.

## Shrinkwrap / derived simplified export

Shrinkwrap (envelope an assembly, remove internals, fill holes → a single simplified
body for sharing/IP-protection/performance) reuses **derived components** (modeling/02)
and kernel ops (Phase C/D). It is also the substitution target for assembly occurrences
(assembly/00).

## Why the framework is light

Interop is **registry + kernel**: the framework is self-registration (core/07), import
is parse→heal→base-feature (core/03 + modeling/02), export is serialize-topology or
tessellate (core/03), and the hard formats (DWG, robust STEP) are isolated behind the
gRPC out-of-process boundary (ADR-0003) so the **core stays pure Go and cross-
compilable** (ADR-0002/0008). Mesh/visualization formats ship first (Phase A); neutral
B-rep fidelity tracks kernel Phase D.

## Net mapping from COM

| COM | Here |
|---|---|
| `TranslatorAddIn` + `TranslationContext` | registered `Translator` / gRPC add-in + `DataMedium` |
| STEP/IGES/SAT translators | pure-Go parsers → heal → base feature (modeling/02) |
| `DWGEntity`/`DWGBlock*` | DXF pure-Go; DWG isolated out-of-process |
| STL/OBJ/glTF export | kernel tessellation (core/03) + appearances (apps/02) |
| shrinkwrap | derived simplified body (modeling/02) |
