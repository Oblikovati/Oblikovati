# ADR-0014: Document recipes and storage

Date: 2026-10-10. Status: Accepted for the Rust target design.
Supersedes [ADR-0003](ADR-0003-documents-and-storage.md), preserving its storage
boundary and decisions except for the schema extension below.
Implementation evidence: partial R2; R3/R6/R8 remain pending.

Boundary: model persistence, resources, projects, and user configuration.
Owner: model and storage modules in `oblikovati-rust`.

## Context

The Go application stores editable recipes in YAML. Rust and Martensite will
have different in-memory types. A file must preserve the design rather than
depend on a particular kernel object layout or the author's machine.

## Decision

Read and write the baseline schema-2 YAML recipe. Preserve the manifest's
numeric document kind, stable IDs, feature discriminators, parameters,
expressions, attributes, and named add-in sections. Public type and extension
mappings come from the API. Rust enum discriminants and type names are not
storage IDs. Resolve relationships through stable IDs and owned collections.

Keep `.opd` parts, `.oad` assemblies, `.odd` drawings, `.ord` presentations,
and the separate `.opj` project. The manifest is authoritative when opening;
the extension is a hint. Save As derives the extension from the kind. A design
project owns search locations and referenced-document resolution policy; an
assembly can still depend on other document files.

Persist recipes and imported inputs, not evaluated Martensite shapes or display
meshes. Thumbnails remain regenerable sidecars. A root resource table retains
UUID keys, type, encoding, origin, and content. Decode `utf8`, `base64`, and
application-provided `embedded` resources by encoding, not filename. Preserve
text bytes or use base64 when text encoding cannot preserve them exactly.
Deduplicate identical inputs without changing existing UUID references.

An imported file's original path is display metadata, not a dependency to read
again at reopen. An `embedded` font/resource must resolve to the expected bundled
content; a missing resource is an error, not a substitution from installed
fonts. The resource table makes imported inputs portable. It does not inline
all referenced component documents into an assembly.

Wrap YAML, filesystem access, and OS path resolution behind application-owned
interfaces. Use one configuration-root resolver: `~/.oblikovati` on Linux/macOS
and the roaming AppData `oblikovati` directory on Windows. Retain separate
preferences, view-state, window-state, and theme stores. Preview installation
isolation follows [ADR-0013](ADR-0013-packaging-and-release.md).

## Schema-3 derived-link extension

Schema 2 reconnects each source link to a local parameter with the same name.
A local rename cannot round-trip through that representation. Schema 3 adds an
optional `localNames` mapping on each derived-table recipe, from source name to
local name. Omitted entries retain same-name lookup. Explicit entries must name
selected links and existing, uniquely owned derived parameters. The mapping is
validated before any table is adopted.

Continue writing schema 2 when no extension is needed. Promote to schema 3 when
capturing a renamed link; retain schema 3 on subsequent saves. Never write this
extension under schema 2, and reject it when restoring a schema-2 recipe. The
baseline Go reader does not support schema 3. There is no implicit downgrade or
claim of a lossless Go rewrite for extended documents.

This replaces the Rust save refusal for renamed links. Keeping that refusal
would leave a supported edit unsavable. Adding ignored fields under schema 2
would allow an older reader to lose link identity without a clear version error.
No geometry, evaluated health, or source snapshot is embedded by this extension.
Reservation marks and general reference-key evolution need their own evidence.

Validation: schema-2 fixtures, schema-3 rename/save/reopen/refresh, invalid
mapping ownership and names, unknown-version refusal, and the full Linux gate.
See [implementation evidence](../rust-port/implementation-derived-tables.md).

## Interfaces and invariants

- Save writes a complete validated temporary file and atomically replaces the
  destination using a tested platform adapter. A failed save preserves the old
  file. Return a specific error for a kind or field that cannot round-trip.
- Keep schema, key, and API versions separate. Unknown required content or
  unsupported versions cannot be silently discarded on save. Reject unsafe
  rewrites and preserve the source file for a supported migration.
- Preserve all non-default assembly feature inputs and bind occurrences before
  their dependent features. New Rust relationships use stable IDs; translation
  from legacy indices/names requires an explicit unambiguous mapping.
- Preserve stored working scale, dimensions, and resource references. Units
  follow [ADR-0006](ADR-0006-parameters-and-constraints.md); reference migration
  follows [ADR-0005](ADR-0005-identity-and-references.md).
- Earlier ZIP/schema-1 and `.obk` forms are not newly promised compatibility.
  Record accepted/rejected legacy forms from fixtures. Keep clear errors for
  unsupported forms instead of inheriting old "regenerate all files" shortcuts.

## Alternatives and consequences

A binary shape snapshot would bind the file to kernel internals and lose
editable history. External source paths would make imported recipes depend
on another machine. Reject both. YAML and embedded binary inputs can make
large files; that cost is accepted for portability and readable recipe diffs.
Stable ordering is required, but preserving arbitrary YAML comment layout is
not inferred from semantic round-trip compatibility.

## Legacy disposition

Consolidate Oblikovati ADR-0006, 0020, 0029, 0031, 0034, and the serialization
parts of 0035 at the [fixed revision](legacy-map.md#source-baseline). Resource
embedding qualifies 0020's original no-binary rule. Theme XML supersedes theme
YAML under [ADR-0009](ADR-0009-ui-and-selection.md). Do not preserve Go closures
as runtime state or mistake evaluated scalar snapshots for retained formulas.

## Validation required

Use Go `persistence/manifest.go`, `yamlcodec/`, model codecs, and
the baseline document corpus. R2/R3 must test all field shapes, unknown kinds,
atomic-write failures, Save As/Copy, resources without source files, and old
key decoding. R6 covers all document kinds, repeated occurrences, missing
components, and project paths. Choose a Rust YAML dependency only after these
fixtures and its current toolchain/license checks pass.
