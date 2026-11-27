# Native Classic Context Menu preference

The published case passed thirteen steps. Checking Use Classic
Context Menu enabled OK, Cancel, and Apply while the live preference
stayed unchanged. Cancel closed Customize. The Marking Menu tab
still showed the unchecked value and no pending edits on reopening.
[Native Cucumber report](case-120/cucumber.json)

The persistence tests verified that the classic choice and menu
assignments share one atomic file update. A failed write preserved
both former values. The preferences still reject external file changes.
The native Apply regression reopened the same profile and read the
saved value at Home and in a new Part. The existing marking-menu
slot Apply regression also passed.

The Linux X11 builds used `0e4f16ce3`, this commit's changes, and
the existing local GUI filesystem, opening, and saving edits.
The runs used Xvfb, Openbox, and Vulkan Lavapipe.
[Input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)
