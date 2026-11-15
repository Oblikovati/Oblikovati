# Git package metadata and lint observation

The locked source at `2bb7cfa0413654e6cfcd230d5b4155261a1d705c` has
Cargo.lock SHA-256 `3e4636962747b157c0283b41f5ddaade0b5f0aad35b8da55b485ca591de4d20d`.
Cargo 1.99.0 (`5f94df478`, 2026-08-27) generated the vendor directory from
an offline cache. The checker compared 18 Git package manifests and 22,972
tracked files with the locked Git objects. See [observation.json](observation.json).

The checker rejected altered description and lint values in the generated API
manifest even after its Cargo checksum metadata was updated. The full Python
BDD tooling test suite passed. This was not a native UI run.

The result is partial. Path-based package metadata and other manifest fields,
the compiler, host tools, and runtime dependencies remain unproven. All Git
manifests remain unverified. The result remains `complete: false` and
`reusable: false`; it does not authorize acceptance receipt reuse.
