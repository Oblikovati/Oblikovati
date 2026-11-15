<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Full document tabs diagnostic

On 2026-11-15, the full pinned document-tabs feature ran on Linux x86-64 with
Xvfb and Mesa software Vulkan. The application commit was
`3fe4635e3a8067727fa1ec93c6d26f6cfedeb475`, with an unrelated modified root
README. The scenario snapshot was `05317bbb2d0809a532b29b7ff6f726a5722b865f`
and remained clean and unchanged during the run.

The run passed 18 of 38 scenarios and 211 of 231 executed steps. The other
20 scenarios stopped at undefined steps. They are not product failures or
passes; later steps did not execute. The complete report and captures are
retained here. The later focused visibility work binds and tests two of those
cases separately; do not add results from different runs as a new total.

The command ran `cargo test --locked -p oblikovati-head --lib
application_scenarios_use_the_native_window -- --nocapture` with
`OBK_BDD_FEATURE=document-tabs` under `xvfb-run -a -s '-screen 0 1920x1080x24
-noreset'`, using the Mesa software ICD. The harness sends input through the
GUI frame loop. Build provenance is unbounded (#3978), so this diagnostic is
not acceptance evidence. The run returned failure because of the undefined
steps and did not run the full gate.
