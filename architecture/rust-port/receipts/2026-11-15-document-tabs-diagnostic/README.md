# Document-tabs diagnostic run

The pinned document-tabs feature run failed on 2026-11-15. Cucumber reports
nine scenarios passed and 29 failed at undefined steps; 159 steps passed and
29 were undefined. The unbound steps include keyboard switching, tab menus,
reordering, detached windows, previews and window selection. Steps after the
first undefined step in each scenario did not run. See
[issue #3900](https://github.com/Oblikovati/Oblikovati/issues/3900) and the
keyboard work in [issue #3950](https://github.com/Oblikovati/Oblikovati/issues/3950).

This is diagnostic evidence, not an acceptance receipt. The nine passing cases
need review and a passing run before they can be counted as current verified
scenarios by the ledger. The report, captures and compressed log are retained
for analysis.

Application commit: `0e186f5f5c0117b1325303cb3b1560befc19c861`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. The live feature hash matched the pinned copy after
the run.

```sh
OBK_BDD_ROOT=/tmp/obk-tabs-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-tabs-full-report-20261115 \
OBK_BDD_FEATURE=document-tabs \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
