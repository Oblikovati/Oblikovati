# Explicit project asset reload — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0019/0020 and the existing project-library read boundary.

`FileWorkspace::reload_project_assets` reads both library files for a registered
canonical project path. It replaces that project's library snapshot only after
both candidates pass the same admission used at registration. A failed second
file cannot publish a new first file. Unknown project identities fail before I/O.

Reload uses the registered design-data directory. It does not reload project
configuration, register another project, change the active project, or change
document snapshots and dirty state. Reloading an inactive project changes its
scope only when that project is selected. Document-owned definitions keep their
priority over reloaded project definitions.

Typed-client tests check complete replacement, failure preserving both previous
libraries, inactive-project selection, unknown identity without reads, and
document snapshot/dirty-state preservation. The named filesystem fake counts
reads. Run `cargo test -p oblikovati-automation --test queries project_assets`.

This is an explicit session operation. Library editing and persistent mutation,
native UI controls, and public project-control API integration remain open.
Kernel and exchange code are outside this change. Tests run only on this Linux
host; no push or PR is made.

## Local validation receipt

On 2026-10-13, `make gate` passed across all 13 workspace crates with llvmpipe
and the local Vulkan validation layer. Formatting, clippy, build, tests,
doctests, documentation, SPDX, workflows and architecture checks passed.
Every production crate exceeded 80% coverage; duplication was 0.64%.
The seven project asset integration tests passed, including four new reload
tests. The existing AMD native presentation fault remains open.
