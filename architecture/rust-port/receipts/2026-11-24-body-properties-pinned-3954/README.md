# Pinned Body Properties scenarios

The six scenarios ran one at a time in the native desktop on 2026-11-24.
The runner opened a rectangular solid from the native fixture. The physical
properties scenario verified the initial N/A fields, computed values after
Update, and the same values when the dialog reopened in the same session.
The appearance scenarios checked pending Black selections, the applied Black
appearance, and restoration of As Part. Escape also preserved the selected
body and its viewport highlight.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Update unavailable physical properties for a solid body | 17 | [Report](physical/cucumber.json), [final frame](physical/final.png) |
| Cancel a body appearance selection | 13 | [Report](cancel/cucumber.json), [final frame](cancel/final.png) |
| Apply and restore a body appearance | 13 | [Report](apply/cucumber.json), [final frame](apply/final.png) |
| Dismiss body properties with Escape | 6 | [Report](escape/cucumber.json), [final frame](escape/final.png) |
| Discard a pending appearance with Escape | 10 | [Report](pending-escape/cucumber.json), [final frame](pending-escape/final.png) |
| Discard a pending appearance using the title-bar close control | 10 | [Report](pending-close/cucumber.json), [final frame](pending-close/final.png) |

All 69 steps and each scenario's setup and cleanup passed. Each case directory
also contains JUnit and a scenario argument report.

The clean scenario checkout stayed at `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The SHA-256 of `features/application/body-properties/body-properties.feature`
before and after the runs was
`0b20ae95482d755cafe546d0bae98db647acdaa2ebfb5353e3fcb3bc6af0cf50`.
[Source report](source-revision.json) and [feature input hashes](feature-inputs.sha256).

The native test binary was built from
`e569e530aebfcf9624ba1f941baa3773f25fe935` with existing uncommitted GUI
filesystem, document-opening, and document-saving edits. The lockfile pins
Martensite to `bef25f47548431c8f128f08f66a1f719bcee4982`. The runner used
Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe. Inside that session, the
following command ran once per scenario with a fresh report directory.

```sh
OBK_BDD_ROOT="$pinned_checkout" OBK_BDD_AREA=application \
  OBK_BDD_FEATURE=body-properties OBK_BDD_SCENARIO="$scenario_name" \
  OBK_BDD_REPORTS="$output_dir" LLVM_PROFILE_FILE="$profile_file" \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json LP_NUM_THREADS=3 \
  "$test_binary" application_scenarios_use_the_native_window --nocapture
```

These runs support the implemented ledger state for the pinned feature.
Strict build and runtime provenance remains open under #3978. The live
scenario source still names the Inventor fixture and tracks the native
fixture substitution in its `BLOCKERS.md`.
