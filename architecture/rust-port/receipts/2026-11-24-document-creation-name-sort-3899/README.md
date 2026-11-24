# Sort named native Part templates

The current grouped Details scenario passed all 16 steps, including the three
background steps. The runner loaded the fixture project and inspected its
native Metric and English templates. It selected Standard (in), switched to
Metric, reversed the name order twice, and checked the four Part rows and the
retained inch details after each sort. Cancel kept the original Part active.
[Cucumber](cucumber.json) · [JUnit](junit.xml) · [Step arguments](scenario-arguments.json)

A separate native check created an unsaved Part from a named template,
confirmed the saved extrusion was present in the new Part, and verified the
source bytes. [Creation check](create-template-test.log)

The [input hashes](current-inputs.sha256) matched both the shared scenario
sources and the disposable checkout after the run. The test binary was built
from `25989b0a8be95dcda6f633ae83d79a4b0ffd34f0` with the catalog changes and
existing local GUI filesystem, opening, and saving edits. It ran on Linux
with Xvfb, Openbox, and Vulkan Lavapipe. Martensite was pinned to
`bef25f47548431c8f128f08f66a1f719bcee4982`.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=document-creation \
  OBK_BDD_SCENARIO='Reverse the template name order in grouped Details view' \
  OBK_BDD_REPORTS="$output_dir" LP_NUM_THREADS=3 \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

The two current keyboard-navigation scenarios still need native verification.
Strict build and runtime provenance remains open under #3978.
