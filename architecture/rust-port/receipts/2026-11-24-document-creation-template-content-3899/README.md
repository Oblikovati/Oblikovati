# Create a Part from a geometry-bearing template

All 10 steps passed in the native desktop. The runner loaded the supplied
project and selected Geometry Template in its Metric catalog. Create opened
one unsaved Part. Its viewport displayed the six-face solid, and its model
browser showed Extrusion1 and its consumed Sketch1. The template file matched
its original bytes after creation.

The [Cucumber report](current-cucumber.json), [JUnit report](current/checked/junit.xml),
and [final frame](current/checked/final.png) record the run. The
[input hashes](current-inputs.sha256) matched the shared source and the
disposable copy.

The binary was built from `4e956594fb910cb2c8c25e117d36ddfcdca080cc` with the
native scenario bindings and existing local GUI filesystem, opening, and
saving edits. The run used Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe.
Martensite was pinned to `bef25f47548431c8f128f08f66a1f719bcee4982`.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=document-creation \
  OBK_BDD_SCENARIO="Preserve the selected part template's geometry in the new document" \
  OBK_BDD_REPORTS="$output_dir" LP_NUM_THREADS=3 \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

Strict build and runtime provenance remains open under #3978.
