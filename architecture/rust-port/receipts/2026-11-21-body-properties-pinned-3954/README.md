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

## Native fixture migration

The [candidate Cucumber report](candidate-cucumber.json) and
[JUnit report](candidate-junit.xml.gz) cover six scenarios and 69 steps.
The run used a local copy of the published scenarios with the six fixture
references changed from `.ipt` to `.opd`. The
[source report](candidate-source-revision.json) confirms that the local
copy stayed unchanged. Every scenario passed its cleanup hook and produced
a final capture.

The `.opd` fixture opened through File Open. The first visit to Body Properties
showed N/A. Update showed 20 g, 4,600 mm², 20,000 mm³, and a center of gravity
of (−20, −12.5, 10) mm. Reopening retained those values. The other scenarios
covered appearance, selection, Escape, and the title-bar close control.

The candidate feature SHA-256 is
`4d2cc1457382465a23a2ec9469a3db9c85aa5474b9412df5068064de0707bba1`.
The native fixture SHA-256 is
`50d6b58b9d2d48e778334b9bce4f1610c0401f88e3e37d304536120b6d099643`.
The published feature still names the Inventor file. Its fixture reference
update is tracked in the scenario folder's `BLOCKERS.md`.
