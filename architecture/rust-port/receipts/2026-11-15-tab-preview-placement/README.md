<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Tab preview placement

On 2026-11-15, native tests ran on Linux x86-64 under Xvfb with Mesa software
Vulkan. The input was sent through the GUI frame loop. With the ribbon
minimized and font scale at 2, the preview opened below the tab strip, stayed
inside the viewport, and disappeared when the pointer moved away. The
attached capture was inspected: the tab remains visible and the thumbnail
does not contain a recursive copy of the preview. A second native test
confirmed that a full ribbon retains the normal above-tab placement.

The first native attempt found a one-logical-pixel overlap with the tab strip.
The final run passed after increasing the clearance. GUI tests cover viewport
edges, partially clipped tabs, constrained height, wrapped labels, and the
capture guard. All 523 GUI library tests passed before a final split of the
placement calculation; all 32 document-tab GUI tests and both native tests
passed again after that split. Strict Clippy passed for GUI and head test
targets, and `cargo fmt --all -- --check` passed.

The preview is intentionally omitted when neither side can fit its wrapped
labels and one logical pixel for each thumbnail source cell; it does not cover
the tab in that case. This
is a native diagnostic, not OS-level input or BDD acceptance evidence. The
full gate remains open and build provenance is not bounded (#3978).

The application base was `6c0822e46e37e171f07965cf23a57bb5c11b802b` with
the changed `previews.rs` SHA-256
`ceed3b97f9140a4841ed84d58d6d80ab00d3744fa582bcec4cafc6b3e6426db6` and
`preview_placement.rs` SHA-256
`c1fad8d9d4839f75345a717f8c2dce1e1641822f30d5d3a20106523d176535f0`.
