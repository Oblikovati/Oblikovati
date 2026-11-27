# Save preview cancellation regression checks

At application commit `687d7dea0a66378a107e0509a11ec80b4ea00fc4`, the
pinned document-saving run passed four scenarios and 30 steps, including both
preview cancellation scenarios. The scenario checkout was clean at
`48987af7b5cbb06515bc6f30385f508382f34064` before and after the run. The
final captures show the restored checked preview option and the original
selected preview source. Input was injected into the native frame loop under
Xvfb and Lavapipe; this is not an OS-level input test.

The native regression `cancel_discards_actual_checkbox_and_radio_changes`
passed unchanged. Two independent temporary mutations then removed one click
at a time. The checkbox mutation failed at `expected preview unchecked after
clearing`. The radio mutation failed at `expected "Active Window on Save"
selected`. Both mutations were removed before the scenario run. The compressed
logs retain the complete outputs. These checks test the sensitivity of the
existing assertions, not the full save feature or the full head suite (#3948).

```sh
CARGO_TARGET_DIR=/tmp/obk-3948-target CARGO_PROFILE_DEV_DEBUG=0 \
CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  cancel_discards_actual_checkbox_and_radio_changes -- --nocapture

OBK_BDD_ROOT=/tmp/obk-3948-scenarios OBK_BDD_REPORTS=/tmp/obk-3948-bdd \
OBK_BDD_FEATURE=document-saving OBK_BDD_SCENARIO='Cancel ' \
CARGO_TARGET_DIR=/tmp/obk-3948-target CARGO_PROFILE_DEV_DEBUG=0 \
CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
