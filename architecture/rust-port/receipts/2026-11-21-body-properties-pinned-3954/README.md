# Pinned Body Properties baseline

All six scenarios and 69 steps passed on application revision
`4edca8774cb5af70954113ffb1ef9d1fafff4232` under Xvfb, Openbox, and Vulkan
Lavapipe. The [Cucumber report](cucumber.json) matches every step in the
pinned feature. Each scenario passed its cleanup hook and produced a final
capture. The [JUnit report](junit.xml.gz) and [source report](source-revision.json)
confirm the case count and the clean pinned checkout.

The [analytic test](analytic.log) checked the native prism's physical values.
The [Go baseline](go-reference.log) checked part-level commands and appearance
assignment. The Go tests cover separate behavior from the native dialog.

All 13 tests in the [focused run](targeted.log.gz) passed. The native mouse and
Escape test [opened Body Properties](body-properties-native-open-3954.png)
from the body row and confirmed that Escape preserves selection and highlight.
The suite also checked that an appearance edit invalidates calculated values
and that the dialog shows N/A when it opens again.

The ledger now binds the R0 baseline to the pinned feature hash. Strict build
and runtime provenance is tracked in
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
