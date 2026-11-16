<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Cube checks on current branch

On application commit `ccc69b11f899c712b4e704ae51c912c3462e81ee`, the
headless `one_size_parameter_drives_certified_cube_replay_and_saved_names`
test and native `native_new_part_button_creates_and_selects_an_unsaved_document`
test passed. Their logs are retained here. The tests ran offline with a fresh
Cargo target; the native test used Xvfb and Vulkan Lavapipe.

The headless test checks the 10 mm and 20 mm sketch prism, topology, volume,
per-face area, and reference keys through replay, undo, redo, and serialization.
The native test drives the GUI frame loop through the cube journey and uses a
typed API fixture for model state. It is not an OS mouse or keyboard test and
does not retain a foreground screenshot. The user's README was modified during
the run. Build provenance is not bounded (#3978), and the solve precision bound
remains open (#3928), despite the test's name. These are diagnostic results,
not verified BDD receipts or proof of the R3 milestone. The full gate was not run.
