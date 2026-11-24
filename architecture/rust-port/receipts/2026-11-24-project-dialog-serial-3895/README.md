# Project Dialog native scenarios

The current Project Dialog feature starts the application with named native
projects and, for the template chooser, a named Part. The scenario setup uses
the desktop project loader and the typed startup document operation. Mouse
clicks, Ctrl+N, Enter, and the arrow keys go through the X11 window. Each case
starts with disposable application preferences.

All eight current scenarios and 110 steps passed individually. The pinned
feature also passed individually, with eight scenarios and 101 steps.

| Scenario | Current steps | Pinned steps | Current report |
| --- | ---: | ---: | --- |
| Inspect project information from the template chooser | 14 | 13 | [Cucumber](current/inspect-active/cucumber.json) |
| Inspect an inactive project without activating it | 12 | 10 | [Cucumber](current/inspect-inactive/cucumber.json) |
| Inspect and cancels the new single-user project wizard | 19 | 19 | [Cucumber](current/inspect-wizard/cucumber.json) |
| Navigate backward through the new-project wizard | 16 | 14 | [Cucumber](current/wizard-back/cucumber.json) |
| Cancel browsing for a project workspace folder | 14 | 13 | [Cucumber](current/cancel-browsing/cucumber.json) |
| Choose the workspace folder for a new project | 11 | 10 | [Cucumber](current/choose-folder/cucumber.json) |
| Expand and collapse project settings with the keyboard | 12 | 11 | [Cucumber](current/settings-tree/cucumber.json) |
| Expand library groups independently in the project tree | 12 | 11 | [Cucumber](current/library-groups/cucumber.json) |

The [combined current report](current-cucumber.json) preserves the case
results from the individual runs. Each case directory contains the original
Cucumber and JUnit reports, step arguments, and a final frame.

The [current input hashes](current-inputs.sha256) matched the shared source
before and after the run. The runner read a disposable, unversioned copy.
The [pinned input hashes](pinned-inputs.sha256) matched the clean checkout at
`b576445b98cffd686dcc05aa8c017de2a591c8c8`.
[Current source record](current/source-revision.json) · [Pinned source record](pinned/source-revision.json)

Both runs used Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe on 2026-11-24.
The pinned run used the test binary built from
`e569e530aebfcf9624ba1f941baa3773f25fe935`. The current run used
`4f2e9ac69b56ec75528a8a70ff10c458b4d23ed0` and the updated Project Dialog
bindings. Both builds included existing GUI filesystem, opening, and saving
edits. Martensite was pinned to `bef25f47548431c8f128f08f66a1f719bcee4982`. Inside the display session,
the following command ran once per scenario with a fresh report directory.

```sh
OBK_BDD_ROOT="$scenario_source" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=project-dialog OBK_BDD_SCENARIO="$scenario_name" \
  OBK_BDD_REPORTS="$output_dir" LLVM_PROFILE_FILE="$profile_file" \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json LP_NUM_THREADS=3 \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

Strict build and runtime provenance remains open under #3978.
