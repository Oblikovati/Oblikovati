# Current native Document Saving

All 23 dependency-ready scenarios passed 260 steps on the published feature.
The final two scenarios verified the shaded isometric preview of a measured
40 × 25 × 20 mm solid and retained the selected filename after selecting a
subfolder and clearing the file-list highlight.

The twenty-fourth case requires Assembly Simplify with New Part File to
produce two solids, a source-assembly reference, and an assigned unsaved
output path. The shared source marks it `@needs_fixture`. There is no registered
Rust desktop action for this workflow under #3963. The
[case report](case-24/cucumber.json) records the undefined setup step.

The earlier [first saves](../2026-11-25-document-saving-first-save-3888/README.md),
[sketch, preview, and prompt checks](../2026-11-25-document-saving-sketch-preview-and-prompts-3888/README.md),
and [native folder checks](../2026-11-25-document-saving-native-folders-3888/README.md)
keep their individual reports and setup. The folder cases used separate test
processes with writable copies of the published starting folder.

The matching builds used Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe and
included existing local GUI filesystem, opening and saving edits. The published
[input hashes](current-inputs.sha256) matched the isolated copies. See the
[combined report](cucumber.json) and [scenario inventory](scenario-inventory.json).
Issue #3978 tracks strict build and runtime provenance.
