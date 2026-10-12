# Legacy appearance migration — partial R5/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0019/0020. The fixed Go `model/material/legacy_migrate.go` and
`recipe.go` define the old scalar appearance conversion.

Document storage now reads the old hex albedo, metallic, roughness, hex emissive
and opacity fields into the released OpenPBR groups. Base and specular remain
weighted; opacity and roughness retain their stored values. Emission uses the
legacy linear color's largest channel as luminance and normalizes color by it.
Black or absent emission remains exactly off. Every additional lobe retains its
complete zero off-state, not the unrelated full OpenPBR default groups.

The old renderer used gamma 2.2. The migration retains that existing numeric
meaning; it does not substitute the modern piecewise sRGB transfer curve or add
a working-space transform. Hex data has an exhaustive eight-bit channel domain.
The 256 Go-generated IEEE float bits are frozen in the migration table, avoiding
platform `pow` differences. This is a complete transfer domain, not input-case
recognition. Alpha bytes are validated but opacity remains its independent
legacy field, as in Go.

The independent Go exporter verifies the fixed HEAD and uses the public YAML
recipe migration. Its 256 colored cases exercise all channel values and the
emission split; Rust compares every resulting API group exactly. Tests separately
cover black emission, invalid/Unicode hex, scalar range/finiteness, mixed legacy
and grouped shapes, scope overrides, extension retention and repeated capture.
The exporter and receipt live in `crates/persistence/tests/legacy_appearance/`.
Run it from the fixed Go checkout with:

```sh
go run ../oblikovati-rust/crates/persistence/tests/legacy_appearance/go_oracle.go
```

A successful snapshot capture rewrites legacy records into grouped data even
when their typed values were not edited. Only consumed legacy fields are removed;
opaque extensions survive. The candidate must restore to the exact same typed
asset state before adoption. A real CLI Lua test reads an old appearance through
the normal document API and saves its grouped replacement. No new runtime codec
or dependency is introduced.

Conversion does not certify rendered color, physical analysis, project library
loading, or complete asset mutation APIs. The old numeric working-space convention
is retained intentionally; rendering conformance remains a separate acceptance
item. All work is local Linux, with Windows/macOS deferred, no kernel/exchange
code changes, and no push or PR.

## Local validation receipt

On 2026-10-13, `make gate` passed for all 13 workspace crates with llvmpipe and
the local Vulkan validation layer. Persistence production coverage was 96.63%;
every crate exceeded 80%. Repository duplication was
0.63%. Formatting, clippy, build,
tests, doctests, documentation, SPDX, workflow and architecture checks passed.
The exhaustive Go migration comparison, four targeted migration tests and real
CLI load/save upgrade passed. The AMD native presentation fault remains open.
