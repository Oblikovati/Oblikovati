<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Git vendor manifest declaration diagnostic

The check ran against Cargo 1.99.0 vendor output for the lockfile in application
commit `421b06a7418ce7abfb3d10a3b7e1ed846a4cd723`. It compared 22,972 tracked
Git vendor files and selected manifest declarations for all 18 locked Git
packages. The observation is in [observation.json](observation.json).

As an adversarial check, the generated `oblikovati-api` manifest was changed
from `build = false` to `build = "forged.rs"`, and its Cargo checksum metadata
was updated to match the altered manifest. The independent declaration check
rejected the build script. The task-owned vendor files were restored afterward.

The check compares dependency, target and feature declarations and refuses
unsupported or added build scripts. It does not independently verify normalized
library and other target fields, package metadata, lints, or the rest of each
generated manifest. It declines workspace-inherited dependencies and source
build scripts that it cannot compare. The observation remains `complete: false`
and `reusable: false`; issue #3978 remains open. This was not a native UI run.
