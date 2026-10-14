# ADR-0026: Owned native keyboard and text input

Date: 2026-10-14. Status: Accepted for the Rust target design.
Boundary: Linux native events to the existing GUI and command frame loop.
Tracking: [#3630](https://github.com/Oblikovati/Oblikovati/issues/3630).

## Decision

Delete the symbol-only native key decoder. Copy each native key into one owned
physical-key event with modifiers and its optional semantic command. Retain a
press's key and command identity through release; native repeats do not repeat
Save/Undo/Redo. Focus or visibility loss clears press ownership. Use XKB key names for keyboard
positions, not a guessed native keycode offset or a typed character. Admit
detectable repeat explicitly and refresh the map on native mapping events.
Copy the active layout group's unshifted letter separately for Dear ImGui's
letter shortcuts, as the Go GLFW backend does. Preserve keypad identity and
retain both keys through release, even if the map changes while the key is held.

Define the internal presentation key enum and modifiers once in the viewport.
The GUI aliases these types and maps them to the pinned Dear ImGui keys. These
types belong to the internal viewport seam in ADR-0008. They add no public host
contract, automation DTO, wire method or add-in dependency.

Open the configured X11 input method and one input context per owned window.
Use its root preedit/status style after checking admission. Configure only
`LC_CTYPE` before native renderer threads; modelling numeric locale is unchanged.
Keep the input method, context and callback storage inside the platform adapter.
Retire the context before its window and display; provider destruction marks its
owned liveness state before any further lookup. Refuse missing providers and
unsupported styles with a named error. Do not silently select another provider.
An input-method or locale failure declines only the text channel. Keep the
window, physical keys and commands usable. Store the decline in the viewport's
owned health query and show it in the GUI alongside command feedback. A failed
lookup quarantines that context; later focus or key events do not retry it.
Configure and record locale admission before renderer threads even when the
text channel is unavailable. Never replace the requested provider or invent
character commits from physical key symbols.

Follow `XFilterEvent` and `Xutf8LookupString` for committed text. Keep physical
key transitions separate from committed Unicode scalars. A buffer-overflow
reply allocates exactly the reported capacity and repeats the lookup once, as
the XIM protocol requires. Reject invalid status, byte count or UTF-8. Native
control characters are key operations rather than text. Preserve plain text
and AltGr commits without turning them into application command chords.

Expand copied commits into the existing bounded event queue. Remaining scalars
keep native readiness active across poll limits. GUI input precedes command and
camera routing. Resolve the copied semantic command at that same dispatch seam;
physical keys and text do not create a second document command path. Enable
Dear ImGui keyboard navigation and retain the existing global chord/text-focus
policy.

## Evidence and acceptance

The Go baseline at `aa8297d1b0ac33c024f19e54821536d76e8a447a` uses the GLFW
native input bridge. The [Xlib input-method specification](https://www.x.org/releases/X11R7.6/doc/libX11/specs/libX11/libX11.html)
defines context admission, focus, filtering and UTF-8 lookup. The
[GLFW 3.4 native event path](https://github.com/glfw/glfw/blob/3.4/src/x11_window.c)
keeps physical key reporting separate from filtered text lookup.

Use named fake lookup replies for malformed and oversized UTF-8 commits. Drive
the actual X11 queue for physical keys, modifiers, repeat, release identity,
focus loss and bounded text delivery. Use a private Xvfb server with its own
international keyboard map for dead-key composition and supplementary Unicode;
the test must not change the desktop keyboard map. Check GUI key/text capture
remapped letters, active XKB groups, press identity across map changes,
and keyboard activation through the real native owner. Run the full local gate
and native/software driver checks before local integration. Clipboard, inline
preedit/candidate placement and additional native platforms retain their own
acceptance requirements.

The native AMD fixture passes with the explicitly selected local XIM provider
(`XMODIFIERS=@im=none`). The inherited `@im=ibus` provider is unavailable on
this machine's display and is refused with its configured name. That run is
not an IME-provider acceptance pass. The later startup regression verifies
that missing providers and unavailable locales leave native presentation and
the GUI usable, with a persistent visible decline. Keep availability separate from
keyboard-map and rendering evidence; no desktop service or keyboard map is
changed by the application tests.
