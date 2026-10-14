# Native GUI loop: frame integration and acceptance

This implementation consumes the tested kernel v0.2.7 revision
`d40ed71b704efd08310a4c56a1c1c3a64d4e3e9b`.

The native window lazily owns the GUI context and compiler. The frame loop
copies each input into the GUI before it reaches document and camera commands.
Each pointer button retains its press owner through release. Focus loss and unmap
clear ownership. Remapping does not invent keyboard focus. Save, Undo, Redo and
Home use the existing command paths; global command chords and key releases keep
the existing text-focus rules. GUI activity joins the existing owner wake deadline.
The active-frame interval follows the pinned Go head's 1/60-second policy.
Command feedback is copied into the GUI panel so the panel cannot cover it.

The production X11 event source now copies physical keys and XIM UTF-8 commits
under [ADR-0026](../decisions/ADR-0026-owned-native-keyboard-and-text.md).
Layout letters feed GUI shortcuts; physical positions retain navigation identity.
Keyboard navigation reports pending GUI work even when the pointer is outside
the panel. Clipboard and inline IME candidate placement remain open.
Text-provider and locale declines are visible through viewport health and the
GUI. They leave physical input, commands and rendering available. The host
keeps the requested provider and produces no substitute text commits.
Observed native resize between GUI attachment and presentation now requests a
fresh packet through `SurfaceChanged`. Surface revisions also reject a packet
after resize away and back to the same extent. Native tests verify recovery and
visible completed framebuffer content on both drivers. A resize that interrupts
the kernel's draw or capture with an error still requires a typed kernel outcome;
the released text-only error must propagate. These remain open for complete R4
acceptance.

## Verified local checks

The GUI-loop integration gate at app `a71c7ba8` passed for all 13 crates, all
targets and features, on Xvfb/lavapipe.
The check includes format, Clippy, build, tests, doctests, each-crate coverage,
SPDX, Markdown, document links, workflow checks and duplication.
Head coverage is 96.31%, GUI coverage is 99.29%, and viewport coverage is
94.33%. Every crate exceeds 80%. Duplication is 0.76%: 577 of 75,675
scanned lines, 67 pairs.

New tests cover first-click dispatch, pointer ownership across panel edges,
focus and mapping, semantic command capture, copied pointer/modifier input,
monotonic GUI time, idle return, activity deadlines, command routing, deferred
publication and wake merging. The native command test verifies Save content,
Undo/Redo whole-model pixels, wheel capture and Home through the existing camera
path. Its GUI images come from completed framebuffer captures on both drivers.
Changing or clearing a GUI packet invalidates the prior capture. Separate native
tests retain window, event, mapping, resize and swapchain checks.

## Native AMD diagnosis and corrected input order

The original AMD command test compared GPU capture with client `XGetImage`.
The [kernel oracle diagnosis](../../../RustyCADKernel/docs/evidence/native-gui-oracle-2026-10-14/README.md)
proved that the Xwayland client read can remain stale while the compositor
displays the current GPU frame. The accepted [framebuffer decision](../../../RustyCADKernel/docs/adr/0016-framebuffer-images-for-rendering-acceptance.md)
replaces that GUI pixel oracle. The command test now compares completed
framebuffers without an extra redraw to improve a comparison result.

That correction exposed a separate wheel-routing defect. After a GUI button
click and a wheel event over the panel, a wheel over the model sometimes left
the camera unchanged. A reduced live case failed within four AMD runs. The GUI
engine test showed the cause: Dear ImGui's input trickle queue retained the old
panel position when a pointer move and wheel arrived in one GUI frame. Disabling
trickling repaired the wheel but lost a queued rapid-click batch. The native GUI
owner now processes the wheel's pointer position in one logical GUI frame and
the wheel value in the next, then publishes only the final draw packet. It
retains both sets of actions and keeps input trickling enabled.

The reduced wheel case passed 50 of 50 AMD runs after the fix. The full
Save/Undo/Redo/Home command case passed 20 of 20 before cleanup and 10 of 10
after cleanup on AMD. Both focused cases pass on Xvfb/lavapipe. The rapid-click
engine test still passes. This evidence clears the earlier native GUI blocker;
the former physical-key/committed-text gap is now handled under ADR-0026.
Renderer-interrupted resize remains open. No kernel source or dependency
override was introduced.

## Native keyboard acceptance

The keyboard and text-input gate passed for all 13 crates, every target and
feature, on Xvfb/lavapipe. Every crate exceeds 80% line coverage: viewport
94.33%, head 96.18%, GUI 99.30%; the lowest crate is 91.82%. Duplication is
0.76% (584 of 76,634 lines). The gate includes native tests, doctests, Clippy,
Markdown, documentation links, SPDX, workflow checks and the build.

The native keyboard Save check also passes on AMD with the inherited unavailable
IBus configuration. Separate child processes verify unavailable providers and
locales, completed GUI framebuffers, persistent visible health, and command
feedback. Native corpus checks cover composed Unicode, supplementary characters,
active layout groups, mapping changes, repeat, release identity, focus and unmap.
The retained gate log is `target/quality/native-keyboard-2026-10-14-gate.log`.
This is local integration evidence; full R1V/R4 acceptance remains open.
