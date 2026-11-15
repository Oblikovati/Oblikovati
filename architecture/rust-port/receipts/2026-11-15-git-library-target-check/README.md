<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Git vendor library target diagnostic

The check ran against Cargo 1.99.0 vendor output for the lockfile in application
commit `0ea4a3e81cdf462bc7f0110411278f36b2ed493b`. It checked the conventional
library target and selected declarations for 18 Git packages and compared
22,972 tracked Git vendor files. The result is in [observation.json](observation.json).

As an adversarial check, the generated `oblikovati-api` library path was changed
to `src/forged.rs` and its Cargo checksum metadata was updated to match. The
independent check rejected the altered path. The task-owned vendor files were
restored afterward.

The check declines explicit library declarations and other layouts it cannot
independently normalize. Binary, example, test and bench targets, most package
metadata and lints remain unverified. The result stays `complete: false` and
`reusable: false`; issue #3978 remains open. This was not a native UI run.
