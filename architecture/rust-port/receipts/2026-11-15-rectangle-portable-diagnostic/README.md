# Two-point rectangle diagnostic

On application commit `af2e2b010`, the pinned scenario snapshot was
`e76cb749ed43c73a91c24854057da7c890041f51`. The feature SHA-256 was
`c5b51038c64283e9ec2a81e892ea8f912a06264a80eee7e42d65932f3146aa59`.
The source checkout was clean and unchanged during the run.

The four runnable cases, `rectangle-two-point-01`, `-06`, `-07`, and `-10`,
passed in diagnostic mode. The six remaining cases were tagged capture-only
and were not executed. The report records each step and exclusion. The test
uses injected frame-loop input, not OS mouse and keyboard input. It is not a
verified BDD receipt: native build provenance remains open in #3978.

The host used Xvfb, Lavapipe and `XMODIFIERS=@im=none`. The relevant command was:

```sh
OBK_BDD_FEATURE=rectangle-two-point OBK_BDD_DIAGNOSTIC=1 \
OBK_BDD_REPORTS=/absolute/empty/report-directory XMODIFIERS=@im=none \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a -s '-screen 0 1920x1200x24' \
  cargo test --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```

An initial attempt without the input-method setting failed before the sketch
opened and was stopped. Its output is not counted above. See #3886 for the
rectangle work and the scenario folder for fixture restrictions.
