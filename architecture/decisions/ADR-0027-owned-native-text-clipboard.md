# ADR-0027: Owned native text clipboard

Date: 2026-10-14. Status: Accepted for the Rust target design.
Tracking: [#3630](https://github.com/Oblikovati/Oblikovati/issues/3630).
Boundary: platform clipboard I/O to the GUI owner in ADR-0008/0009.

## Decision and deletion gate

Replace the native GUI's dummy clipboard with one injected platform service.
Keep the internal `ViewportClipboard` interface separate from host wire
contracts. It owns text reads and writes, reports named operation failures, and
exposes no display, window, property, or callback pointer. The GUI owns a cloned
adapter and its health record. No add-in gains a GPL dependency or raw JSON path.

Validate byte budgets and embedded NUL before returning text to Dear ImGui or
writing it to the platform. The pinned imgui-rs 0.12.0 clipboard C callback uses
`CString::new(...).unwrap()` and aborts after a panic. Return no text for an
invalid value and keep the failure available to the health UI. An explicit
successful operation clears that operation's old failure. It does not retry an
edit or fabricate clipboard contents.

Use X11 selection ownership and negotiated text targets. Each transfer owns its
snapshot, identity, encoding, byte budget, and completion state. Direct and
incremental values use the same final text decoder. Validate property types and
formats. Incremental chunks can split UTF-8 characters; decode only after the
terminal property. Invalid or oversized transfers are terminal declines, not
partial success. Ownership loss preserves snapshots for accepted requests.

Answer selection requests while the application is idle. Clipboard readiness
joins the native owner wait and bounded event processing. Same-owner reads use
the retained snapshot. A GUI callback must not wait for a callback that needs
that same GUI thread. External waits have an explicit deadline and report
failure. Native peer failures must not invoke Xlib's default process-exiting
error path or close the renderer.

Use a checked Rust X11 protocol connection behind the platform interface.
Pin x11rb 0.14.0 and keep its types private. Its request cookies report peer
window errors to the owning operation; a disappearing requestor must not
replace process-global Xlib error policy. The selection window is hidden and
owns no renderer, GPU resource, or model. Its descriptor must join owner
readiness before native GUI adoption.

Before exit, transfer owned text through CLIPBOARD_MANAGER/SAVE_TARGETS. Use a
fresh requestor window and server timestamp for each conversion, one deadline,
and the existing direct/INCR service while the manager reads. Advertise the
SAVE_TARGETS marker and TARGET_SIZES metadata. No owner, no manager, and a refused
transfer are explicit normal outcomes; timeout and malformed replies are named
failures. The head still retires its GUI and viewport after a failed transfer.
This follows the [clipboard manager specification](https://wiki.freedesktop.org/www/ClipboardManager/).

## Required acceptance

The fixed Go baseline uses GLFW. Preserve external UTF-8 copy and paste, empty
text, ownership changes, large transfers, and shutdown. Follow the
[ICCCM selection specification](https://xorg.freedesktop.org/archive/X11R7.7/doc/xorg-docs/icccm/icccm.html).
A local string cache is not a substitute for native selection transfer.

Named fake providers cover callback safety and operation health. Real C
callbacks must copy Unicode and decline NUL without aborting. Independent native
peers must exercise direct and incremental transfers in both directions, target
negotiation, disappearance, timeout, idle servicing, and retained snapshots.
Use private displays so tests do not replace the desktop clipboard. Check the
real text-widget copy/paste path and rendered native result. Run the full local
gate before integration. These checks prove clipboard adoption; full R1V/R4
and application acceptance remain separate tracker requirements.

## Current evidence

The GUI adapter passes five callback and provider tests, including the real C
getter path with Unicode and NUL-bearing input. On a private Xvfb display, an
independent client passes five native tests for Unicode and empty text, target
and acquisition-time replies, invalid-input isolation, selection loss, and a
requestor destroyed before its reply. That destruction returns a checked error;
the endpoint still serves the next client.

The receive path now negotiates advertised targets and uses one deadline for
all transfer phases. Clipboard receive tests pass, including an independent
owner, no owner, retained own text, timeout recovery, separate target/text
budgets, Latin-1 conversion, and a scripted native INCR owner that splits one
Unicode character across properties. Invalid formats, NUL, incomplete UTF-8,
and decoded expansion past the budget are declines.

Outbound INCR now keeps an immutable snapshot per accepted request, respects
server request size and the injected chunk budget, and sends an empty terminal
property. Thirty-eight clipboard tests pass. Native checks cover large Unicode,
publish and ownership changes during a transfer, stalled-send retirement and
requestor destruction. Requestor event masks are restored after completion or
expiry. MULTIPLE conversions report each unsupported or aliased destination
as refused and cannot overwrite the pair list. STRING is advertised and emitted
only for losslessly representable Latin-1 snapshots. An injected pending-send
limit refuses new transfers without disturbing accepted ones. Malformed pair
properties receive a refusal while the endpoint remains usable.

The native head now injects the checked clipboard backend into its GUI owner.
Clipboard requests join the window and worker wait. The earliest pending-send
deadline bounds that wait without a periodic frame tick. Readiness preserves
events already copied into the protocol connection, and bounded event service
keeps peer failures local. Private-display checks prove external requests wake
an idle owner, copied requests remain available for service, and stalled sends
wake at their deadline. Native GUI Save, resize, wheel and physical keyboard
tests pass with this backend installed. A native test confirms an injected
clipboard connection decline stays visible in GUI health while the camera
and frame stay available. A real text widget pastes external Unicode, copies
and cuts it to an independent native client, then pastes the retained snapshot.
The shared private-display fixture prevents these tests from replacing the
desktop clipboard. An independent manager retains direct, empty and incremental
Unicode after the source endpoint closes. Refusal, malformed reply, manager
absence and a stalled manager are covered.

The full local gate passed across all 13 crates: minimum line coverage 91.82%,
viewport 92.98%, head 96.08%, GUI 99.39%, and duplication 0.75%. Native GUI,
clipboard health, Save, resize, wheel and keyboard checks also pass on AMD with
the inherited unavailable ibus provider. Rendering and commands remain available
with the named text-input decline.

The first gate run stopped with XGetImage BadMatch
on the 1440 by 900 test display: the default application window starts at
100 by 100 and cannot fit that display. The same compiled native composition
test passes on a 1920 by 1200 display. This also exposes a separate capture
safety requirement: a clipped drawable must produce a checked decline instead
of invoking the process-exiting Xlib error path. Clipboard transfers already
use checked requests and do not change this native image-read path.
