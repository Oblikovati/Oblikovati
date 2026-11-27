<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Clarified ribbon appearance diagnostic

On 2026-11-15, the full pinned ribbon appearance feature ran on Linux x86-64
with Xvfb and Mesa software Vulkan. The native harness sends input through the
GUI frame loop. It passed 15 of 18 scenarios and 166 of 169 executed steps.
Three scenarios failed in their setup because Show Panels has no Team Web
entry. Tutorials is also absent. The scenario does not establish a Team Web
destination or panel contents. The failures remain open in #3957.

The application base was `81da23e65` with the following test files changed:

- `submenu.rs`: SHA-256 `86b2a308e2b645c3cbf015f75387e9932f686d824ffc905316622a818f242374`
- `reset.rs`: SHA-256 `1fb37ae93cea6bebd077f49146105363bd945211acaa633b860ef0442185fe9b`

The snapshot was `05317bbb2d0809a532b29b7ff6f726a5722b865f`, verified before
and after the run. The feature SHA-256 was
`7ebd05f3a8b347bd7eee5a132e7782ba9731a16f1bb0e8ac26a756fe5da312fa`.
The dedicated target used the revisions in `Cargo.lock`; the full build
provenance is not established (#3978), so these are diagnostic results.

The fixture helper also passed the native
`tools_panel_fixture_changes_and_restores_visibility` test. It hid and restored
the existing Find panel through the Show Panels menu.

```sh
OBK_BDD_ROOT=/tmp/obk-ribbon-fixture-3957-scenarios \
OBK_BDD_REPORTS=/tmp/obk-ribbon-fixture-3957-report \
OBK_BDD_FEATURE=ribbon-appearance \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json XMODIFIERS=@im=none \
xvfb-run -a -s '-screen 0 1920x1080x24 -noreset' \
cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```

Both tests used a dedicated Cargo target with debug information and incremental
compilation disabled. The full run intentionally returned failure for the three
missing setup controls. Its JSON, JUnit, source report and captures are retained
here.

`cargo fmt --all -- --check` passed. Strict head test Clippy stopped at an
existing `cloned_ref_to_slice_refs` warning in
`quick-access-toolbar/clear_face.rs:153`. The same check passed with that one
lint suppressed; it does not replace the full lint gate.
