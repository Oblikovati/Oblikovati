# Git vendor copied manifest fields

The checker examined Cargo 1.99.0 vendor output for the locked application
source at `0022d245ac9d722efc58ee523ab8fd6e1bec7347`. Cargo.lock SHA-256 was
`3e4636962747b157c0283b41f5ddaade0b5f0aad35b8da55b485ca591de4d20d`.
It checked 18 Git packages and 22,972 tracked files. The added scope includes
package metadata, default-run and resolver, and root badges and hints.
See [observation.json](observation.json).

A separately created Git package with nonempty package metadata, default-run,
resolver and badges also passed after `cargo vendor --locked --offline`.
Regression tests reject changes to these fields, including a forged vendor
manifest with an updated checksum. These tests did not run a native UI scenario.

README and license-file paths, other unchecked manifest fields, the compiler,
host tools and runtime remain unproven. The result remains `complete: false`
and `reusable: false` and cannot authorize receipt reuse.
