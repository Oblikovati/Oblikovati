# Embedded material catalog — partial R5/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0010. The source is Go `model/material` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The Rust application now embeds the complete Go catalog: 126 physical materials
and 94 appearances, including the neutral default. The original seven category
files are split at asset boundaries into files under 500 lines. Property values,
units, source citations, directional-axis explanations, and derivation comments
are retained. The neutral appearance is exported from the Go asset itself;
Rust API struct zero values are not OpenPBR defaults.

Storage decodes these embedded YAML bytes with the existing project-owned codec
boundary. Only stored field names are adapted: `name`, `isotropy`, `orthotropic`,
and `appearance` become the released API record names. All grouped values use
the API types. Explicit source overrides and conflicting alias names are refused.
There is no new codec dependency, filesystem lookup, working-directory fallback,
or startup panic in the production loader.

The model admits the entire immutable catalog together. IDs must be unique within
each asset family, the neutral appearance must exist, and every material must name
a shipped appearance. Built-in names and source policy are checked. The Go catalog
admission requirements are retained: positive physical properties, complete
positive directional moduli with an anisotropic tag, complete magnetic constitutive
values, bounded metalness/roughness, and visible opacity. These are catalog
admission checks; positive-definite stiffness, complete simulation qualification,
and analytic mass consumers still require their own acceptance evidence.

`AssetCollection` retains both stable ID iteration for storage and insertion order
for catalog display. Replacing a snapshot does not move it. `BuiltinAssetCatalog`
exposes borrowed immutable collections that can enter the document/project/builtin
resolver. It owns no I/O, renderer, session, or geometry calculation.

## Independent baseline evidence

`crates/persistence/tests/catalog/go_oracle.go` exports the Go library through
its public accessors into released API DTOs. It refuses a different Go HEAD.
`go-baseline.json` stores that export and the SHA-256 receipts for the seven
original category sources. Recreate the export from the fixed Go checkout:

```sh
go run ../oblikovati-rust/crates/persistence/tests/catalog/go_oracle.go
```

Rust tests compare every asset record and both complete display-order sequences
with the independent Go export, including the normalized isotropy read contract.
The comparison first exposed a missing `appearance` field adaptation, which was
fixed in the shared decoder. Counts and exact grouped values are hard assertions,
not catalog-size smoke tests. Optional defaults are compared too.

Tests also check read-only edits, required grades, duplicate IDs, missing links,
positive-field admission, directional/magnetic classification, appearance ranges,
malformed input, source overrides, alias collisions, finite values, and the exact
embedded file inventory. The Go material suite and the checked exporter pass on
this host. Run `cargo test -p oblikovati-persistence --test catalog`, then
`make gate` for full local Linux acceptance.

Session startup attachment, wire routing, editable project catalogs and revisions,
document recipe integration, assignment resolution, asset editors, appearance
conversion, material rendering, and analytic mass properties remain open. Catalog
loading does not close R5/R7 or resolve the ADR-0010/Go #1103 assignment-order
difference. Original rendering comments are legacy provenance; they do not certify
the Rust render path.

No kernel or exchange code is added. Windows/macOS are deferred; no push or PR is
included.

## Local validation receipt

On 2026-10-12, the full `make gate` passed for all 13 workspace crates with
llvmpipe and the local Vulkan validation layer. Model production line coverage
was 98.82%, persistence was 96.86%, and every crate exceeded 80%. Repository
duplication was 0.62%. Formatting, clippy, build, tests, doctests,
documentation, SPDX, workflow and architecture checks passed. The twelve catalog
integration tests and embedded-file inventory test passed. The existing AMD
native presentation fault remains open.
