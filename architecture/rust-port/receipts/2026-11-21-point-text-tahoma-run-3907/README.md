# Published Point Text run with Tahoma

The full native run at application commit `30a723da3` covered 38 Point Text
scenarios. It passed 26 scenarios with cleanup and failed 12. The benchmark
recorded 210 passing steps and 12 failures. The Rust test exited with status
101. The feature, its documentation, the icon, and the external blocker notes
matched the original hashes after the run.

The runner used the published MCAD-Scenarios checkout, an isolated X11 display
under Xvfb, Vulkan Lavapipe, and a private Fontconfig profile with Wine Tahoma.
Arial was unavailable. The 12 failures comprised nine scenarios with undefined
steps, the old retained-defaults assertion in text-08, and the missing Arial
fixture in text-18 and text-30. All nine scenarios with undefined steps remain
open for the source and font requirements recorded in the behavior ledger.

The screenshot shows the saved Sketch2 reopened in the same application
session. The viewport shows both stacked fractions, numbered paragraphs,
`Part6.opd`, and the `10.00` field. The status bar reports 47 dimensions needed.
The Cucumber report records the separate check for the earlier `10.000` field.

```sh
OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios OBK_BDD_FEATURE=text \
  OBK_BDD_REPORTS=<reports-folder> \
  FONTCONFIG_FILE=<private-tahoma-fontconfig> \
  VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  XMODIFIERS=@im=none WINIT_UNIX_BACKEND=x11 \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The feature progress inventory counts the 26 source-matched native passes at
the recorded revision. Strict receipt provenance remains open in
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
