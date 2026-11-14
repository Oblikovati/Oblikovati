# Inspect Save As controls

Four pinned scenarios passed on 2026-11-15, with 24 passing steps. They check
file-type choices, save-prompt frequency choices, Save As commands, and the
first-save preview of a solid. The preview test creates the solid with native
mouse and keyboard input and checks the rendered preview pixels. The captures were reviewed; the prompt
frequency capture shows the part after the list and prompt were dismissed.
The assertions, not that final capture, check the intermediate list choices.

Application commit: `915c1d8deb5a08c87556545c2801fce30e9f13f7`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. Go baseline details for these four rows remain open.

```sh
OBK_BDD_ROOT=/tmp/obk-save-inspect-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-save-inspect-report-20261115 \
OBK_BDD_FEATURE=document-saving OBK_BDD_SCENARIO='Inspect ' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
