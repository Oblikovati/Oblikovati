<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Combined dependency build probe

On 2026-11-15 the Linux x86-64 sandbox built the locked native head test runner
from committed source `5afb2d5402eb5c1582bbfd11535d4215caa50cd6`. Before
building, the probe compared 280 registry packages to their locked archives and
22,972 Git-sourced files in 18 packages to their locked Git objects. It checked
the mounted inputs again after the build. The Cargo log, JSON messages, source
replacement configuration and observation are retained here.

The source was exported with `committed_build_source.py`; the vendor tree was
generated with `cargo vendor --locked --offline`. The probe used Rust 1.99.0,
the local crates.io archive cache, the two local Cargo Git object stores, and
`/usr/bin/bwrap` copied to the temporary task directory. The source, vendor,
toolchain and configuration mounts were read-only; the network namespace was
isolated. The runner was compiled but no scenario was executed.

This is a diagnostic observation, not acceptance evidence. All 18 generated
Git manifests remain unverified. Host libraries, kernel, devices and transient
host writes are not bounded. The result says `reusable: false`; #3978 remains
open. The temporary paths in the observation identify the run and have been
removed after retaining these artifacts.
