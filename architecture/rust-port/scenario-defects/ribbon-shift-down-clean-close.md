# The Shift+Down scenario asks for Cancel on a clean draft

**Select adjacent available commands with Shift and Down** checks that
2D Sketches and 3D Annotations are both highlighted, the target stays
empty and the dialog has no pending edits. It then asks the user to
click Cancel and expects the dialog to close.

The functional notes describe Close as available on a clean draft.
The observed Shift+Down sequence used Cancel after a later transfer
staged rows. That transfer is outside the published scenario.
The clean-draft scenario should click Close after checking the selection.

[Native results](../receipts/2026-11-26-ribbon-source-shift-selection-3856/README.md)
