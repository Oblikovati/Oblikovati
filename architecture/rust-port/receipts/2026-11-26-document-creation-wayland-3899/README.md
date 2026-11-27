# Current Document Creation scenarios on Wayland

All 39 scenarios and 572 steps passed across the nine published Document
Creation feature files. Each scenario ran in a fresh native Wayland window
with its own profile.

| Coverage | Scenarios | Steps | Report |
| --- | ---: | ---: | --- |
| Catalog folders, views, sorting, selection, and creation | 21 | 313 | [Cucumber](catalog-basics/cucumber.json) |
| Named part template navigation and group boundaries | 3 | 52 | [Cucumber](navigation-native/cucumber.json) |
| Native Part, Assembly, Drawing, Sheet Metal, and Presentation creation | 5 | 68 | [Cucumber](basic-native/cucumber.json) |
| Cancel and Escape | 2 | 12 | [Cucumber](cancel/cucumber.json) |
| Inspect Projects from the chooser | 1 | 12 | [Cucumber](project-dialog/cucumber.json) |
| Preserve saved part geometry in a new document | 1 | 10 | [Cucumber](template-content/cucumber.json) |
| Populated drawing layouts, component browsing, and cancellation | 4 | 65 | [Cucumber](populated-layout/cucumber.json) |
| Restore the selected template after restart | 1 | 22 | [Cucumber](template-restart/cucumber.json) |
| Home and End within a template group | 1 | 18 | [Cucumber](group-endpoints/cucumber.json) |

The runs used the shared scenario tree, including the current
`workspace-layout-native.opd` Part, and checked feature and fixture hashes
before and after execution. The feature progress checker matched all scenarios
and step arguments to the published definitions. Each report directory holds
its input hashes and runner argument capture.

The test binary included application commit `95515bee5` and existing local
GUI filesystem edits. The runs used headless Weston with scripted input,
Pixman, and Vulkan Lavapipe. Issue #3978 tracks strict build and runtime
provenance.
