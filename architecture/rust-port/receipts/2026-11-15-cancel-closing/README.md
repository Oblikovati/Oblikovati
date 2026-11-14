# Cancel closing a modified part

The two pinned close-cancel scenarios passed on 2026-11-15. They check that
cancelling the save-changes prompt or the first Save As keeps the part active
and its finished sketch in the browser. The reviewed captures show that state.

The filter also ran two preview-setting scenarios. They are diagnostic only:
the steps do not establish that the settings changed before cancellation.
See [issue #3948](https://github.com/Oblikovati/Oblikovati/issues/3948).

Application commit: `607e9f8b217fabfcb05ff27e388717255c7045fa`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run.

```sh
OBK_BDD_ROOT=/tmp/obk-save-cancel-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-save-cancel-report-20261115 \
OBK_BDD_FEATURE=document-saving OBK_BDD_SCENARIO='Cancel ' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
