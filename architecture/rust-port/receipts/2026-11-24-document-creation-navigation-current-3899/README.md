# Navigate named project templates

All three current scenarios passed separately in the native desktop. The
52 steps include three background steps in each case. The shared setup
loads the supplied project, confirms the named native catalog, and opens
the Part with Sketch1 visible.

- Name sorting reversed the four Metric rows and kept the selected inch
  template's details.
- Right and Left moved between Sheet Metal (mm) and Standard (DIN). Down
  scrolled without changing the selected template.
- The narrowed chooser retained Standard (mm) at the right boundary and
  advanced its horizontal scroll. Home and Left retained Sheet Metal (DIN)
  and returned the gallery to a zero horizontal offset.

The [combined Cucumber report](cucumber.json), [source inventory](scenario-inventory.json),
and [runner step arguments](scenario-arguments.json) cover the same three
cases. The case directories contain the original Cucumber, JUnit,
step arguments, and final frame. The [input hashes](current-inputs.sha256)
matched the source and the disposable copy before and after the runs.

The binary was built from the application commit identified in the feature
result record with the gallery fix, native test bindings, and existing local
GUI filesystem, opening, and saving edits. The runs used Linux x86-64, Xvfb,
Openbox, and Vulkan Lavapipe. Martensite was pinned to
`bef25f47548431c8f128f08f66a1f719bcee4982`.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=document-creation OBK_BDD_SCENARIO="$scenario_name" \
  OBK_BDD_REPORTS="$output_dir" LP_NUM_THREADS=3 \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

Strict build and runtime provenance remains open under #3978.
