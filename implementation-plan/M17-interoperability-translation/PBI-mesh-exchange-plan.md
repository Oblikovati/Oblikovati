<!-- SPDX-License-Identifier: GPL-2.0-only -->
# Implementation Plan: Mesh Exchange — STL / OBJ / 3MF Import & Export

> Realizes M17-F04 (mesh exchange). Mirrors the contract-first wiring of the STEP plan
> (`PBI-159-step-import-export-plan.md`). Grounded in the actual kernel/model code.

## 0. Grounding summary (what already exists, reused — not duplicated)

- **Exchange seam** (`kernel/exchange/translate.go`): owned `BodyImporter`/`BodyExporter`
  interfaces + `TranslationOptions`. The STEP pair (`kernel/exchange/step/reader.go`,
  `writer_facade.go`) is the worked example — mesh mirrors its `reader.go`/`writer.go` shape.
- **Mesh → solid** (`kernel/subd/`): `subd.Mesh{Verts, Faces}` and `subd.ToBody(mesh, feat)`
  weld a cage of vertices/faces into a real `*topo.Body`: one shared vertex per cage vertex,
  one shared edge per cage edge, one planar face per cage face. **A closed cage (every edge
  used by exactly two faces) becomes a SOLID body; an open cage a SURFACE body** (`isClosed`).
  `model/feature/swept.go` is the precedent: `subd.ToBody(mesh, feat)`, then rebuild
  face-reversed when `ops.BodyGeometryProperties().Volume < 0`. This IS the mesh→solid path.
- **Validation / mass / tessellation** (`kernel/ops/`): `Validate(b) → {Valid, Manifold,
  Closed, OrientationOK}` is the manifold/closed/oriented gate; `BodyGeometryProperties(b, q)`
  gives Volume/COM; `TessellateBody(b, Quality)` + `Quality{ChordTolerance, AngleTolerance}`
  / `DefaultQuality()` is the EXPORT facetting path and the **resolution knob**.
- **Feature injection** (`model/feature/derived.go`): `NonParametricBaseFeature` already wraps
  pre-built `*topo.Body` into the recompute history (`Recompute` appends frozen bodies). The
  M17 imported-body feature builds on this: it persists its *source* (path+format+options) so
  reopen re-imports — the associative `.obk` story (an in-memory body cannot round-trip a
  topo.Body through YAML; re-import is the chosen persistence).
- **Recipe persistence** (`model/feature/serialize.go`): `FeatureData` payload union +
  `serializeFeature`/`buildFeature` dispatch keyed on `Kind()`. A new kind = one payload struct
  + one serialize case + one restore case.
- **Spine**: `model/compdef.PartComponentDefinition.Features()` / `.SurfaceBodies()` /
  `.Recompute()`; `model/doc.Workspace` (`Open`/`Save`/`SaveAs`); CLI `cmd/oblikovati-cli/`.
- **Public-API seam**: `Oblikovati.API` `types`/`contract`/`wire`/`client`; host serves `wire`
  via `addin/router` keyed on `wire.Method*`. No `documents.import/export` yet.

## 1. Format coverage

| Format | Import | Export | Notes |
|---|---|---|---|
| STL binary | yes | yes (binary) | 80-byte header + uint32 count + 50-byte triangles; little-endian |
| STL ASCII | yes | — (binary only on export) | `solid…facet normal…vertex×3…endfacet…endsolid` |
| OBJ | yes (`v`/`f`; ignore `vt`/`vn`/`usemtl`/`mtllib` first cut) | yes (`v`/`f`) | 1-based face indices; negative-relative indices supported on import |
| 3MF | yes (ZIP + `3D/3dmodel.model` XML) | yes | `<mesh><vertices><vertex…/><triangles><triangle…/>`; single object |

## 2. Mesh → solid strategy (the headline requirement)

1. Decoder produces an `exchange/meshio.RawMesh{Verts []Point3, Tris [][3]int}` (a triangle
   soup; STL/3MF give per-triangle verts, OBJ shares them).
2. **Weld** coincident vertices on a tolerance grid (default 1e-6 mm, scaled by unit) so faces
   share vertices/edges — the precondition for `subd.ToBody`'s closed-cage detection. (Mirrors
   the existing `weldVertex` in `model/feature/mesh.go`; lifted into a reusable welder.)
3. Build `subd.Mesh{Verts, Faces}` from the welded soup; `subd.ToBody(mesh, feat)`.
4. **Watertight ⇒ solid**: a welded mesh whose every edge is shared by two faces makes
   `subd.ToBody` emit a SOLID (`b.IsSolid()`); fix inside-out winding by rebuilding
   face-reversed when `BodyGeometryProperties().Volume < 0` (the swept.go pattern).
5. **Open ⇒ surface body**: a mesh with boundary edges makes `subd.ToBody` emit a SURFACE body;
   `ops.Validate` allows it open (no crash). Downstream features that need a solid will fail
   honestly; selection/measurement still work.
6. **Non-manifold** (an edge in >2 faces): `subd.ToBody` still builds the body; `ops.Validate`
   reports `Manifold=false`. We surface that as a warning (not a fatal) — the body exists.

