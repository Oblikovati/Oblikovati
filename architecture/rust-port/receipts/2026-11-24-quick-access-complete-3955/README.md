# Complete native Quick Access feature run

All 13 current scenario declarations passed. The run expanded the tooltip
outline into 13 examples, for 25 executed cases and 268 steps. The cases
ran with fresh native application sessions. The [combined Cucumber report](cucumber.json)
matches the [parsed inventory](scenario-inventory.json) and
[current input hashes](current-inputs.sha256).

The [tooltip run](tooltip-examples/cucumber.json) checked the 13 default
command icons, their tooltips, and dismissal. Each example kept the
supplied Part active without opening a command popup. The report and
final frames came from base revision `102f310cee5be55f65a187d16445de194224f804`
with the tooltip assertions and existing local GUI filesystem, opening,
and saving edits.

The [Print](../2026-11-24-quick-access-print-icon-3955/README.md),
[Extrude](../2026-11-24-quick-access-extrude-icon-3955/README.md),
[Part and Assembly](../2026-11-24-quick-access-part-context-3955/README.md),
[duplicate Add](../2026-11-24-quick-access-disable-add-3955/README.md), and
[Redo, menu, and Material Browser](../2026-11-24-quick-access-redo-menu-material-3955/README.md)
receipts record the application revisions and native environment for the
remaining cases. The Print scenarios cover the icon and its toolbar setting.
Strict build and runtime provenance remains open under #3978.

Formatting and Clippy passed. The 13 focused toolbar tests and 16 icon
tests also passed.
