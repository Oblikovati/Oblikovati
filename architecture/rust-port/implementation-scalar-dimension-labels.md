# Scalar dimension annotation boundary

Date: 2026-11-06. Tracking:
[#3839](https://github.com/Oblikovati/Oblikovati/issues/3839).

## Failure evidence

Combined candidate `787e2a670d87ce5d77d372541622897000bd7e08`, hosted run
42162243927, Windows ARM64 job 127713537193, exits the `ribbon_click` binary
with `0xc0000005 / STATUS_ACCESS_VIOLATION`. The last printed test checks
dimension-label theme colours and physical projection at both display scales.
The annotation and scalar-widget files are unchanged between that candidate
and the later native relation-tool stack before this repair.

The pinned imgui-rs 0.12.0 `DrawListMut::add_text` forwards a by-value `ImVec2`
to `ImDrawList_AddText_Vec2`. A cross-compiler probe of that argument signature
uses Rust 1.99.0 and Zig 0.15.2 targeting `aarch64-pc-windows-msvc`. Rust passes
`text_end` in `x3`, while C++ reads it from `x4`. Its scalar control agrees on
`x3`. Thus the text range is not preserved across the vector boundary. This
is the same class of failure recorded in
[the widget ABI implementation](implementation-gui-widget-abi.md). The probe
establishes argument disagreement; it does not replace native Windows testing.

## Repair and ownership

Background annotation and current-window literal text share one scalar C++
entry point. Rust passes bytes, length, coordinates, colour and a layer flag.
C++ selects the draw list, constructs `ImVec2` locally and derives the text end
from the byte length. The direct Rust vector draw-list call is deleted. No
private native pointer escapes the adapter, and no second renderer is added.

The live Ui borrow retains the context through each synchronous call. ImGui
copies glyph vertices before the borrowed label bytes expire. Background text
stays behind chrome; current-window text keeps its existing layer. Label
centering, semantic theme colours and physical display scaling are unchanged.

## Verification boundary

The original theme/scale test remains the native crash regression. A new native
widget test checks literal glyph count, colour and background-before-window draw
order. Local widget tests and the original label test pass. The final full local gate
passes all 13 crates, including 365 head tests, with line coverage from 91.96%
to 99.60% and duplicated lines at 1.00%. Windows x64 GNU and macOS ARM64
host/GUI Clippy pass; these are compile checks, not native runtime claims.

Native X11 mouse/keyboard acceptance passes preview, cancellation, invalid
expression correction, a 10 mm dimension bound to `size`, edits to 20 mm,
undo/redo, save and fresh-process reopen. Reopen retains the expression, value,
text anchor and healthy DOF three. MCPBridge reads and inspected captures check
preview, redo and reopened labels. Both acceptance desktops close normally.

Native Windows ARM64 qualification passes at exact head
`f55f9710c00032b88cb737c3a44087d2f67a7eab`, hosted run 42167873444,
job 127731793203. The original theme/scale label test and the new scalar
background-layer regression both pass. The gate and Linux x64, macOS Intel/ARM64
and Windows x64 native jobs also pass. This verifies the ABI repair; integration
in PR #3840 remains blocked by the separate Linux ARM64 runtime failure.
No assertion or platform test is disabled.

The Linux ARM64 Vulkan runtime failure in Martensite#388 remains separate. The
complete Extrude, cube and named-face journey remains unfinished.
