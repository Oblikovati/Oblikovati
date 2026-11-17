<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Face Properties native diagnostic

The X11 pointer test selected a face on the saved prism fixture, opened its
context menu and Properties, checked the analytic area, and closed the window
with its X. The image records the open panel on a 1440 × 900 Xvfb display with
software Vulkan. It was captured before the final factoring of shared face-selection and
menu code, with no intentional UI changes. It is diagnostic evidence, not a
verified BDD scenario.

Command: `OBK_BDD_REPORTS=/tmp/obk-face-properties-4053-final XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json tools/with-xvfb.sh cargo test --locked -p oblikovati-head --lib native_selected_face_properties_shows_area_and_title_close -- --nocapture --test-threads=1`

The capture SHA-256 is `94d2cb93b3183d7e6a5e993a1e7fb57fa2e8555713b351da6f24044cf23afdb1`.
