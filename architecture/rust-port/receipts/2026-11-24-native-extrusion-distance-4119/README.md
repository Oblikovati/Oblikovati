# Native extrusion distance in the model-state table

The [earlier native probe](../2026-11-24-browser-property-fixture-3896/before/native-extrusion-parameter/cucumber.json)
committed a one-inch extrusion and could not find its distance in the
available table parameters. Inline Extrude expressions now get a
built-in Model parameter such as `d0`. The feature stores the parameter
identity and depends on it during replay.

| Native scenario | Published | Pinned |
| --- | ---: | ---: |
| Cancel a staged model-state parameter column and value | [18 steps](published/cancel/cucumber.json) | [15 steps](pinned/cancel/cucumber.json) |
| Accept distinct parameter values for model-state members | [21 steps](published/accept/cucumber.json) | [18 steps](pinned/accept/cucumber.json) |

The published cases draw an undimensioned rectangle on the saved blank
metric Part, extrude it one inch, and create both model states in the
browser. The accepted-edit case measures the solid centroid at 25.4 mm
in the two-inch Copy State and at 12.7 mm after reactivating the one-inch
Primary. The pinned cases start with their existing authored distance.
Both gained R0 records.

The 25 extrusion-owner tests and two deletion-checkpoint tests passed.
The literal-distance lifecycle test checks the built-in parameter and its
owner on creation, preserves its ID across a distance edit, and restores
that ID when undoing deletion. The six parameter conversion tests and the
automation distance validation test also passed.

The published binding inventory dropped from
[43 unresolved steps](../2026-11-24-browser-suppression-fixture-3896/published-bindings.json)
to [30](published-bindings.json). Other parameter-table cases still
require their own setup and native verification, including the angular
parameter cases.

The runs used base `c701ef76f` with local extrusion and harness changes
and existing file-browser edits on Linux X11, Xvfb, Openbox, and Vulkan
Lavapipe. The
[published snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the blank metric Part.

See [#4119](https://github.com/Oblikovati/Oblikovati/issues/4119).
