# Native model-state table and member setup

The [initial parameter removal](before/removal/cucumber.json),
[invalid equation](before/verify/cucumber.json),
[arithmetic expression](before/arithmetic/cucumber.json),
[row insertion](before/row/cucumber.json), and
[duplicate name](before/names/cucumber.json) cases stopped at unbound setup
steps. The fixture now edits the native table, creates or renames members
through the browser or table, and keeps the saved blank metric Part active.

| Native scenario | Published | Pinned |
| --- | ---: | ---: |
| Cancel removal of a model-state parameter column | [13 steps](published/removal/cucumber.json) | [9 steps](pinned/removal/cucumber.json) |
| Report an unevaluable model-state parameter equation | [18 steps](published/verify/cucumber.json) | [14 steps](pinned/verify/cucumber.json) |
| Retain an arithmetic expression in a model-state parameter | [14 steps](published/arithmetic/cucumber.json) | [10 steps](pinned/arithmetic/cucumber.json) |
| Cancel inserting a model-state table row | [16 steps](published/row/cucumber.json) | [11 steps](pinned/row/cucumber.json) |
| Decline removal of duplicate model-state member names | [16 steps](published/names/cucumber.json) | [11 steps](pinned/names/cucumber.json) |
| Rename an inactive model state through the table | [12 steps](published/rename/cucumber.json) | [8 steps](pinned/rename/cucumber.json) |
| Reject an empty model-state member name | [16 steps](published/empty/cucumber.json) | [11 steps](pinned/empty/cucumber.json) |
| Duplicate the active model state in an empty part | [17 steps](published/duplicate/cucumber.json) | [16 steps](pinned/duplicate/cucumber.json) |

All 122 published and 90 pinned steps passed. Seven pinned scenarios
gained R0 records. The binding inventory fell
from [30 unresolved steps](../2026-11-24-native-extrusion-distance-4119/published-bindings.json)
to [15](published-bindings.json). The 15 belong to four published scenarios
that require an owned angular parameter. The pinned profile-extrusion
contract has no taper input. See
[Martensite #463](https://github.com/Oblikovati/Martensite/issues/463).

The runs used base `ab15a3e3f`, the local harness changes, and the
existing local file-browser edits. The
[published fixture snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the saved blank Part. Native tests ran on Linux X11 with Xvfb,
Openbox and Vulkan Lavapipe. The previous commit's ten native Extrude
regressions also passed.

See [#4119](https://github.com/Oblikovati/Oblikovati/issues/4119).
