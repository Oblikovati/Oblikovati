# Built-in template material check

The native `selected_builtin_part_shows_its_unassigned_material` regression ran
on Linux with Xvfb and Vulkan Lavapipe. It selected the built-in metric Part and
checked `Material: Unassigned` in its details. The capture was reviewed.

The owner test `builtin_part_recipe_material_matches_the_created_document`
checked that creating the same template leaves the Part material unassigned.
This is a focused regression, not a BDD scenario receipt. The configured
`Generic` template remains unavailable; see #3932.

The native `material_source_failure_is_visible_only_in_the_chooser` regression
injected the state produced by a failed catalog and a failed size inspection.
It checked that the regular window remains usable with the chooser closed and
that the chooser shows both named causes. The failure capture was reviewed.
The separate owner regression builds an assigned recipe and uses a failing
catalog source to check that the state includes the material identity and cause.
