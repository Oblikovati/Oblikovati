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

This command does not provide the numeric Line HUD or additional creation-format
modes. It does not complete general Extrude or the cube/named-face journey;
Martensite#390 remains the tracked analytic region prerequisite.
