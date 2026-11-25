# Current native Ribbon Customization

Eighty-five of 126 executions passed on Linux X11 with Xvfb, Openbox,
and Vulkan Lavapipe. They covered 81 of 122 declarations and 1,167
passing steps. Each case started with a writable copy of the published
Normal ribbon profile. The passing cases cover tab retention, staged
commands, keyboard input, target switching, export and import, and
opening Offset Plane from the grouped Plane menu. The
[placement and selection cases](../2026-11-25-ribbon-customization-dialog-placement-3856/README.md)
verified reopening at the moved position and preserving one selected face.
The [source order case](../2026-11-25-ribbon-customization-source-order-3856/README.md)
verified sort order across category changes and reopen.
The [resize cases](../2026-11-25-ribbon-customization-resize-3856/README.md)
verified resizing both lists, retaining staged edits, and restoring the default size.
The [Sketch entry cases](../2026-11-25-ribbon-customization-sketch-entry-3856/README.md)
verified the Create title menu and picking a source command without staging it.
The Tools and Manage cases verified independent pending edits and their
rollback on Cancel. [Two-target runs](../2026-11-25-ribbon-customization-two-targets-3856/README.md)
The grouped Plane menu case verified its icons and dividers, then
dismissed it with Escape. [Plane menu run](../2026-11-25-ribbon-customization-plane-menu-3856/README.md)

The [Button Size case](case-025/cucumber.json) reached the main-panel
command menu. The [command list](case-059/cucumber.json),
[reorder](case-082/cucumber.json), and [remove](case-083/cucumber.json)
cases reached the Available Commands list, which lacks `2-D Drawing`.
The [target selector](case-021/cucumber.json) and
[letter navigation](case-071/cucumber.json) cases also remain open.

One [cancellation case](case-055/cucumber.json) closes Customize and
then expects a Measure panel that the scenario never opens. Another
[cancellation case](case-086/cucumber.json) expects Cancel to dismiss a
clean draft, where the application disables Cancel. The two Shell
cases stopped at the Options panel menu step.
[First Shell case](case-110/cucumber.json) ·
[Second Shell case](case-111/cucumber.json)

The combined report retains the failed and undefined cases. The
[source inventory](scenario-inventory.json), [input hashes](current-inputs.sha256),
and [argument capture](scenario-arguments.json) tie results to source
lines, examples, and table data. The first 75 passes used builds through
`934df02ce` with the local command-list probes and existing local GUI
filesystem, opening, and saving edits. The follow-up receipts identify
their tested builds. Issue #3978 tracks strict build and
runtime provenance.
