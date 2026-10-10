# Oblikovati Rust application

This branch will contain the Rust port of the Oblikovati application. It will
use the Rust contract in **Oblikovati.API** and the **Martensite** kernel in
**RustyCADKernel**.

Go-kernel capabilities that support application features and are outside OCCT
must be ported and adapted into Martensite as required kernel extensions.

The branch currently contains the port plan. Application implementation has not
started.

- [Port plan](architecture/rust-port/plan.md): scope, architecture, and decisions.
- [Source map](architecture/rust-port/source-map.md): inspected revisions and
  the Go modules that must move or be replaced.
- [Martensite integration](architecture/rust-port/martensite.md): required
  kernel capabilities and their acceptance checks.
- [Milestones](architecture/rust-port/milestones.md): work order, dependencies,
  and completion criteria.
- [Verification and release](architecture/rust-port/verification.md): parity,
  live tests, platform checks, and cutover.

`rust` is an orphan branch with an independent root commit. It is a named
branch, not a detached HEAD. Keep the Go application history as the port
reference. Feature branches for the Rust application will start from `rust`.
