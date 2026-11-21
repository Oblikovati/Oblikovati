# Native widget ABI

Issue #3803 tracks the Windows ARM64 abort in the pinned `imgui-sys` 0.12.0
docking build. Native job `127076849273`, run `41975546842`, aborts at
`imgui.cpp:7853` in `SetNextWindowPos`: the condition argument no longer has a
valid single-bit value. This is an ABI boundary failure, not an invalid command
or a solver failure.

## Boundary and cause

The pinned C wrapper passes C++ `ImVec2` classes by value. Rust declares those
arguments as C aggregates. These layouts match in memory, but the Windows ARM64
calling conventions do not agree on their argument registers. Upstream
[imgui-rs issue 790](https://github.com/imgui-rs/imgui-rs/issues/790) records the
same platform defect. The current upstream cimgui wrapper uses C vector
structures and converts them inside C++; updating all pinned ImGui internals
is a separate change.

A minimal cross-compiler probe uses the same two-float class constructors and
the signature `(Vec2, int, Vec2)`. Rust 1.99.0 targeting
`aarch64-pc-windows-msvc` puts the condition in `w0`. Zig 0.15.2's C++ compiler
targeting the same ABI reads that condition from `w1`. The scalar signature
`(float, float, int, float, float)` agrees in both compilers: condition in `w0`,
coordinates in `s0` through `s3`. This establishes the register mismatch;
it does not replace native Windows ARM64 runtime qualification.

## Implementation

`crates/gui/src/widget_abi.rs` owns the private safe widget operations. Its C++
counterpart accepts only scalar values and byte pointers. It constructs vectors
inside C++ and calls the same pinned Dear ImGui implementation. The build reads
the exact header path exported by `imgui-sys`; it does not vendor another copy
or change the context layout. `IMGUI_USE_WCHAR32` matches the dependency build.

Window position and size, cursor position, buttons, selectable rows, and literal
draw-list text use this boundary on every platform. The old vector-by-value
calls are removed from application controls and the input test fixture.
The remaining SDK calls use scalar arguments or output pointers. The binding
must keep this rule when it adds controls that accept vector values.

The GUI thread still owns the context and every frame. Label owners and text
slices remain live through each call. C++ copies glyph vertices and retains no
borrowed text pointer. No native pointer leaves the private adapter. Window
conditions, widget dimensions, disabled behavior, literal `##` labels, and
command dispatch retain their existing meaning.

## Verification

Native widget tests inspect position, size, and condition before `Begin`
consumes the next-window record. They cover every admitted nonzero condition,
fractional coordinates, and `Never`. Further checks read actual cursor position,
button dimensions, selectable behavior, and draw-list text color.

The Linux GUI all-target/all-feature tests, head and GUI Clippy checks, and seven
native GUI loop tests pass. A real OS keyboard/mouse journey with the Go
MCPBridge creates and fully constrains an XY rectangle, checks preview/cancel
and sketch-tool undo/redo, edits size from 10 to 20 mm, saves/reopens, and switches
two same-named Part tabs through their stable identities. Captures confirm the
controls and sketch; the desktop and add-in close normally.

These Linux checks establish preserved controls and rendering. Native Windows
ARM64 CI must pass the same portable widget tests before the original platform
failure is declared resolved. No assertion, test, or platform admission is
disabled by this change. Extrude and the complete cube journey remain open.
