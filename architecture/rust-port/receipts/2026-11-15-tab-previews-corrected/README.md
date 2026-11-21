<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Corrected tab preview diagnostics

On 2026-11-15, native Linux Xvfb runs used application base
`f4219bb3db715120633647a9c9a94528b8e4898d` with the test changes in this
branch. The source snapshot was `05317bbb2d0809a532b29b7ff6f726a5722b865f`.
The saved inactive-document preview passed all seven steps, including the
full path and part name on separate lines. The assertion accepts the rendered
model-state suffix, while requiring the name to start a separate row in the
same preview draw list.

The Home preview run failed at its fourth step: the preview picture interior
was blank. A repeat with a wider probe starting below the two label rows also
failed; its report is in `home-red-wide/`. The interior-detail check rejects a
uniform image even when its
border has contrast; a focused negative test covers that case. The screenshot
was inspected and shows an empty preview. Issue #3989 tracks the product fix.
The check establishes that there is variation in the picture area; it does not
by itself prove that every thumbnail pixel represents the intended workspace.

The harness sends input through the GUI frame loop. JSON, JUnit, source reports
and captures are retained in `home-red/` and `location/`. Strict head test
Clippy and the negative test passed. The filtered Cucumber command was
`cargo test --locked -p oblikovati-head --lib
application_scenarios_use_the_native_window -- --nocapture`, run under Xvfb
with the Mesa software Vulkan ICD. Build provenance remains open (#3978); these
are diagnostic observations, not verified BDD acceptance.

After the Home capture fix was merged into the capability branch at
`f12341af1b1c0116eebd3903907091c2e3df4664`, the three scenarios whose names
contain `Preview` passed 21 steps, and the saved-location scenario passed seven
steps. Their reports and captures are in `green/` and `location-green/`. The
Home thumbnail now contains visible workspace detail near its upper edge. The
picture probe begins below the rendered name row so it includes that detail
without counting the label or window border. These runs remain diagnostic.
