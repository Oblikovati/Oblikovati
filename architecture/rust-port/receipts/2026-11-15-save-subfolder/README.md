# Save As folder navigation

Two pinned scenarios passed on 2026-11-15, with 15 passing steps. They check
navigation into a subfolder and back, and reopening Save As after cancellation.
The tests use the `src` subfolder of the head crate as a fixture. The reviewed
captures show the initial folder and proposed part name after each path.

Application commit: `82fc6a4400f6e69663ae97898a94dba1c7507f26`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. Go baseline details for these rows remain open.

```sh
OBK_BDD_ROOT=/tmp/obk-save-subfolder-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-save-subfolder-report-20261115 \
OBK_BDD_FEATURE=document-saving OBK_BDD_SCENARIO='subfolder' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
