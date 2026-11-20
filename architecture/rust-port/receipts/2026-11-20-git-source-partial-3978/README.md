# Partial Git dependency diagnostic

On 20 November 2026, `git_vendor_check.py --allow-partial` checked the locked
Martensite and Oblikovati.API Git sources against a freshly generated offline
vendor tree. It checked 21 packages and 24,473 tracked files. It did not check
Corliss, and all 22 generated Git package manifests remain unverified. The
result reports `complete: false` and `reusable: false`.

The Cargo.lock SHA-256 was
`901be5ce7170640044456a0c1d40e74bca717f52e18a322cf0f774c755076f9c`, matching
the lockfile in the application commit used by the complete docking diagnostic.
This was a separate vendor generation and check. It does not retroactively
validate that earlier run's vendor tree.

The full-source check stopped on Corliss because its package excludes tracked
examples. Cargo's own offline `package --list` matched the freshly generated
vendor files except for Cargo's generated manifest and checksum metadata. The
verifier needs a sound way to account for Cargo package selection before it can
check that source. The package listing is retained here for diagnosis.

The source repositories and toolchain were local and unauthenticated. This is
partial diagnostic evidence, not build provenance or a verified BDD result.
Issue #3978 remains open.
