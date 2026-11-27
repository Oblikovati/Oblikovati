# Two-point rectangle diagnostic run

The pinned feature run failed on 2026-11-15. Cucumber reports nine scenarios
passed and one failed; 71 steps passed and one failed. Scenario 10 expected a
transient count of 24 dimensions needed after a zero first size; the native
status displayed 22. The recorded fixture is not supplied, and the test uses a
constructed fixture. The premise and behavior are under investigation in
[issue #3953](https://github.com/Oblikovati/Oblikovati/issues/3953).

This is diagnostic evidence, not an acceptance receipt. The nine passing cases
need review and a passing run before they can be counted as current verified
scenarios by the ledger. The report, captures and compressed log are retained
for analysis.

Application commit: `f104b9fd6c109b47e3a3507dac3cb2cc09efa7f1`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. The live feature hash matched the pinned copy after
the run. A first run inherited an unavailable IBus input method and failed six
Ready-status assertions; this run uses the Xvfb local input method.

```sh
OBK_BDD_ROOT=/tmp/obk-rectangle-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-rectangle-xim-report-20261115 \
OBK_BDD_FEATURE=rectangle-two-point XMODIFIERS=@im=none \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```
