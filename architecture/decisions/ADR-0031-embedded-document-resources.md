# ADR-0031 — Imported files are embedded in the document as a root `resources` dictionary (UUID-keyed)

**Status:** Accepted (2026-06-09) · **Builds on / refines:**
[ADR-0020](ADR-0020-yaml-git-friendly-document-format.md) (the single git-friendly YAML
`.obk` document) — this ADR adds *where imported file bytes live* in that file.
**Touches:** the mesh/STEP importer (`model/exchange`), the imported-body feature
(`model/feature/imported_body.go` + `serialize_import.go`), TrueType text/emboss
(`model/text`, `model/feature/serialize_emboss.go`), and the recipe codec
(`persistence/yamlcodec`).

## Context

The `.obk` document is a single YAML text file holding **only the recipe** — parameters,
sketches, and the ordered feature program — never realized geometry (ADR-0020). That ADR
deliberately left one thing open: a recipe step can *consume an external file*, and we had
not decided how that file is referenced.

Today imported files are referenced **by their absolute filesystem path**, and the file is
**re-read from disk on every reopen**:

- An imported mesh/STEP body records its source as a path + format + body index. The
  `*topo.Body` is not serialized (it cannot round-trip through YAML); instead the recipe
  stores `ImportData{Path, Format, Index}` and `restoreImportedBody` calls
  `ImportBodies(format, path)` — i.e. `os.ReadFile(path)` — again on open
  (`model/feature/serialize_import.go`, `model/feature/imported_body.go`).
- A sketch text box / emboss records a `fontFamily` string (`model/sketch/serialize.go`
  `FontFamily`, `model/feature/serialize_emboss.go`), and the font is resolved at recompute
  time against a host catalogue — i.e. the glyph source is, again, an external file located
  at open time.

This makes a document **not self-contained**. Move the `.obk` to another machine, hand it to
a reviewer, check it into a fresh clone, or simply rename the source `bracket.step`, and the
import breaks: `restoreImportedBody` errors loudly ("read … no such file"), which is correct
behaviour but a bad experience. The whole point of ADR-0020 — `cat foo.obk` is a complete,
portable, reviewable document that lives happily in git — is undercut the moment a feature
points at a path outside the file.

We want imported resources to be **part of the document**:

- **Self-contained / portable** — one file carries everything it needs; no companion
  directory, no broken external paths after a move, rename, or clone.
- **Reproducible** — reopening on any machine, with no access to the original source files,
  rebuilds the exact same model. The bytes the model was built from travel with the model.
- **Git-friendly** — a resource that *is* text (OBJ, ASCII STL, STEP) stays line-diffable
  inside the YAML, consistent with the SVG / vector-as-text rule of ADR-0020.
- **No silent dependence on host state** — a font or mesh the user imported is pinned in the
  document, not re-resolved against whatever happens to be installed on the opening machine.

## Decision

Imported files are **embedded in the document** in a single dictionary named `resources` at
the **root** of the `.obk` file (a sibling of `schemaVersion` / `documentType` / `model` /
`data`, per the `onDisk` root in `persistence/yamlcodec`). Each entry is **keyed by a unique
UUID** generated when the file is imported; features reference a resource **by that UUID** —
never by filename, never by position.

```yaml
schemaVersion: 2
documentType: 1               # uint32 enum (1 = Part); serialized as a number, not a name
displayName: bracket
resources:
  9f1c2e7a-5b40-4d3e-9a21-2c7e0b8f4d61:
    type: StepFile            # extensible resource type tag
    encoding: utf8            # how `value` is encoded: utf8 (verbatim text) | base64
    origin: bracket.step      # optional: the original filename, for display/round-trip
    value: |                  # text content, embedded verbatim (document encoding)
      ISO-10303-21;
      HEADER;
      ...
      END-ISO-10303-21;
  3d6b1f08-9e2a-4c11-8f7d-1a5e6c904b22:
    type: TrueTypeFont        # a binary file
    encoding: base64
    origin: Arial.ttf
    value: AAEAAAASAQAABAAg... # base64-encoded bytes
model:
  features:
    - kind: importedBody
      resource: 9f1c2e7a-5b40-4d3e-9a21-2c7e0b8f4d61   # cite the resource BY UUID
      bodyIndex: 0
    - kind: emboss
      font: 3d6b1f08-9e2a-4c11-8f7d-1a5e6c904b22       # the TrueTypeFont resource
      ...
```

Rules:

1. **One root `resources` dictionary, keyed by UUID.** Every imported file the document
   depends on is an entry, keyed by a unique UUID minted at import time. It is the document's
   resource table; nothing outside the `.obk` is required to reopen it.

2. **Each entry has a `type` tag.** A string discriminator — e.g. `"ObjFile"`, `"StepFile"`,
   `"StlFile"`, `"TrueTypeFont"`. The set is **extensible**: new importers register new type
   tags (3MF, other font formats, raster textures, …) without changing the container shape.
   An unknown `type` fails Save/Open loudly (naming the offending tag), matching the
   no-silent-data-loss rule of ADR-0020 §5.

3. **Each entry has an `encoding` tag — `value` is decoded by `encoding`, never inferred from
   `type`.** `utf8` means `value` is the file's text embedded verbatim in the document
   encoding; `base64` means `value` is the file's bytes base64-encoded; `embedded` marks an
   APP-PROVIDED resource that carries **no `value`** (see rule 3a). This is a separate field
   because a `type` does **not** fix the encoding: an STL may be ASCII *or* binary under one
   `StlFile` tag, and a 3MF (a ZIP) is always binary. A writer may also choose `base64` for
   text that would not round-trip cleanly through a YAML block scalar (trailing whitespace,
   tabs, unusual bytes); readers always honour `encoding`.

