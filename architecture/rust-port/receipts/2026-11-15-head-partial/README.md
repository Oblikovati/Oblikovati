# Partial head library run

On application commit `687d7dea0a66378a107e0509a11ec80b4ea00fc4`, this
serial Linux Xvfb and Lavapipe run was interrupted to integrate the completed
multi-body change. It recorded 149 passing tests and two failures before the
interrupt; 984 tests were selected, with the two BDD runners filtered out.
The run did not finish and is not a head-suite or gate result.

The reported failures were `native_owned_imgui_packet_composes_with_the_document_after_context_destruction`
and `application_step_bindings_cover_selected_source`. The test harness had not
yet emitted their captured panic output when the run stopped. Reproduce each
case independently before assigning a cause. The native display closed on exit.

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib -- \
  --test-threads=1 --skip application_scenarios_use_the_native_window \
  --skip sketch_scenarios_use_the_native_window
```

The binding inventory was reproduced independently on `7c2de1494`: it found
13 undefined steps in two populated drawing-layout scenarios. The default
source was an older bundled copy, not the pinned snapshot. The report and log
are retained here; source selection is tracked in #3972, and the missing
behavior remains in #3899. This result does not count as pinned BDD evidence.
