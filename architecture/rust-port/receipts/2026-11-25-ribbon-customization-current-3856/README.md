# Current native Ribbon Customization

One hundred and eight of 126 executions passed on Linux X11 with Xvfb, Openbox,
and Vulkan Lavapipe. They covered 104 of 122 declarations and 1,475
passing steps on this retained input snapshot. Each case started with a writable copy of the published
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
Two more cases verified separate Plane choices, regrouping, and native
window resizing. [Plane grouping and reflow](../2026-11-25-ribbon-customization-regroup-3856/README.md)
Three cases verified Shell's Button Size menu, adjacent command reflow,
and retention across tab and appearance changes.
[Shell button size](../2026-11-25-ribbon-customization-button-size-3856/README.md)
A separate case verified adding and removing Shell in Quick Access.
[Shell shortcut](../2026-11-25-ribbon-customization-shell-shortcut-3856/README.md)
The Classic Context Menu case canceled a staged checkbox change and
reopened at the saved value. [Classic menu](../2026-11-25-ribbon-customization-classic-menu-3856/README.md)
The presentation case opened and dismissed both Marking Menu selectors
without editing their saved values.
[Marking Menu presentation](../2026-11-25-ribbon-customization-marking-display-3856/README.md)
The environment case checked seven labels and dismissed the selector
without changing the active profile.
[Marking Menu environments](../2026-11-25-ribbon-customization-environments-3856/README.md)
The 2D Sketch sub-environment case opened the choices, checked the
selected row, and dismissed the selector without changing preferences.
[Sketch sub-environments](../2026-11-25-ribbon-customization-subenvironment-3856/README.md)
The 3D Sketch profile case inspected eight menu locations and the 3D
sub-environment selector without creating a pending edit.
[3D Sketch profile](../2026-11-25-ribbon-customization-sketch3d-profile-3856/README.md)
The filtered sketch-exit list case verified Page Up and Page Down and
left the custom panel empty.
[Source paging](../2026-11-25-ribbon-customization-source-paging-3856/README.md)
The Ctrl+A transfer placed the focused sketch-exit command first and
kept the separator available for reuse.
[Sketch-exit transfer](../2026-11-25-ribbon-customization-exit-transfer-3856/README.md)
The target arrow-key case selected a staged separator and its adjacent
commands without reordering the draft.
[Target navigation](../2026-11-25-ribbon-customization-target-arrows-3856/README.md)
The target inventory case checked the Part and Assembly labels and
closed the clean selector with Escape.
[Target selector inventory](../2026-11-25-ribbon-customization-target-inventory-3856/README.md)
The Assembly Design case kept the Part Tools and Assembly drafts
independent and discarded both on Cancel.
[Assembly Design target](../2026-11-25-ribbon-customization-assembly-design-3856/README.md)

The [keyboard defaults run](../2026-11-25-ribbon-customization-default-aliases-3856/README.md)
verified the editor controls and stopped at the final sketch assertion.
The published Part opens outside sketch edit.
The interface export case read applied settings from six areas and kept
Customize open.
[Interface export](../2026-11-25-ribbon-customization-interface-export-3856/README.md)
The staged drawing entry was inserted after the selected target row and
discarded on Cancel.
[Space after editing size](../2026-11-25-ribbon-customization-staged-size-space-3856/README.md)
verified consecutive transfers and rollback.
[2-D Drawing insertion](../2026-11-25-ribbon-customization-drawing-insertion-3856/README.md)

Case 44 kept both selected rows in place after the Up button.
[Multiple-row ordering](../2026-11-25-ribbon-customization-multiselect-up-3856/README.md)
includes the rerun with the published design references.
Case 60 verified the first two rows stay selected and in place after Down.
[Down ordering](../2026-11-25-ribbon-customization-multiselect-down-3856/README.md)

The [reorder after source selection run](../2026-11-25-ribbon-customization-reorder-after-source-3856/README.md)
verified that Up still moves the selected target row. The
[return after source selection run](../2026-11-25-ribbon-customization-return-after-source-3856/README.md)
verified removing and adding the same staged command after selecting a
different source row.
[Home and End navigation](../2026-11-25-ribbon-customization-target-ends-3856/README.md)
selected the first and last staged rows and preserved their order and options.
[Letter navigation](../2026-11-26-ribbon-z-parallel-3856/README.md) later passed
on both Wayland and X11.

One [cancellation case](case-055/cucumber.json) closes Customize and
then expects a Measure panel that the scenario never opens. Another
[cancellation case](case-086/cucumber.json) expects Cancel to dismiss a
clean draft, where the application disables Cancel. The later Shell Apply run passed, and the Cancel run reached the Martensite
preview tolerance check.
[Current Shell runs](../2026-11-25-ribbon-customization-shell-3856/README.md)

The combined report retains the failed and undefined cases. The
[source inventory](scenario-inventory.json), [input hashes](current-inputs.sha256),
and [argument capture](scenario-arguments.json) tie results to source
lines, examples, and table data. The first 75 passes used builds through
`934df02ce` with the local command-list probes and existing local GUI
filesystem, opening, and saving edits. The follow-up receipts identify
their tested builds. Issue #3978 tracks strict build and
runtime provenance.
