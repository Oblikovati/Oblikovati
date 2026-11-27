# Current Home, Switch, and saved tab previews

Ten current scenarios passed all 73 steps with the published Parts.
Canceling Select Window kept PartOne active. Switch restored the saved
Sketch1 edit. Visiting Home and returning by tab or keyboard kept the
same sketch active. The tab preview checks covered an inactive sketch,
Home, saved file paths, and the active Revolve profile sketch.

The tests inspected tab and preview positions and sampled pixels inside
the thumbnail. The active profile's rendered label and saved path were
checked against its actual document name and open file.

The runs used base revision `9a0aa4d2ce0b2d879513b83695d12ba078251ef2`, the fixture bindings
available for each run, and existing local GUI filesystem, opening and
saving edits. The native window used Linux X11 with Xvfb, Openbox and
Vulkan Lavapipe. The shared [input hashes](current-inputs.sha256)
matched the isolated copies. Strict build and runtime provenance
remains open under #3978.
