# GUI frame continuation and focus input

Tracking: [native GUI #3630](https://github.com/Oblikovati/Oblikovati/issues/3630)
and [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This extends the owned frame producer used by
[native composition](implementation-native-gui-composition.md).

A frame now reports `GuiFrameActivity`: deferred input, pending window layout,
and active interaction. `needs_frame` combines these reasons without choosing
a clock or sleep duration. The native loop must combine this signal with its
mapped/focused state, document work, input capture and timing policy. This step
does not yet activate automatic GUI frames in the executable.

Dear ImGui 1.89.2 can defer press/release events across multiple frames. The
previous output exposed capture flags and actions but did not expose its remaining
queue. A host that waited only for another native event could leave clicks pending.
The producer now reads the actual queue after Render. Its fast two-click regression
drains according to that signal and verifies two Save actions, replacing the fixed
three-extra-frame assumption. Moving the pointer out of the controls returns the
producer to idle.

Layout continuation reads active windows' autofit and cannot-skip-layout counters.
An ordinary hidden/clipped window alone does not keep the loop active. A real
content-sized window test follows activity until its size settles. The native
head composition test also waits for reported work instead of always drawing two
warmup frames. It still checks the actual X11 pixels, model preservation, retained
font lifetime and close on the AMD display.

Interaction covers active or hovered items, requested text input, held pointer
buttons, window movement and the wheel-release timer. It reports GUI work, not
native window focus or visibility. The host still owns those independent states.
No constant redraw interval, global timer or second event queue is introduced.

`GuiInputEvent::Focus` enters the same native input queue as the other events.
A real text-widget test holds Control and a pointer button, then loses focus.
The native frame clears the held keys, buttons and modifiers. Focus gain does not
restore them. The binding's `AppFocusLost` flag is transient: it is visible during
the frame and reset by EndFrame. It is not used as stored native focus state.

The pinned imgui-rs 0.12.0 wrapper omits the deferred queue and focus-event method.
The existing private FFI boundary now reads the owning context and window array,
and invokes the native focus method through the binding's `RawCast` contract.
No native pointer escapes. The remaining GUI modules still deny unsafe code.
Tests cover absent context, invalid window records, each layout and interaction
source, queued real clicks, idle return and focus transitions. Input validation
still occurs before the first event is queued.

Cross-platform and coherence review: this adds no platform path, geometry algorithm,
or production timing constant. The exact binding stays pinned. The native host
must still copy physical keys and text, preserve Go command-chord rules, route
pointer ownership, schedule continued frames, and dispatch GUI actions through
the existing document commands. Full R1V/R4 acceptance remains open.

## Local acceptance

The complete `make gate` passed on the final source for all 13 application crates,
all targets and features. It includes formatting, Clippy, build, tests, doctests,
native display, coverage, Markdown, documentation links, SPDX, workflow checks
and duplication. The full suite used Xvfb with software Vulkan. The focused
native panel check also passed on AMD with the actual Xwayland display.

Every crate exceeded 80% line coverage. GUI reached 99.01%, viewport 94.33%
and head 96.20%. Duplication remains 0.77% (577 of 74,910 scanned lines,
67 pairs), with no new clone pair. Retained logs are
`/tmp/gui-frame-continuation-gate.log` and
`/tmp/gui-frame-continuation-native.log`. Windows and macOS remain deferred.
Integration is local only; no push, PR or publication is included.
