# Native continuous Line tool

Date: 2026-11-06. Tracking: [#3830](https://github.com/Oblikovati/Oblikovati/issues/3830).

## Host operation and interaction

The sketch ribbon starts a continuous Line tool. Clicks add transient vertices.
The current document's inferred line-chain preview draws the accepted chain and
the cursor segment. It uses the same inferred chain operation as final creation.
Enter finishes. Escape finishes a chain with at least two vertices and abandons
an incomplete draft. C closes the chain with the first shared endpoint. Backspace
or U removes the last transient vertex. Tool replacement and focus loss abandon
the draft.

The chain is committed once. Consecutive segments share endpoint identity and
explicit closure reuses the first vertex. Inference reads current document
settings. Preview neither publishes model geometry nor allocates permanent IDs.
Placement uses the existing recipe validation before commit, including rejection
of zero-length closing edges. The typed API's documented zero-length sketch
entity behavior and its 80 Go parity fixtures remain unchanged.

Arming Line opens one bounded transaction. Save, Undo, Redo, and document exit
retain their existing host guards. A GUI snapshot identifies an owned sketch-tool
transaction so users can replace the tool without admitting commands into an
unrelated caller transaction. A read-only session query returns group identity;
it neither opens a group nor changes history. Cleanup compares that identity
before aborting, so an expired tool cannot abort a newer caller-owned group.

Enter and Backspace belong to the owned tool outside text entry, even when a
ribbon control retains keyboard-navigation focus. They are released from GUI
capture and do not reactivate a focused ribbon button. Active text entry keeps
its normal ownership.

A failed commit rolls back before rearming the draft. A commit veto already
aborts through the session owner. The desktop verifies the original sketch
recipes and the absence of another caller transaction before opening a fresh
Line scope. Reserved model IDs remain retired. Local preview or projection errors
retain the accepted draft and report their cause.

## Evidence path and limits

Owner tests cover shared endpoints, inference parity between preview and commit,
private preview and ID marks, Enter/Escape/Close/Undo, projection failures,
zero-length closure, invalid replacement, focus loss, redo retention, commit veto,
and cleanup when a newer caller group exists. ImGui tests cover the Line ribbon
at both display scales and the owned transaction guard. Native acceptance uses
mouse and keyboard input for creation, Fix, history, save and fresh-process
reopen, with MCPBridge read-only checks and screenshots.

The final local gate passed all 13 crates and all 354 head tests. Coverage is
91.96% to 99.60%, duplicated lines are 1.00%, and Markdown lint is clean. Windows
x64 GNU and macOS ARM64 host/GUI Clippy checks passed; these compile checks do not
claim native runtime coverage on those systems.

Native X11 acceptance covered incomplete cancellation, Enter, a private chain
preview, inert document Undo while Line owned its group, Backspace after ribbon
focus, C closure, line endpoint Fix and a final point Fix to DOF zero. Three
Undo/Redo steps removed and restored the same chain and Fix operations. Save and
fresh-process reopen retained the geometry and identities. MCPBridge supplied
read-only inspections and captures; the captures were inspected. Both test
desktops closed normally.

The numeric Line HUD remains unported. Construction and Centerline creation modes
now share the owner path described in
[native sketch formats](implementation-native-sketch-formats.md).
General Extrude and the cube/named-face journey remain incomplete;
Martensite#390 is the tracked analytic region prerequisite.
