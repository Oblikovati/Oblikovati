<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Git vendor manifest regeneration observation

On 2026-11-15, Cargo 1.99.0 generated the locked vendor tree twice. The second
generation used a new Cargo home with no existing Git checkouts or extracted
registry sources. It copied the original Cargo Git object stores and registry
archive and index caches, then ran Cargo with an empty environment except for
`PATH`, `HOME`, and `CARGO_HOME`. Both commands used `cargo vendor --locked
--offline` against application commit `4bd62fc0531d89bcf18697077e1d462612c119b2`.

All 18 Git-sourced package manifests were byte-identical. Their individual
SHA-256 values are in `observation.json`. The original working checkout was
used for both runs and contained a modified README; the experiment did not
verify whether that file could affect vendoring.

This checks repeatability, not provenance: the Git object stores were copied
from the same cache, and the toolchain, workspace, host, and generated
manifests were not independently authenticated. Do not use this observation
to authorize receipt reuse. Issue #3978 remains open. A future verifier must
bind regeneration to checked Git trees, workspace inputs and a controlled
Cargo toolchain, then test altered manifests and workspace inheritance.
