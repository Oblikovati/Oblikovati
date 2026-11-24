# Inspect the project from the root template chooser

All 12 steps passed in the native desktop. The chooser listed Standard (mm)
from the project template root and displayed its assembly kind and millimeter
units. The Projects dialog showed the complete active project name wrapped
within the Name column, its location, and its settings tree. Done returned
to the same selected assembly template and details.

The [Cucumber report](current-cucumber.json), [JUnit report](current/wrapped/junit.xml),
and [final frame](current/wrapped/final.png) record the run. The
[input hashes](current-inputs.sha256) matched the shared scenario source and
disposable copy after the run.

The test binary was built from the application commit in the feature result
record with the root catalog and Projects changes, native bindings, and
existing local GUI filesystem, opening, and saving edits. The run used Linux
x86-64, Xvfb, Openbox, and Vulkan Lavapipe. Martensite was pinned to
`bef25f47548431c8f128f08f66a1f719bcee4982`.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=document-creation \
  OBK_BDD_SCENARIO='Inspect projects from the template chooser' \
  OBK_BDD_REPORTS="$output_dir" LP_NUM_THREADS=3 \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

Strict build and runtime provenance remains open under #3978.
