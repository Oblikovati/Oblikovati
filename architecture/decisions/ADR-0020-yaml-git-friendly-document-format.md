# ADR-0020 — Documents are a single git-friendly YAML file (not a binary ZIP package)

**Status:** accepted (user decision, 2026-06-02) · **Amends:**
[ADR-0006](ADR-0006-no-com-object-model.md) / core/05 (the on-disk format was
specified as a ZIP package of binary streams; it is now one text file).

## Context

The `.obk` document format (architecture core/05) was a **ZIP archive** of named
binary streams (`persistence/io.go` `marshalZip`/`readZip`; manifest `manifest.json`,
model streams `model/*.bin`). A ZIP is opaque to version control: git stores the whole
archive as one binary blob, so diffs, blame, and three-way merges are impossible. Users
want to keep documents **in git** and review changes to a model the way they review
code.

At the same time, real document content must finally persist: a reopened part must
restore its **recipe** (parameters, sketches, the ordered feature program), since this
is a history-based modeler and the realized B-rep is regenerable cache
(see the recipe-persistence work that depends on this ADR).

## Decision

A document is a **single YAML text file** (`.obk`), not a ZIP package. The file holds
**only the recipe** — text, no binary blobs:

```yaml
schemaVersion: 2
documentType: part
displayName: bracket
model:                      # the recipe (parameters / sketches / features)
  parameters:
    - {name: width, kind: user, expression: 4 cm}
  sketches: [ ... ]
  features: [ ... ]
data: {}                    # add-in / attribute text sections (UTF-8; base64 only if a
                            # stream is genuinely non-text — the rare exception)
```

Rules:

1. **One text file per document.** Replaces the ZIP container. Git diffs/merges a
   document line-by-line. Atomic save (write-temp-then-rename) is unchanged.
2. **Recipe only; no binary in the committed file.** Realized geometry (bodies) is never
   persisted — reopen restores the recipe and runs `Recompute()`.
3. **Vector references use SVG.** Where a document references vector graphics, the
   reference is SVG (text), never a raster/binary blob — so it stays diffable.
4. **Thumbnails are generated, not committed.** A thumbnail may be generated next to the
   document on save, but it is **always git-ignored** (a sidecar, never inside the YAML).
   It is pure cache, regenerable on demand.
5. **No silent data loss.** Any sketch entity, constraint, or feature kind without a
   serialization codec makes **Save fail loudly** (naming the offending kind), rather
   than dropping data quietly.

## Why YAML, and the first third-party dependency

YAML (over JSON) for human-authored-adjacent diffs: comments, block scalars, and
compact flow style keep a model readable and a change reviewable. We use
**`gopkg.in/yaml.v3`** (Apache-2.0, GPL-compatible). This is the **first external
dependency of the GPL core module** (previously only the sibling Apache `/api`). Per
the dependency rule (CLAUDE.md) it is wrapped behind a thin, project-owned interface
(`persistence/yamlcodec`) so exactly one package imports the library and a future
swap stays local.

## Consequences

- **`persistence` swaps its container backend** (ZIP → YAML) while keeping the logical
  document model (manifest fields + named sections) and the atomic-save invariant.
  `schemaVersion` bumps **1 → 2**; legacy ZIP `.obk` files are rejected with a clear
  message (pre-release: the only existing `.obk` files are regenerable test fixtures, so
  no ZIP→YAML migration is written).
- **The opaque exception:** reference-key contexts (topological naming) are serialized
  opaque bytes; they are carried as a single base64 string field, justified here as the
  one non-diffable value, because correct geometry rebinding after recompute requires it.
- A repo `.gitignore` rule excludes generated thumbnails.
- `cat foo.obk` is now a readable, reviewable document.

## Shape

```
bracket.obk            # one YAML text file = one document (committed)
bracket.obk.thumb.*    # generated thumbnail (git-ignored, regenerable)
```
