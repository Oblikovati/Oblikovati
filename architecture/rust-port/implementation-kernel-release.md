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

## v0.2.1 local intake

The application now pins [kernel v0.2.1](https://github.com/Oblikovati/RustyCADKernel/releases/tag/v0.2.1)
at `193572dc3665598c4948777c87375bad7dc8114a`. The annotated tag resolves
to this commit and agrees with the published manifest. All 35 payloads and
`martensite-release.json` passed the published SHA-256 checks. The manifest
records `extracted-workspace-tests-and-doctests`. The 31 crate archives also
identify this exact commit with clean source metadata. Kernel tests and publication
remain the kernel owner's work.

All 15 consumed kernel packages use this exact version and revision.
`release.toml`, both consumer manifests, and `Cargo.lock` agree. Other dependency
versions remain unchanged. The Rust toolchain remains 1.99.0. Local builds use
the released Cargo source checkout and do not use the active sibling repository.
The `set_center` equal-distance behavior still exists in this release; the
application keeps its existing camera update path and its original comment.

Acceptance is local Linux only. No push, PR, remote test, or workflow run is
part of this intake. Windows and macOS remain deferred with explicit failing
workflow paths. This release does not close the remaining R1 feature scope.

## v0.2.5 local intake

The application pins [kernel v0.2.5](https://github.com/Oblikovati/RustyCADKernel/releases/tag/v0.2.5)
at `8118c7e1fae4d28bb5bb5656cd8f86445dadef10`. The published tag and
release manifest agree. All 35 payloads and the manifest passed the published
SHA-256 checks. All 31 crate archives record this revision and clean source
metadata. The manifest records `extracted-workspace-tests-and-doctests`.

Both consumer manifests, the lockfile, packaging configuration, and pin regression
test use this release. The 16 consumed kernel packages come from the exact released
Git revision. The release adds `num-bigint` and `num-integer` as transitive
requirements. Existing registry versions and the API v0.158.0 pin stay unchanged.
The Rust toolchain remains 1.99.0. No sibling checkout supplies kernel code.

This release does not contain the separate sketch spline constructor commit
`bab95916c`. Kernel issue [#9](https://github.com/Oblikovati/RustyCADKernel/issues/9)
remains open for construction, handles, release, and application acceptance.
This dependency update does not complete that work or the full R1 scope.

The viewport now retains `Handle<dyn V3dViewSource>`, as returned by the
released viewer factory. This preserves the kernel's view interface and its
virtual dispatch. The initial build exposed the old concrete handle type;
no geometry or camera fallback was added. Existing native tests exercise view
creation, camera changes, navigation, resize, presentation, and destruction.

Local Linux acceptance passed `make gate` over all 13 application crates,
all targets and features. It includes formatting, Clippy, build, tests,
doctests, coverage, Markdown, documentation links, SPDX, workflow checks,
and duplication. Native tests ran serially on a 1920×1080×24 Xvfb display
with software Vulkan and validation layer 1.4.321. They check GPU captures
against presented X11 pixels, including navigation and document edits.

Every crate exceeded 80% line coverage. The kernel adapter reached 90.16%,
the viewport 93.52%, and the native head 95.01%. Duplication was 0.75%
(510 of 67,647 scanned lines). The compatibility change adds no numeric
constant or platform-specific path; the view remains in the existing Linux
renderer boundary. Existing comments and camera behavior remain intact.
Windows and macOS acceptance remains deferred. No push, PR, or publication
is part of this intake.

## v0.2.7 local intake

The application pins [kernel v0.2.7](https://github.com/Oblikovati/RustyCADKernel/releases/tag/v0.2.7)
at `d40ed71b704efd08310a4c56a1c1c3a64d4e3e9b`. The annotated tag,
published manifest and all 31 crate archives identify this revision. The archives
record clean source metadata. The manifest records
`extracted-workspace-tests-and-doctests`. Release workflow
[38511000655](https://github.com/Oblikovati/RustyCADKernel/actions/runs/38511000655)
completed successfully before application adoption.

All 35 payloads and the manifest passed the published SHA-256 checks. All 37
published files, including `SHA256SUMS`, also match the completed workflow's
package artifact byte for byte. The local checked download is
`/tmp/oblikovati-kernel-v0.2.7-published`.

Both consumer manifests, the lockfile, packaging configuration and pin regression
test agree. All 16 consumed kernel packages use the exact release revision. Every
other locked package remains unchanged, including API v0.158.0. The Rust toolchain
remains 1.99.0. Cargo reads the released Git source; no sibling checkout supplies
code or binaries.

This release includes native GUI composition and the owned spline and exact
predicate prerequisites. Their application integration remains separate work.
The release notes explicitly leave the general sketch-region-to-solid certificate
incomplete. Internal selected-prism diagnostics are not a certified application
body. The kernel blocker priorities remain until the relevant consuming feature
passes its acceptance tests. This pin change alone does not complete R1, R3 or R4.

The complete local Linux `make gate` passed for all 13 application crates,
all targets and features. Formatting, Clippy, build, tests, doctests, coverage,
Markdown, documentation links, SPDX, workflow checks and duplication passed.
Native tests used Xvfb at 1920×1080×24 with software Vulkan and checked captured
pixels against the presented X11 window. The retained log is
`/tmp/kernel-v0.2.7-gate.log`.

Every crate exceeded 80% line coverage. The adapter reached 91.67%, viewport
94.28% and head 96.19%. Duplication remains 0.77% (577 of 74,756 scanned lines).
No application compatibility fix was needed. This change adds no numeric or
platform-specific constant. The existing camera and preview Fit comments remain
valid against the released source and were retained. Windows and macOS remain
deferred. Integration is local only; no push, PR or application publication is
part of this intake.
