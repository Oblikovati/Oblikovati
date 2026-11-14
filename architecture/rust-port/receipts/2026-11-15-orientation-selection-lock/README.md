# Toggle orientation lock for a selected face

The pinned scenario passed on 2026-11-15, with 15 passing steps. It checks the
menu availability, check mark and indicator when a face is selected and the
selection is cleared. The reviewed capture shows the final menu state. The
scenario does not test the pivot; separate native regression tests cover that.

Application commit: `190e360edce00557ad4bb159177f8f012ccb1359`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run.

```sh
OBK_BDD_ROOT=/tmp/obk-save-navigation-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-orientation-integrated-report-20261115 \
OBK_BDD_FEATURE=viewport-orientation \
OBK_BDD_SCENARIO='Toggle orientation lock for a selected face' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```

Go reference commit `aa8297d1b0ac33c024f19e54821536d76e8a447a` passed
`go test ./app -run '^(TestSessionViewPreferences|TestViewCubeTogglesPersist)$' -count=1`.
Those tests check preference state, not the face-selection menu or pivot.

On the same application commit, five native regression tests passed for named
body and face pivots, selection changes and errors, orthographic and perspective
fit, visible sketch content, and document save plus application reopen:

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  selection_pivot_regression -- --test-threads=1
```
