# Viewport Navigation Wheel with the named native part

The current scenarios explicitly open `fixtures/cad/main-window-frame-start.opd`.
The native runner opens one disposable copy through the X11 file dialog. The
view and wheel setup steps then use the same active Part. The published and
pinned fixture conventions are described in the
[baseline](../2026-11-21-viewport-navigation-wheel-r0-3881/README.md).

All 12 current scenarios and 155 steps passed in separate native runs with
fresh application preferences. The cases exercised wheel layouts, menu
dismissal, size and opacity settings, tooltips, Fit to Window, Go Home, and
repeating the last wheel command.

The [combined report](current-cucumber.json) preserves the results from the
individual runs. Each case directory holds its Cucumber and JUnit reports,
step arguments, and final frame. The [current input hashes](current-inputs.sha256)
record the feature, notes, diagrams, and native Part used by the runner.

The runner read a disposable copy of the unversioned scenario source.
The test binary was built from `ade2ea431c81c1e8f2750ee8fbd4f9daf7bdc603` with the new fixture
binding and existing local GUI filesystem, opening, and saving edits.
Martensite was pinned to `bef25f47548431c8f128f08f66a1f719bcee4982`. The
native runs used Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=viewport-navigation-wheel OBK_BDD_SCENARIO="$scenario_name" \
  OBK_BDD_REPORTS="$output_dir" LLVM_PROFILE_FILE="$profile_file" \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json LP_NUM_THREADS=3 \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

Strict build and runtime provenance remains open under #3978.
