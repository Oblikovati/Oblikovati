# Create basic documents from native templates

All five current scenarios passed individually in the desktop, covering
68 steps. Each case loaded the supplied project and its Metric catalog
before opening the existing Part.

| Template | Steps | Observed result |
| --- | ---: | --- |
| Standard (mm) | 12 | Empty Part with a new tab, Origin, and End of Part |
| Standard Assembly | 14 | Empty assembly viewport, browser groups, and component commands |
| Standard Drawing | 14 | Sheet with border and title block, drawing browser, and view commands |
| Sheet Metal (mm) | 14 | Empty folded model and the sheet-metal ribbon |
| Standard Presentation | 14 | The assembly Insert dialog opened. Cancel left the presentation and its panels visible. |

The [combined Cucumber report](cucumber.json), [source inventory](scenario-inventory.json),
and [step arguments](scenario-arguments.json) match the five cases. Each
case directory contains its original reports and final frame. The
[input hashes](current-inputs.sha256) matched both the shared source and the
disposable copy after the runs.

The binary was built from the application commit in the feature result record,
with the native test bindings and existing local GUI filesystem, opening,
and saving edits. The runs used Linux x86-64, Xvfb, Openbox, and Vulkan
Lavapipe. Martensite was pinned to `bef25f47548431c8f128f08f66a1f719bcee4982`.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=document-creation OBK_BDD_SCENARIO="$scenario_name" \
  OBK_BDD_REPORTS="$output_dir" LP_NUM_THREADS=3 \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

Strict build and runtime provenance remains open under #3978.
