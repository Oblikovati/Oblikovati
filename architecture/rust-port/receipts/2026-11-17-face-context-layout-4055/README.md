<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Face context layout native diagnostic

The X11 test selected a face, opened and dismissed its context menu, reopened
it, and started New Sketch. It checked the linear region above the ring, the
ring's directional positions, the pointer's position relative to the ring,
and the selection toolbar's visibility. The frame was captured on a 1440 × 900
Xvfb display with software Vulkan. It is diagnostic evidence, not a verified
BDD result.

Command: `OBK_BDD_REPORTS=/tmp/obk-face-layout-4055-final5 XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json tools/with-xvfb.sh cargo test --locked -p oblikovati-head --lib native_part_menu_starts_sketch_on_selected_face_and_reopens_it -- --nocapture --test-threads=1`

The capture SHA-256 is `48878d926ec894d64206a28d6b4855c34c1a52549358c140802c5884197c9064`.
