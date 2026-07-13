# Inventor Translator

Part of the **Translators** category (`model/exchange/translators/`): tools that **convert a
foreign CAD file into a native Oblikovati document** — each source parameter becomes an
Oblikovati parameter, each sketch an Oblikovati sketch, each feature an Oblikovati feature, and
the kernel rebuilds the solid by replaying that feature tree. This is distinct from geometry
**Import** (STEP/STL), which brings in a static body.

Surfaced in the GUI as **File ▸ Translate ▸ From Inventor** (file dialog for `.ipt`); the headless
entry is `cmd/from-inventor`. It reads `.ipt` files directly and needs **no running Inventor** —
it is the offline twin of the COM-driven `Oblikovati.Exporter.Inventor`, targeting the same
`partRecipe`. Feasible because of M44 Inventor↔Oblikovati API parity (near 1:1 feature mapping).

This module is dependency-isolated (its own `go.mod`; only external dep `klauspost/compress` for
Zstd; resolved via `go.work`, main module untouched). RE notes + oracles: `../../../../../_iptformat/`.

## Packages
- `olecf/` — dependency-free Microsoft Compound File (CFBF/OLE2) reader.
- `ipt/` — segment layer (Zstd), and decoders for the `.ipt` payload:
  - `segment.go` — the `B/M` segment pairs → decompressed named segments.
  - `dc.go` — `PmDCSegment` parameters; `sketch.go` — sketch points/lines/circles;
    features (extrude/revolve/…) are next.
  - `sab.go`/`brep.go` — the ACIS B-rep (fallback for non-parametric bodies).
- `translate/` — maps the decoded model onto Oblikovati (`FromInventor(iptBytes, outPath)`).
- `cmd/from-inventor/` — `from-inventor <file.ipt> [-o <out.opd>]`.

## Status
- ✅ Container + **Zstd (Inventor 2027+) *and* zlib (older releases)** segments + SAB parse
  (topology counts match the oracle). Real-world `.ipt` files from Inventor 2020–2026
  zlib-compress their segment payloads (`0x78` header) rather than Zstd; both inflate.
- ✅ **Parameters** → native Oblikovati parameters (user params; model params excluded).
- ✅ **Sketches** → native points / lines / circles / **arcs** (centres + endpoints resolved
  by reference, so **off-origin** geometry is exact), and **arbitrary closed polygons**
  (convex *and* non-convex) via endpoint-reference-graph loop resolution (validated;
  round-tripped) — an L-profile rebuilds to an exact 6 cm³, notch and all.
- ✅ **Features** → extrude, revolve (**full + partial-angle**), **loft (multi-section)**, **sweep**, **boolean operations** (cut / join / intersect),
  **holes** (drilled through/blind + **counterbore + countersink + tapped**, real `HoleFeature`),
  **rectangular + circular patterns**, and **mirror**, **multi-feature + multi-sketch**.
  Everything rebuilds parametrically; reopened volumes match: box 8, cylinder ≈ π·1²·2,
  revolve tube ≈ 4π, box+pocket 7, box+box 10, box−Ø1 through 14.43 / blind 15.21,
  counterbore 30.81 / countersink 31.22, pocket ×3 rect 22.92, disk−pocket ×6 circular 23.56,
  pocket + mirror 23.28, 270° revolve 3π ≈ 9.42, loft (2×2→4×4) 112/3 ≈ 37.33, **3-section
  loft (2×2→4×4→1×1)**, sweep Ø0.5 along L-path (elbow) + along a 90° arc (3.66 cm³).
