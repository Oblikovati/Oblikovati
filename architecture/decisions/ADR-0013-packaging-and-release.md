# ADR-0013: Packaging and release transition

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R9; publication remains disabled.

Boundary: distributable application artifacts, updates, and Go-to-Rust cutover.
Owner: application release tooling; API and Martensite release independently.

## Context

The new `rust` branch has independent history and no implementation yet. The
Go release scripts assume Go binaries, native dependencies, and particular
branches. Reusing them unchanged could publish an incomplete replacement.

## Decision

Keep `rust` as the new integration branch and the Go checkout as a fixed
reference. Feature branches target `rust` once it is published. Fetch upstream
for comparison; do not merge unrelated Go history to populate this branch.
Do not activate application release publishing during architecture migration.

Retain development-preview and stable delivery as separate channels. Give Rust
previews distinct artifact/update identities so they cannot replace the Go
stable installation. Before enabling R9 publication, define and test the
trigger branches, tags, promotion procedure, and rollback path. Port the
baseline application version behavior where clients depend on it; do not
silently reinterpret its four-component release label as Cargo/API SemVer.

Record application version, application commit, API version/commit, Martensite
commit, document schema, reference-key scheme, and C ABI compatibility in the
release manifest. Cargo package versions use their required format; the API
retains its own Semantic Versioning and dual-language release checks. A
dependency pin does not prove that the application implements its full API.

Package GUI and CLI with their required native libraries, assets, shaders,
fonts, templates, catalogs, notices, and file associations. Use target-native
build and runtime checks for the desktop. Inventory each OS/architecture from
the Go release and report which Rust packages have passed. Do not label an
absent platform artifact as a successful complete release.

macOS distribution needs a tested self-contained app bundle with the Vulkan
loader/MoltenVK, correct library paths, and signing/notarization appropriate
to the channel. Linux packaging must work outside the build checkout; Windows
must find its intended runtime DLLs. Missing release credentials block that
platform's publication; development builds remain possible. Validate actual
platform requirements during R9 instead of copying old SDK assumptions.

## Interfaces and invariants

- A published dependency must be fetchable by clean CI or supplied as a
  checksummed, reproducible source artifact. A local Martensite commit ahead
  of its remote is not a reproducible release dependency by itself.
- API release must pass Go/Rust definition, version, example/documentation,
  and parity gates in its owning repository. Application packaging checks
  that it consumes that exact verified contract.
- Preview tests open copies of user projects and isolate configuration writes
  from a concurrent Go installation. The final user configuration convention
  remains [ADR-0003](ADR-0003-documents-and-storage.md); any migration is explicit.
- Test install, update, uninstall, reopening saved designs, and return to the
  previous Go build. A new file/key schema requires preserved readable backups
  and a tested migration. Never overwrite a design that cannot be represented.
- Add-ins remain separately distributed. Test both Go and Rust add-ins against
  the packaged host; signing/loading requirements must be checked with real
  third-party libraries, not inferred from the main executable's success.
- Go production release retirement is a tracked cutover after all required
  behavior and kernel-extension gates pass. Keeping the reference source and
  fixtures does not keep a second production kernel in the Rust binary.

## Alternatives and consequences

Publishing on every push to the new branch could replace a usable Go build
with an incomplete port. A single platform-neutral archive could omit native
dependencies. Reject both. Isolated previews and native package tests add
release work but make installation and rollback reviewable before promotion.

## Legacy disposition

Consolidate Oblikovati ADR-0017 and distribution sections of 0019 at the
[fixed revision](legacy-map.md#source-baseline). Retain GUI/CLI artifacts,
channel separation, bundled resources, and platform validation. Replace Go
commands and branch-specific triggers. Do not carry forward unsigned macOS
tarball assumptions or silently skipped stable platform artifacts.

## Validation required

Use Go `RELEASING.md`, version tooling, workflows, and packaging scripts as
behavior evidence. R9 runs clean-source reproducibility, native installation,
dependency discovery, add-in loading, update/rollback, and document migration
tests after [ADR-0012](ADR-0012-build-and-validation.md)'s full gate. This
migration creates no workflow, release tag, upload, or publication trigger.
