<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Viewport face selection and clearing

The published feature ran on the code committed as `5128782e7`. Two of seven
scenarios passed, with 27 passed steps and five failed steps. X11 pointer input
selected the saved Part's exact front face and verified its teal fill, green
outline, pick cue, and adjacent toolbar. Clicking clear viewport space restored
neutral rendering and removed both contextual controls.

The earlier run detected the toolbar at the face's outline corner. A separate
X11 regression confirmed that the new toolbar stays beside the picked point
after the label disappears. A GUI unit test checked its placement at the top
and right display edges.

The face context menu scenario reached both menus and stopped at the missing
`Select Other...` entry ([#4055](https://github.com/Oblikovati/Oblikovati/issues/4055)).
The other four failed scenarios require deferred 3D-sketch support. The feature,
notes and native Part hashes matched before and after. This local run has
unsealed source and runtime provenance.

```sh
OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios \
OBK_BDD_FEATURE=viewport-entity-selection OBK_BDD_REPORTS=/tmp/obk-face-results \
VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```