- 🟡 **Real-world parts** (interactively-modelled, not corpus-synthetic): sketch decode is
  generalized — a part's point nodes carry a schema marker that *varies* (0x0A96 / 0x0B5C /
  0x0C20), so keying on one value found nothing; keying on the structural node shape instead
  now decodes sketches on ~70% of a real mechanical library (e.g. a shaft's 7-line revolve
  profile rebuilds exactly). The **boolean operation** and **model parameters** also decode now
  (the enum/named nodes carry a schema marker that varies per part — 0x0A96 / 0x0B5C / 0x0C20 —
  so the readers anchor on node structure, not the marker; ~30% of the library decodes an
  operation). The **point set is completed** (`forEachNamelessNode` keys on the +12 marker's high
  bit — a small enum is a geometry carrier, the high bit a structural node — so points the earlier
  `+16==0` gate silently dropped, the sketch origin among them, are now collected; the point tag at
  +20 is required to be the exact `0x800000XX` node-type shape, rejecting phantom vertices from a
  word that merely has the high bit set).
  **Profile connectivity from point incidence — the exact SketchPoint→geometry association**
  (`ipt/incidence.go`, `LineProfiles`). A geometry Point2d node carries, right after its coordinates,
  a connectivity header `0x30000002 | deg | deg | 0x10` followed by `deg` entity references — the ids
  of the curves through the point. **Two points that name the same reference are the endpoints of that
  line.** Rebuilding lines this way replaces the fragile creation-order *rank-alignment* (which
  mis-maps coordinates when the reference and point sets aren't a clean bijection) and — because each
  reference is a globally-unique edge id linking its two endpoints by coordinate — **reunites a profile
  even when Inventor splits it across the 800-byte cluster gap** (the failure that left TorquimeterShaft
  an open chain across clusters). Connected components of the incidence graph *are* the sketches, so
  multi-sketch parts separate without the byte-offset heuristic; non-convex loops (the L-profile notch)
  rebuild exactly. Two graph fix-ups make real shafts close: **degree-completion** recovers the edge
  hidden inside a vertical/horizontal-constraint group (3+ collinear points share one reference; the
  missing segment joins the two still short of degree two, adjacent along the run), and **leaf-pruning**
  drops a diameter dimension's dangling witness-point edge that would otherwise make the ring non-simple
  (degree counted per *coincident vertex*, so distinct point nodes at one corner count once). A
  **revolve is emitted only when it can be proven correct** (`ipt.RevolveProfile`): a genuine *closed,
  one-sided* loop about an *unambiguous* axis — a **separate single-line centreline the profile is
  strictly offset from** (any orientation; checked first so a user-drawn centreline wins), or (a solid
  shaft) a **profile edge on the vertical axis (x≈0)**. When a profile offers **both** — its own x≈0
  edge *and* a separate non-collinear centreline — the axis is genuinely ambiguous (1677K262, actually
  a partial revolve whose angle lives in no readable parameter), so it declines to the mesh rather than
  sweep a wrong solid. The **feature→sweep-extent binding** decides full vs partial from the revolve's
  OWN extent flag: a single-variable corpus (one profile revolved at 80/150/220°/full) showed the
  extent-type enum (`kind=12`, the `0x26`-trailer shape shared with extrude/hole) stores **3 for a full
  revolve, 1 for a partial (angle) one** — confirmed on real parts (ReelMotorBearingShaft/
  TorquimeterShaft = 3). So a full revolve is swept 360° even when its profile carries an angle
  **dimension** (a chamfer's 135°) that the earlier param scan mistook for the sweep and rendered as a
  sliver. Where the enum is present it is authoritative; where a real part omits it, a partial angle is
  read from the sole angle-unit parameter — located by its on-disk shape (a float64, its nominal
  duplicate, then the angle unit id at +20), more robust than the `d`-named model-param reader (which
  mis-picked a stray near-zero double ahead of a 150° sweep, rendering it full). A many-param part with
  no full-extent flag stays FULL — a wrong partial is worse than the mesh.
  Net: **the ReelToReel shafts (bearing / roller / torquimeter) rebuild as correct parametric solids —
  the stepped bearing shaft at full 360° (≈11.9 cm³, was a 1-radian sliver), the torquimeter shaft
  reunited across its cluster split — with zero wrong revolves; partial revolves (80/150/220/270°)
  rebuild at their exact swept fraction.** Still open: **arc/fillet profiles** (`LineProfiles` declines
  an arc-bearing part so a fillet arc is never emitted as a straight chord), and partial revolves whose
  extent enum is absent AND whose angle isn't a lone parameter (1677K262 — declined on axis ambiguity).
- 🟡 **Static-body fallback from Inventor's own display mesh** (`ipt/graphics.go`): for a part that
  doesn't rebuild parametrically, the `PmGraphicsSegment` tessellation (per-face triangle patches —
  curved faces, holes, and fillets already meshed) is decoded and imported. It preserves the part's
  silhouette and reopens for the great majority of a real mechanical library (a reel-to-reel deck:
  102 of 104 parts), **but** Inventor's stored tessellation is *non-manifold / not watertight*, so it
  imports as an **open surface body (not a solid)** — its mass properties are unreliable (volume reads
  ~2× true). Making the fallback a watertight solid (weld + hole-fill, or gate on manifoldness) is a
  known follow-up. The older planar-only ACIS reconstruction remains a secondary fallback.
- ✅ **Assemblies (`.iam`)** → native Oblikovati assemblies, **including sub-assemblies**. The
  RSeStorage **node graph** is decoded (M-stream block metadata → typed B-stream blocks,
  `ipt/nodegraph.go`): occurrence structure from `AmDcSegment`, and each occurrence's
  **placement transform** from the `AmRxSegment` `232792BC` node's sparse `Transformation3D`.
  `AssemblyFromInventor(iamPath, outPath)` **recursively** translates each referenced
  component — a `.ipt` part or a `.iam` sub-assembly → `.opd` — and places every occurrence at
  its full transform (**translation and rotation**; a bar rotated 90° about +Z and translated
  (7,0,0) reopens with the exact matrix). A two-level corpus (top ▸ 2 subs ▸ 2 leaves each)
  reopens with the nested structure and placements intact; a shared-component cache + cycle
  guard keep the recursion safe.
- 🟡 **Assembly constraints** — a constrained assembly's **geometry** already translates
  correctly, because occurrences are placed at their *solved* positions (a mate that pulls a
  box to z=4 reopens at z=4). The constraint **kind** is decoded (mate / flush / angle /
  symmetry, from the `4E86F047` node), and a warning notes each. Rebuilding the parametric
  **relationship objects** needs resolving each geometry selection to a primitive (plane /
  axis / point) — the remaining step. Occurrence filtering now rejects the spurious
  `hash:N` names that constraint selections emit, so constrained/complex assemblies place
  the right components.
- 🎯 **Next**: guided (rail/centreline) loft, non-circular-profile / spline-path sweep,
  rebuild constraint relationships (selection→primitive), 2D-grid patterns, off-centre hole
  placement, revolve
  revolve, mixed extrude+revolve ordering, and richer holes (counterbore / countersink /
  tap, off-centre placement) — validated against the C# exporter's golden recipe.
  (Fillet/chamfer deferred while the host solver is refactored.) Reference: InventorLoader
  `importerDC.py` + `Exporter.Inventor/src/Exporter.Inventor.Recipe`. Tracked as #1997 (M17).
- ⏳ Analytic B-rep topology (loops/coedges) for the non-parametric fallback needs the
  field-exact ShapeManager grammar (InventorLoader `importerSAT.py`).

## Test / run
```
go test ./...
go run ./cmd/from-inventor testdata/10_box.ipt -o /tmp/box.opd
../../../../dist/oblikovati-cli.exe open /tmp/box.opd
```
