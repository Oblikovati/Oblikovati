# ADR-0034 — Per-document-type file extensions, and a project file

**Status:** accepted (user decision, 2026-06-13) · **Amends:**
[ADR-0020](ADR-0020-yaml-git-friendly-document-format.md) / core/05 (the single
`.obk` extension is replaced by a per-type extension; the YAML text-file format
and the manifest are otherwise unchanged).

## Context

ADR-0020 made a document **one YAML text file** with a single `.obk` extension
for every kind. The manifest's `documentType` is the canonical identity, so the
extension was deliberately type-agnostic — one extension, kind read from inside.

M11 introduces the **Assembly** document as a second real document kind alongside
**Part**. With more than one kind on disk, a single shared extension has real
costs:

- the OS cannot associate a per-kind action or icon;
- a reference (assembly → part, drawing → model) cannot tell a referenced file's
  kind from its **name** — it must open the file and read its manifest first;
- a user cannot tell a part from an assembly in a file browser.

Separately, a **design project** — the portable search-path config that resolves a
document's referenced files (core/05, `doc.DesignProject`) — has no file format at
all today; it lives only in memory. Assemblies will need to load one from disk.

The reference API has always used per-type extensions; ADR-0020/core-05 departed
from that for git-friendliness, but kept the *container* concern (YAML, one text
file) separate from the *naming* concern. This ADR revisits only the naming.

## Decision

Each document kind carries its own user-facing extension; the project gets one too:

| Extension | Kind                       | DocumentType           | Rich handling lands in |
| --------- | -------------------------- | ---------------------- | ---------------------- |
| `.opd`    | Part document              | `DocumentPart`         | shipped (M07)          |
| `.oad`    | Assembly document          | `DocumentAssembly`     | M11 / M12              |
| `.odd`    | Drawing document           | `DocumentDrawing`      | M14                    |
| `.ord`    | Presentation document      | `DocumentPresentation` | M16                    |
| `.opj`    | Design project (not a doc) | —                      | basic now; used by M11 |

Rules:

1. **The on-disk YAML format and manifest are unchanged** from ADR-0020. The
   manifest's `documentType` stays the **canonical identity**; the extension
   mirrors it and must agree. On open the manifest wins — the extension is a hint
   for the OS and the reference graph, never the source of truth.
2. **The extension is derived from the type in exactly one place.**
   `types.DocumentType.Extension()` and the inverse
   `types.DocumentTypeFromExtension()` live in the Apache-2.0 contract (ADR-0018).
   No other code hard-codes an extension string; the GPL side aliases the project
   extension as `doc.ProjectExtension`.
3. **The project file (`.opj`) is a separate concept** — it is not a document and
   has no `DocumentType`. It is its own YAML text file (the ADR-0020 shape applies:
   one readable, git-diffable text file) holding the project's name, workspace,
   library roots, and template/design-data locations.
4. **No legacy `.obk` support.** This is alpha and the only `.obk` files were
   regenerable fixtures, so there is no migration shim — consistent with ADR-0020's
   stance on pre-v2 ZIP packages.

## Consequences

- `doc.PackageExtension` (the single `.obk` constant) is **removed**. Call sites
  derive the extension from the document's kind (`d.DocumentType().Extension()`),
  and the "has this been saved to a real path yet?" gate becomes
  `doc.HasDocumentExtension(name)` (true for any of the four document extensions).
- **File dialogs** browse and filter the four document extensions; **Save As**
  appends the *active document's own* extension. The **CLI** `new` / `import` /
  `save-as` stamp the correct per-type extension.
- **Templates** are named `<type><ext>` — e.g. `part.opd`, `assembly.oad`.
- **Basic `.opj` read/write** ships in `persistence`
  (`WriteProjectFile` / `ReadProjectFile`). Richer project settings (library/style
  configurations, version-control bindings) are deferred behind `TODO(M14)`;
  assembly modeling (M11/M12) is the first consumer that binds a project loaded
  from disk — `TODO(M11)`.
- Documents of all four kinds already round-trip through the manifest-driven
  store; this ADR makes each kind reachable **end to end under its own extension**.
  Content depth for assembly/drawing/presentation fills in at their milestones,
  already marked on the `doc.DocumentType` constants.

## Shape

```
bracket.opd            # a part document (YAML text)
frame.oad              # an assembly document
frame.odd              # a drawing of the assembly
frame.ord              # a presentation of the assembly
Acme.opj               # the design project that resolves their references (YAML text)
```
