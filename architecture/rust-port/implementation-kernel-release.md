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

## Local pin consistency

The local gate now validates both API and kernel entries in `release.toml`
against every consumed owner crate in `Cargo.lock`, before external commands.
This reuses the release preflight's version, source-owner and commit checks.
A stale pin fails even while product publication is still disabled. The check
uses local files and does not rerun owner tests or query release services.

A regression test first reproduced the stale API v0.157.0 packaging entry against
an application lock using v0.158.0. The corrected API pin is v0.158.0 at
`32ff38398ab6d6a7df0ca838654d5b8bc8d8b9cd`, verified against its published tag.
Tests check the real repository lock for both owners, and prove that stale API
or kernel commits stop the gate before any external check starts. The successful
gate fixture still schedules workspace checks, documentation, workflow lint and
duplication in their original order. Publication readiness remains a separate gate.

The [retained shape isolation probe](implementation-shape-snapshot.md) now keeps
an opaque detached box snapshot. It adds ownership and OCCT comparison evidence;
it does not establish the full R1 feature-result or validity contract.
