# Git vendor default README

The checker examined Cargo 1.99.0 vendor output for application source
`b0d839808d53ced0ef7ebf03181f089a1e0878bf`. Cargo.lock SHA-256 was
`3e4636962747b157c0283b41f5ddaade0b5f0aad35b8da55b485ca591de4d20d`.
It checked 18 Git packages and 22,972 tracked files. The result remains partial;
see [observation.json](observation.json).

A Cargo-vendored Git workspace fixture confirmed `README.md` for a package
and `../../README.md` for a package inheriting the workspace default. The
checker rejected forged values after the vendor checksums were updated.
Regression tests exclude symlinks, Git submodules and lower-priority names.
No native UI scenario was run.

Only a tracked regular highest-priority README is used for this comparison.
Other default cases, full manifest contents and build/runtime provenance remain
unproven. The result remains `complete: false` and `reusable: false`.
