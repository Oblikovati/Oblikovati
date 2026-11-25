# Current native viewport entity selection

The four solid-face scenarios passed all 24 steps against the published
Part. Native pointer input selected the front face, showed its pick cue,
highlight and contextual toolbar, opened both face menus, and cleared
the face and its cues from empty viewport space.

The other four scenarios require desktop presentation and picking of
3D sketch geometry. The last two also require active 3D sketch editing
(#4092). The [CLI opened the native fixture](spatial-cli-open.log), and
the [desktop frame](spatial-native.png) shows it open with an empty
Sketches group and neither spatial entity visible. The Cucumber reports
record which setup steps were undefined for those four cases.

The native build used base revision `b126f7c5e792e96103012979177d654cc4cf07e5`, the
current selection bindings, and existing local GUI filesystem, opening
and saving edits. It ran on Linux X11 with Xvfb, Openbox, and Vulkan
Lavapipe. The [input hashes](current-inputs.sha256) matched the shared
source and isolated copy after the runs. Strict build and runtime
provenance remains open under #3978. See the [combined report](cucumber.json)
and [scenario inventory](scenario-inventory.json).