3a. **`encoding: embedded` records an app-provided resource without its bytes.** When the
   application itself bundles the bytes — e.g. a vendored font face the build always ships — a
   document that uses it stores only the entry (`type`, `encoding: embedded`, and `origin` = the
   face id/family), **no `value`**. The resolver supplies the bytes from the application by
   `origin`. This keeps the resource table the single, uniform place every imported dependency
   is listed (so the document records *which* bundled face it relies on) while not bloating the
   file with bytes the app already has. A host-installed (non-bundled) font, by contrast, is
   embedded as a normal `base64` resource so the document stays self-contained.

4. **`value` carries the bytes.** For `encoding: utf8`, a multi-line block scalar embedded
   verbatim — OBJ, ASCII STL, STEP stay human-readable and line-diffable, the same reasoning
   that made vector references SVG in ADR-0020. For `encoding: base64`, a single base64 string
   — the same opaque-but-self-contained treatment ADR-0020 already grants the `data` map
   (which base64-encodes its values today).

5. **Features reference resources by UUID, not by filename or position.** A feature that
   consumes an imported file stores the resource's UUID string under a role-named field
   (`resource` for a body import, `font` for emboss, etc.). The filename is *not* the binding
   key; the array position is *not* the binding key. Two features may share one resource by
   citing the same UUID, and a resource may be renamed at its source with no effect.

6. **`origin` is optional metadata.** A resource may carry an `origin` property storing the
   original filename (for the browser label, for re-export, for the user's orientation). It is
   **never** the binding — resolution is always by UUID. Dropping or editing `origin` does not
   change which bytes a feature gets.

This supersedes path-based import references: `ImportData.Path` and the
re-`os.ReadFile`-on-open behaviour of `restoreImportedBody` are replaced by a resource UUID;
the font `fontFamily`/host-catalogue lookup is replaced (for imported faces) by a
`TrueTypeFont` resource UUID. The interactive importer writes the source bytes into
`resources` once, at import time, instead of recording a path to re-read later.

## Consequences

- **Documents become self-contained and reproducible.** A `.obk` reopens with byte-identical
  geometry on any machine, with the original source files absent, renamed, or moved. The
  broken-external-path failure mode is gone.

- **Text resources stay diffable; large binaries add diff noise.** An ASCII STEP/OBJ/STL diffs
  cleanly line-by-line in git, as intended. A *binary* resource (a multi-MB font or binary
  mesh) lands as a long base64 string, so the document grows by ~4/3 the file size and that
  blob shows as one churned block on any change — the same git-friendliness tradeoff ADR-0020
  accepted for the `data` map, now potentially much larger. This is the deliberate cost of
  portability; documents that import heavy binaries are heavy. Mitigations (out-of-line sidecar
  resources, compression) are left open for a future ADR if real-world document sizes demand it.

- **UUID keys remove the index-stability problem entirely.** Because features bind by a stable
  UUID rather than a position, the table is order-independent: adding a resource never disturbs
  existing references, and **removing one is just deleting its entry** — no renumbering, no
  reindexing of features. A Save must still validate referential integrity: every feature's
  resource UUID resolves to a present entry, and (optionally) warn on orphaned entries no
  feature cites. A feature citing a missing UUID fails Open loudly (naming the UUID), never
  silently binds to the wrong bytes.

- **Identical resources should be deduplicated.** Importing the same `Arial.ttf` for three
  emboss features, or the same mesh twice, should yield **one** `resources` entry shared by
  UUID, not three copies of the base64 blob. The writer decides dedup by content (e.g. a hash
  of the bytes) and reuses the existing entry's UUID; the stored key stays a UUID, so readers
  are unaffected. This keeps the document small and the diff quiet.

- **The codec grows a root section, not a new container.** `persistence/yamlcodec`'s `onDisk`
  struct gains a `Resources map[string]Resource` field alongside `Model`/`Data`; the atomic
  write-temp-then-rename save and the recipe-only invariant of ADR-0020 are unchanged (a
  resource is *imported input*, not realized geometry — it is exactly the source the recipe
  re-derives bodies from, just carried inside the document instead of read from disk).

- **Migration.** Pre-release, the only existing `.obk` files are regenerable test fixtures, so
  no path-based → embedded migration is written; documents are regenerated. The importer is the
  single place that learns to write `resources`, and `restoreImportedBody` /
  text-emboss-restore read from it instead of the filesystem.

- **Testing (fonts).** Tests must NOT depend on whatever fonts happen to be installed on the
  host — CI, dev, macOS, Linux and Windows machines all differ. Prefer the application's
  vendored face (`model/text` embedded) for deterministic font tests (e.g. write its bytes to a
  temp dir to exercise OS enumeration). If a test genuinely must reference a system font by
  family, restrict it to faces present by default on **macOS, Linux AND Windows**; never assume
  a host-specific font.

## Shape

```
bracket.obk
  schemaVersion: 2
  documentType: 1            # uint32 enum (1 = Part)
  resources:                 # root resource table — UUID-keyed imported files
    9f1c2e7a-…: {type: StepFile,     encoding: utf8,   origin: bracket.step, value: "<STEP text, verbatim>"}
    3d6b1f08-…: {type: TrueTypeFont, encoding: base64, origin: Arial.ttf,    value: "<base64 bytes>"}
  model:                     # the recipe; features cite resources by UUID
    features:
      - {kind: importedBody, resource: 9f1c2e7a-…, bodyIndex: 0}
      - {kind: emboss,       font: 3d6b1f08-…, ...}
```
