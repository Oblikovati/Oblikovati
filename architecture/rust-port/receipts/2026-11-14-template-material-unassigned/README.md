# Built-in template material check

The native `selected_builtin_part_shows_its_unassigned_material` regression ran
on Linux with Xvfb and Vulkan Lavapipe. It selected the built-in metric Part and
checked `Material: Unassigned` in its details. The capture was reviewed.

The owner test `builtin_part_recipe_material_matches_the_created_document`
checked that creating the same template leaves the Part material unassigned.
This is a focused regression, not a BDD scenario receipt. The configured
`Generic` template remains unavailable; see #3932.
