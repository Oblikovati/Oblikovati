# Git vendor inherited path metadata

The checker examined Cargo 1.99.0 vendor output for application source
`ae733c5a1ccfa2f0e05b2dcfe54b3d18e1104636`. Cargo.lock SHA-256 was
`3e4636962747b157c0283b41f5ddaade0b5f0aad35b8da55b485ca591de4d20d`.
It checked 18 Git packages and 22,972 tracked files. The result remains partial;
see [observation.json](observation.json).

A Cargo-vendored nested Git workspace fixture inherited README and license-file
paths from its root. Cargo generated `../../doc/README.md` and `../../LICENSE`;
the checker accepted those values and rejected each forged path after its vendor
checksum was updated. Regression tests also check a dirty workspace checkout
against its locked Git object. No native UI scenario was run.

Only portable relative workspace paths are supported. Default README discovery,
other manifest fields, compiler and runtime inputs remain unproven. The result
remains `complete: false` and `reusable: false`.
