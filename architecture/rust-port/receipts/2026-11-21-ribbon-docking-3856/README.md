<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Published Ribbon Docking scenarios

The published feature ran with the application and bindings in `90c0a3d79`.
The run passed 14 of 22 scenarios and 141 steps. Eight scenarios stopped at
undefined steps. The feature and support files matched before and after.
The earlier application commit `261e41cf7` passed three of 22 scenarios at
the same feature hash. The binding inventory fell from 116 to 62 undefined
steps.

The bindings send scripted input to the native window. They exercise rendered
tabs, panel titles, the floating title menu, and compact groups. An X11 mouse
test undocked from an active tab, docked Right from the floating title, and
restored Top from the docked tab. The same Part remained active throughout.

The remaining cases cover floating Modify, scrolling, Annotate, compact groups
on the left, and the three scenarios that name Inventor Part or assembly
files. The scenario folder records the native Part fixture requests. Source
and runtime provenance are unsealed. See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

```sh
OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios \
OBK_BDD_FEATURE=ribbon-docking OBK_BDD_REPORTS=/tmp/obk-docking-results \
VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```
