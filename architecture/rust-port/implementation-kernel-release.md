# Martensite release intake — partial R1

Tracking: [#3786](https://github.com/Oblikovati/Oblikovati/issues/3786).
The owner published [v0.1.2](https://github.com/Oblikovati/RustyCADKernel/releases/tag/v0.1.2)
at `19108f6650cdfca7050febc759d91bdfec001769`.
Its release workflow `38146970967` completed successfully. The published
`martensite-release.json` records `extracted-workspace-tests-and-doctests`.
All downloaded assets passed `sha256sum --check SHA256SUMS`.

The application uses exact version and Git revision dependencies in
`crates/martensite/Cargo.toml`, with all transitive toolkit versions and revisions
recorded in `Cargo.lock`. No sibling checkout supplies source or binaries.
The adapter imports the modeling toolkits, rather than the foundation-only facade.
Application builds, local gates, and CI setup use the owner's tested Rust 1.99.0.
The API now uses its separate v0.158.0 release for independent ellipse radius
dimensions; see [ellipse implementation evidence](implementation-ellipse.md).

## Application acceptance probe

`oblikovati-martensite` establishes the adapter boundary in ADR-0007. Its narrow
`probe_box_release` entry point constructs a fresh solid and checks geometric
validity before querying analytic volume and centroid. Its shape handles never
leave the call. Input lengths are supplied working units with no hidden conversion.
No tessellation is created. This is release consumption evidence, not an
application box feature or a general feature evaluator.

Tests vary scale, repeat independent evaluations, reject non-finite and
non-positive lengths, and check a below-resolution kernel decline. They check
analytic values against exact rectangular-solid formulas. Volume is a smoke test;
it does not replace the required per-face oracle, provenance, or topology gates.
Run `make gate` for the complete application workspace checks.

## Remaining scope

R1 still requires general profile evaluation, body ownership and isolation,
provenance, diagnostics, tolerance policy, and per-face acceptance. No mutable
body is exposed from this probe. R1V native window and shared UI composition
remain [#3630](https://github.com/Oblikovati/Oblikovati/issues/3630).
The kernel owner retains kernel testing and package publication. Application
publication remains blocked by its product-readiness fields. Windows and macOS
jobs retain their explicit deferred failures; this work checks Linux only.