`feat` lineage root: `fmt.Sprintf("import:%s#%d", format, shellIndex)` so reference keys are
stable across reopen (matches STEP's `importFeature`).

## 3. Resolution → ops.Quality mapping (export density knob)

`types.MeshResolution` ∈ {low, medium, high}. Higher resolution ⇒ tighter chord + angle ⇒ more
triangles for curved bodies. Planar bodies are unaffected (planar faces triangulate exactly).

| Resolution | ChordTolerance (mm) | AngleTolerance | rationale |
|---|---|---|---|
| low | 0.20 | 30° | coarse preview |
| medium | 0.05 | 10° | = `ops.DefaultQuality()` |
| high | 0.0125 | 5° | fine print/CAM |

Mapping lives in `kernel/exchange/meshio/resolution.go` (`QualityFor(MeshResolution) ops.Quality`).
Unit test: triangle count strictly increases low < medium < high for a sphere body.

## 4. Shared import/export framework (contract-first, ADR-0018)

**API first (`Oblikovati.API`, Apache-2.0):**
- `types/exchange_format.go` — `ExchangeFormat` (`FormatSTL`, `FormatOBJ`, `Format3MF`; reserves
  `FormatSTEP*`) + `MeshResolution` (`ResolutionLow/Medium/High`) value enums (no parser).
- `contract/translator.go` — `MeshTranslator` interface (`CanImport`/`CanExport`/`Formats`).
- `wire/exchange.go` — `MethodDocumentsImport="documents.import"`,
  `MethodDocumentsExport="documents.export"` + `ImportRequest{Path, Format, Options}`,
  `ImportResponse{BodyCount, Solid, Warnings}`, `ExportRequest{Path, Format, Resolution}`,
  `ExportResponse{TriangleCount, Warnings}`. (Method constants added to `wire/methods.go`.)
- `client/exchange.go` — typed `Import`/`Export` over `Transport`.

**GPL impl (`/Oblikovati`):**
- `kernel/exchange/meshio/` — `RawMesh`, `weld.go` (welder), `resolution.go`, `tobody.go`
  (`SolidOrSurface(RawMesh, feat) (*topo.Body, []string)`), and per-format
  `stl.go`/`obj.go`/`threemf.go` decoders/encoders, each behind a small owned reader/writer that
  satisfies `exchange.BodyImporter`/`BodyExporter`.
- `model/feature/imported_body.go` — `ImportedBodyFeature` (wraps the imported `*topo.Body` for
  recompute; persists `{Path, Format}` so reopen re-imports) + `ImportedBodies` collection.
- `model/feature/serialize_import.go` — `ImportData` payload + serialize/restore cases.
- `model/exchange/` — `ImportInto(part, path, format, opts)` / `ExportFrom(part, path, format,
  resolution)` dispatch by format (touches compdef/feature/doc).
- `addin/router/exchange.go` — `documents.import`/`documents.export` handlers.
- `cmd/oblikovati-cli/cmd_import.go` / `cmd_export.go`.
- MCP tool: exposed automatically by the bridge through the router method (note if the bridge
  isn't reachable in this worktree).

## 5. PBI slice order (smallest-shippable first; each = tests + a commit)

1. **Framework + contract wiring.** API types/contract/wire/client; `kernel/exchange/meshio`
   skeleton (RawMesh, weld, resolution, tobody); `ImportedBodyFeature` + serialize; `model/
   exchange` dispatch (formats stubbed); router + CLI. Tests: weld, resolution monotonic,
   tobody solid-vs-surface, imported-body recompute + `.obk` persistence, client/router table.
2. **STL** binary + ASCII import → solid; binary export at resolution. Tests: round-trip cube
   (export→import→Validate valid + IsSolid + volume tol); ASCII fixture; open-mesh → surface.
3. **OBJ** import → solid; export. Tests: round-trip cylinder; hand-authored `v`/`f` fixture.
4. **3MF** ZIP+XML import → solid; export. Tests: round-trip; hand-authored fixture. (Hardest —
   if ZIP/XML overruns this run, ship STL+OBJ fully and leave a documented skipped test.)
5. **Feature-on-top proof** (headline): import a watertight fixture → assert solid AND add a
   feature on top (boolean cut / fillet) succeeds. + resolution monotonic sphere test.

## 6. Test strategy (per CLAUDE.md, F.I.R.S.T)

- Round-trip invariant: kernel solid → export → import → `Validate` valid + `IsSolid` + volume
  within tol. (One independent hand-authored fixture per format too.)
- Solid-from-mesh + feature-on-top: the headline gate (slice 5).
- Resolution monotonicity: tri-count low < medium < high on a sphere.
- Open-mesh: non-watertight fixture → surface body, no crash.
- Decoders mock I/O with `bytes`/`strings` readers (no real filesystem in unit tests; CLI tests
  use `t.TempDir()`).

## 7. Risks / known gaps

- `subd.ToBody` makes **planar** faces only — a curved imported solid is faceted (expected for a
  mesh import; the body is a faceted solid, exactly per the requirement "a faceted solid").
- Tolerant sewing of near-coincident-but-not-welded gaps is out of scope (`ops.Sew` is stubbed,
  PBI-084); we weld on a grid, which handles exported-then-reimported meshes and clean fixtures.
- 3MF is the largest surface (ZIP container); if it overruns, STL+OBJ ship complete and 3MF is a
  documented tested-skip stub.
- Export does not preserve our reference keys (no slot in mesh formats) — re-import renumbers
  (documented; same limitation as STEP).
