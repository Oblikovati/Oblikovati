# Sketch creation API and application preferences — partial R3

Tracking: [#3764](https://github.com/Oblikovati/Oblikovati/issues/3764).
Reference: Go `addin/router/sketch.go`, `addin/router/options.go`, and
`app/options/options.go` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The canonical `sketch.create` request now calls the shared
[checked creation command](implementation-sketch-creation.md). There is no
second mutation path or local copy of the public DTO. An omitted origin plane
means XY; names are trimmed and normalized to XY/XZ/YZ. A supplied work-plane
index takes precedence. Its origin prefix and authored entries use the same
classification and order as the datum catalogue, including construction planes.
The sketch retains the selected feature ID rather than a copied frame.

Orientation references use the existing origin/owned-axis resolver. X/Y choice,
reversal, and optional finite model-space origin flow to the model's one frame
operation. The model checks current availability before adopting geometry.
Malformed and unavailable references have named declines. Kernel edge references
remain unavailable until the reference catalogue is integrated.

Part creation uses the application origin-projection preference. The default is
on. Assembly creation follows the Go policy: it creates no part-origin binding.
Creation, selected solve, analysis, and deletion now accept part and assembly
owners. Drawing and presentation owners remain invalid for these calls.

One injected `SketchPreferenceAccess` supplies the current sketch group across
automation client instances and later creations. `SketchPreferenceFile` loads a
Go-compatible YAML group, merges absent keys over Go defaults, and preserves
other groups as opaque typed YAML values. Empty or absent files use defaults;
read errors and invalid content decline. Unknown keys within the owned sketch
group decline before a write can discard them.

Only the supported sketch group is advertised by `options.listGroups`.
`options.getGroup` and `options.setGroup` use the API's canonical union and view.
Writes require exactly the named payload, replace the exposed fields, and retain
the input/display flags omitted by the view (Go #2016). Grid spacing must be
positive and finite. The response is prepared before persistence, and memory
adopts options only after a complete atomic save. Preference edits are outside
document history and do not dirty the active document.

`FileHost::with_sketch_preferences` injects that owner. Preference API calls
without an owner decline, so writes cannot disappear in a temporary host.
Creation without an injected owner uses the standard default, as existing bare
host callers did before option integration. The native application must load
and inject its one preference owner into UI and automation hosts.

Tests cover normalized and indexed planes, construction hosts, current owned and
origin axes, malformed and unavailable sources, assembly commands, typed creation,
grouping and abort, vetoes, undo/redo, native save/reopen, preference reads/writes
across client instances, default-on and disabled projection, hidden flags, unknown
groups, legacy absent keys, and every atomic save failure phase. The storage tests
share one named fake file implementation with the atomic-write tests.

Other application option groups, native preference path selection, live grid/input
updates in the head, interactive sketch environment, face/edge reference catalogue,
and native rendering/kernel acceptance remain required. This step completes the
owned API and preference boundaries for the implemented sketch group.
