<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Git vendor target-set diagnostic

The check ran against Cargo 1.99.0 vendor output for the lockfile in application
commit `c413f55795302f3179a94381bbbaff8aaafa89ce`. It checked supported
normalized binary, example, test and benchmark targets for 18 Git packages and
compared 22,972 tracked Git vendor files. The result is in
[observation.json](observation.json).

As an adversarial check, the generated `oblikovati-api` example path for `datums`
was changed to `examples/forged.rs`, and its Cargo checksum metadata was
updated to match. The independent target check rejected the changed path. The
task-owned vendor files were restored afterward.

The check derives supported targets from locked declarations, tracked paths and
the locked edition. It declines ambiguous paths and unsupported workspace or
target transformations. Most package metadata, lints and other manifest fields
remain unverified. The result stays `complete: false` and `reusable: false`;
issue #3978 remains open. This was not a native UI run.
