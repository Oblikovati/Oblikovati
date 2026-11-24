# Initial native template catalog cases

The first three current native catalog scenarios passed individually, with
42 steps in total. Each run loaded the supplied Metric and English catalog
and confirmed there were no native template files at its root.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Inspect metric and inch part templates | 14 | [Cucumber](inspect-confirmed/cucumber.json) |
| Collapse and expand a template group | 15 | [Cucumber](group-verified/cucumber.json) |
| Change template folders before selecting a new template | 13 | [Cucumber](folders-verified/cucumber.json) |

The details pane showed each selected Part's stored units. Collapsing
the Part group hid its tiles and kept the selected sheet-metal details.
Switching to English replaced the gallery contents and kept the Metric
selection until the user selected the English sheet-metal template.

Each case directory includes its original JUnit report, step arguments,
source record, and final frame. The [input hashes](current-inputs.sha256)
matched the shared scenario source and the disposable copy after the runs.

The test binary was built from `c4597f77f2140749ebf03e41ebf036166da28640`
with the catalog test bindings and existing local GUI filesystem, opening,
and saving edits. The runs used Linux x86-64, Xvfb, Openbox, and Vulkan
Lavapipe. Martensite was pinned to `bef25f47548431c8f128f08f66a1f719bcee4982`.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=document-creation OBK_BDD_SCENARIO="$scenario_name" \
  OBK_BDD_REPORTS="$output_dir" LP_NUM_THREADS=3 \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

Eighteen catalog scenarios remain. Strict build and runtime provenance
remains open under #3978.
