# Oblikovati Rust application

This branch will contain the Rust port of the Oblikovati application. It will
use the Rust contract in **Oblikovati.API** and the **Martensite** kernel in
**RustyCADKernel**.

Go-kernel capabilities that support application features and are outside OCCT
must be ported and adapted into Martensite as required kernel extensions.

The branch contains the port plan and 13 consolidated Rust ADRs. All 66 legacy
Go ADRs have a recorded disposition. The Rust build and release tools are present;
model, parameter, file lifecycle, and storage foundations are present. The headless CLI supports document creation, recipe open/save-as, script execution,
manifest inspection, version, and help. The first desktop executable supplies
native window lifetime and event-driven presentation. Model display, Dear ImGui
chrome, and remaining CLI commands are not yet ported. Publication remains blocked. The
[kernel release intake](architecture/rust-port/implementation-kernel-release.md)
pins Martensite v0.1.2 and adds Linux package acceptance at the adapter boundary.

Run `cargo run -p oblikovati-cli -- info path/to/part.opd --json` to inspect a
package manifest without reconstructing geometry or changing the file.

Run `cargo run -p oblikovati-head -- --frames 3` on the local Linux display for a
bounded native window smoke run. The current head shows an empty viewport; it
does not yet show document geometry or editing controls.

- [Implementation evidence](architecture/rust-port/implementation.md): tested
  behavior, source mapping, and remaining work.

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
root commit. Port work continues on the single `rust-port` working branch.
Create a pull request only when the user explicitly requests one.

`go-legacy` preserves the Go application history. It was the `develop` default
branch before the branch change on 2026-10-10; the repository had no `main`
branch. Keep it as the port reference. Do not merge its unrelated history into
`rust`.
