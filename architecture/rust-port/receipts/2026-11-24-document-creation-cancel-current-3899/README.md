# Cancel document creation with a native Part open

The two current scenarios use `fixtures/cad/workspace-layout-native.opd` with
Sketch1 visible. The runner opened the saved Part through File Open before
opening the template chooser. The Cancel button case confirmed that the
previously active Part remained displayed. The Escape case also checked the
Origin row, the visible solid in the viewport, and the document tab count.

Both cases passed individually, covering 12 steps. The
[combined report](current-cucumber.json) preserves the two results. The case
directories contain the original Cucumber and JUnit files, step arguments, and
final frames. The [input hashes](current-inputs.sha256) matched the shared
source and the disposable copy.

The test binary was built from `2f03519af76f42d538f898e7c611a2294535f312` with
the chooser test changes and existing local GUI filesystem, opening, and
saving edits. Martensite was pinned to
`bef25f47548431c8f128f08f66a1f719bcee4982`. The native runs used Linux x86-64,
Xvfb, Openbox, and Vulkan Lavapipe.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=document-creation OBK_BDD_SCENARIO="$scenario_name" \
  OBK_BDD_REPORTS="$output_dir" LLVM_PROFILE_FILE="$profile_file" \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json LP_NUM_THREADS=3 \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

The pinned document-creation feature requires the original capture host.
Strict build and runtime provenance remains open under #3978.
