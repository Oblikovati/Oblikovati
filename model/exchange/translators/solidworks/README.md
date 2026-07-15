# SolidWorks Translator

Part of the **Translators** category (`model/exchange/translators/`): tools that **convert a
foreign CAD file into a native Oblikovati document** — each source parameter becomes an
Oblikovati parameter, each sketch an Oblikovati sketch, each feature an Oblikovati feature, and
the kernel rebuilds the solid by replaying that feature tree. This is distinct from geometry
**Import** (STEP/STL), which brings in a static body. It is the offline twin of the SolidWorks
COM API: it reads `.SLDPRT` / `.SLDASM` directly and (for the CFBF format) needs **no running
SolidWorks**.

It reuses the `olecf` sibling module (the shared Microsoft Compound File reader also used by the
Inventor translator) and follows the same discipline: **never infer geometry** — decode only what
is read from real nodes, self-validated against a geometric invariant or the live-SolidWorks
oracle; keep geometry bit-exact; gate changes with a batch volume/outcome regression.

## Source corpus & the two container formats
Reverse-engineered against a real BMW S54 engine model (388 parts, 69 assemblies). The corpus
contains **two distinct container formats**, both of which the translator targets:

- **Format A — CFBF/OLE2** (the majority, older SolidWorks saves): the same compound-file
  container as an Inventor `.ipt`, so `olecf` reads it directly. Inside, the model is an **MFC
  `CArchive`-serialized object graph** in the `Contents/Config-0` stream; the solid B-rep is a
  **zlib-compressed Parasolid** partition in `Contents/Config-0-Partition`.
- **Format B — SolidWorks 2026 native** (non-OLE): the ~10 largest parts + some assemblies,
  last saved by SolidWorks 2026 (version stamp 19000). A **log-structured record store** with
  stream *names* nibble-swapped and stream *payloads* raw-DEFLATE-compressed. Reverse-engineered
  by differential analysis (SW2026 reads and writes it). **Fully decoded** (`sldprt/formatb.go`):
  un-swap names + raw-inflate → the **same `Contents/Config-0` MFC CArchive as format A**, so both
  containers share one downstream decoder. Validated on all 10 corpus format-B parts.

## Packages
- `sldprt/` — the format decoder:
  - `doc.go` — open a compound `.SLDPRT`/`.SLDASM`, classify by root CLSID (part/assembly/
    drawing — they differ only in CLSID byte[0]: `0x30`/`0x36`/`0x34`), expose streams.
  - `version.go` — the internal build stamp parsed from the `_MO_VERSION_NNNN` storage (3400).
  - *(next)* `carchive.go` — the MFC `CArchive` tag protocol (new-class `ff ff schema len name`,
    old-class back-refs, string tag `ff fe ff len utf16`); per-class `Serialize` decoders for
    the `sg*` sketch handles (points/lines/arcs/circles), parameters, and features.
- *(next)* `translate/` — maps the decoded model onto Oblikovati (`FromSolidWorks`).

## Status
- ✅ Container: **both formats decode through one interface** (`Document.Stream`). Format A via
  `olecf`; Format B via the `formatb.go` record-log reader (nibble-swap + raw-inflate). CLSID
  classification + `_MO_VERSION` decode. `go test ./sldprt/` green; Format B validated on all 10
  corpus format-B parts (Config-0 → `moPart_c` CArchive, incl. the 44MB/88MB parts).
- 🟡 **Parameters + sketches** (current priority): the sketch object graph is MFC `CArchive` (no
  per-object lengths, so each class's `Serialize` layout is RE'd incrementally). **Sketch decode**
  (`sketch.go`, `Document.Sketches()` → `{Points, Lines, Circles}`): the `1e 00` + X,Y-f64 point tag
  is read from the `sgSketch` region of `Config-0-ResolvedFeatures` (format B) or `Config-0` (format
  A) — one path for both containers. Entities (`{Lines, Circles, Arcs}`) are associated
  geometrically (the point-index references would need a full object-graph walk): **circles**
  (centre+rim → radius), **line loops** (convex reconstruction), **arcs**, and **mixed line/arc
  profiles** — a segment is an arc when its cached centre is equidistant from its two endpoints, a
  line otherwise. Validated exact against the SolidWorks-2026 COM oracle and generated known parts:
  rectangle, triangle, circles (origin/off-origin/multiple), a standalone arc, and a **rounded
  rectangle (4 edges + 4 fillet arcs)**. **Per-sketch splitting** groups the stream into one sketch
  per feature (by the `04 80 ff fe ff` name marker) before association, so a multi-sketch part
  decodes each sketch separately (validated on a rectangle+circle part; format-B parts split into
  their many sketches). Two known limits, both awaiting an MFC object-graph walk: older CFBF
  multi-sketch parts lack the marker and fall back to one merged region; and a sketch that re-uses an
  entity kind first seen earlier loses that class string, so its kind isn't detected. **Next:** the
  object-graph walk, then equations/global-variables → parameters. Oracle: `scratchpad/sw_dump.ps1`.
- ⬜ Features, and the Parasolid body fallback (zlib-inflate the partition), come after.

## RE oracle (SolidWorks 2026)
The live COM API is the ground truth (SW2026 opens both formats; it saves Format B). The compiled-
C# harnesses live in the session scratchpad: `sw_oracle.ps1` (feature tree + volume), `sw_dump.ps1`
(params + sketch geometry), `sw_gen.ps1` (generate minimal known parts — note SW2026 writes
Format B). Use the strongly-typed interop via `Add-Type` + an `AssemblyResolve` handler; PowerShell
late binding cannot resolve `ISldWorks` methods.
