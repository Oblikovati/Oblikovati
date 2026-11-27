# Build source discovery

Issue #3971 excludes `.slim/worktrees` directories from source discovery. It
keeps source checks and Cargo workspace inventory on the active checkout,
including its nested fixture modules. Packaging walks still include all files.
A symlink at the excluded directory remains an error rather than being followed.

The five focused worktree regressions passed. They cover an invalid foreign Rust
file and Cargo manifest, a valid nested module, an active checkout located under
worktree storage, and symlink handling. Before the fix, the source-walk
regression failed because it included the foreign files.

The complete `oblikovati-build` test run on work based at `614ec6f3a` reported 93 passes
and five failures in API-pin and native-installer gate tests. Four failures report
that the lockfile pins API revision `a8e651bc` while the release check expects
`f0943c72`; the fifth fails an installer error-message assertion. The source-discovery
change does not alter either pin. A clean-base
bisect was not performed, so these failures are not classified as pre-existing.
The real build-tool `docs` command stopped on `crates/head/src/native.rs` at the
500-line limit; #3970 tracks that file split. The focused docs regression passed,
but the complete docs command has not passed in this lane.
