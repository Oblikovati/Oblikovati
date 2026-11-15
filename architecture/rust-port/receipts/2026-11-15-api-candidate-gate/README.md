<!-- SPDX-License-Identifier: GPL-2.0-only -->

# API candidate local gate

On 2026-11-15, `make gate` passed in a clean temporary API worktree at
`276a401141547b34726d9114bda408755331ea7e`. The candidate branch is
`agent/express-release-prep`. The remote `develop` tip was
`f0943c724a6e07ec49533769c9e680137bb3d55c` before and after the run.

The command used a dedicated Cargo target with debug information and
incremental compilation disabled. `gate.log` records the generated-source
check, builds, tests, lint, documentation, C ABI and Go package checks. Its
SHA-256 is `c559d4b97d778ff14567f992e6f7d2ce065231a44532219c18ada4ae61622d62`.
The gate did not modify tracked source files.

This is a local candidate result, not a release or hosted CI result. The
candidate has not been merged or published. The Rust application still requires
the released contract and a new pin before its full gate can pass (#3965).
