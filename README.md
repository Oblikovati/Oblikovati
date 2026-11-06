# Oblikovati Rust application

This branch contains the Rust application port in progress. It uses the Rust
contract in **Oblikovati.API**. The native Vulkan/TKV3D viewport uses the
**Martensite** repository. The model adapter still consumes the previous
**RustyCADKernel** packages while its Martensite migration is qualified. The
[v0.9 release intake](architecture/rust-port/implementation-martensite-0-9.md)
records the current renderer and authored sketch geometry pins and qualification.

Go-kernel capabilities that support application features and are outside OCCT
must be ported and adapted into Martensite as required kernel extensions.

The branch contains the port plan and consolidated Rust ADRs. All 66 legacy
Go ADRs have a recorded disposition. The Rust build and release tools are present;
model, parameter, file lifecycle, and storage foundations are present. The headless CLI supports document creation, recipe open/save-as, script execution,
manifest inspection, version, and help. The first desktop executable supplies
native window lifetime, saved prism display, camera navigation, and document
startup. Dear ImGui controls create a Part, start and finish an XY sketch, draw
an anchored rectangle, and create or edit its driving size parameter. Native
Horizontal, Vertical and Coincident tools use analytic picks and the typed API.
General Dimension supplies explicit label placement, expression entry and
parameter-driven values. The native
test verifies zero DOF, recomputation, undo, and redo. The complete cube UI
journey, ribbon and model-browser parity, and native file dialogs remain open.
Publication remains blocked. Current dependencies are pinned in `release.toml`
and the adapter manifests; the
[kernel release intake](architecture/rust-port/implementation-kernel-release.md)
records the earlier release probes.

Run `cargo run -p oblikovati-cli -- info path/to/part.opd --json` to inspect a
package manifest without reconstructing geometry or changing the file.

Run `cargo run -p oblikovati-head -- --frames 3` on the local Linux display for a
bounded native window smoke run. Add a native document path to display its
supported stored geometry, for example
`cargo run -p oblikovati-head -- --frames 3 "path/to/part.opd"`.
See [startup behavior and checks](architecture/rust-port/implementation-desktop-startup.md).
See [native parameter authoring evidence](architecture/rust-port/implementation-native-parameters.md).
See [native dimension creation evidence](architecture/rust-port/implementation-native-dimension-tool.md).

- [Implementation evidence](architecture/rust-port/implementation.md): tested
  behavior, source mapping, and remaining work.

- [Build cache](architecture/rust-port/implementation-build-cache.md): pinned Kache,
  shared worktree cache, measured reuse, and cache controls.

- [Build and release commands](RELEASING.md): local gate, CI, native packaging,
  released API/kernel dependencies, versioning, and publication requirements.

- [Port plan](architecture/rust-port/plan.md): scope, architecture, and decisions.
- [Rust ADR policy and index](architecture/decisions/README.md): a fresh decision
  set, with applicable legacy decisions adapted and consolidated by boundary.
- [Legacy ADR map](architecture/decisions/legacy-map.md): fixed source links,
  retained decisions, rejected implementation choices, and remaining evidence.
- [Domain glossary](CONTEXT.md): shared document, feature, and identity terms.
- [Source map](architecture/rust-port/source-map.md): inspected revisions and
  the Go modules that must move or be replaced.
- [Martensite integration](architecture/rust-port/martensite.md): required
  kernel capabilities and their acceptance checks.
- [Milestones](architecture/rust-port/milestones.md): work order, dependencies,
  and completion criteria.
- [Verification and release](architecture/rust-port/verification.md): parity,
  live tests, platform checks, and cutover.

`rust` is the repository's default and integration branch. It has an independent
root commit. Topic branches retain the implementation commits before integration.

`go-legacy` preserves the Go application history. It was the `develop` default
branch before the branch change on 2026-10-10; the repository had no `main`
branch. Keep it as the port reference. Do not merge its unrelated history into
`rust`.
