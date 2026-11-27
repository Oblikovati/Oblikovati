# Native Sketch Fix

Date: 2026-11-06. Tracking: [#3828](https://github.com/Oblikovati/Oblikovati/issues/3828).

## Command path

The sketch ribbon exposes Go's `Sketch.Fix` command. It accepts an analytic
point or line pick. A point submits the existing typed Fix constructor. A line
submits Fix constructors for both endpoints, as Go `applyFix` does. Shared point
identities are deduplicated in source order. This command does not substitute a
Ground relation or add a solver path.

Several Fix requests use one owned transaction through the existing typed API.
The operation produces one undo step. A failed constructor aborts the group. A
commit veto uses the transaction owner's existing abort. After rollback, the
desktop checks that sketch recipes match the original recipes before rebinding
the retained tool to the new generation and clearing stale picks. Reserved IDs
remain retired under the existing model contract.

Empty clicks keep the tool active. Escape, tool replacement, or a changed
external context uses the shared geometric-tool lifecycle. Unsupported curve
outlines do not enter the point/line pick filter. Existing constructor health,
redundancy, and convergence checks still apply.

## Evidence path

Owner tests check point and line picks, exact frozen coordinates, DOF, single-step
undo/redo, partial constructor failure, commit veto, rollback, tool reuse, and
cancellation. ImGui tests click the ribbon at both display scales and check
transaction guards. Native mouse/keyboard acceptance checks the visible Fix
command, sketch inspection, history, save/reopen, and normal test-window close.

The full local gate passed all 13 crates, including all 341 head tests. Coverage
is 91.96% to 99.60%, duplicated lines are 0.98%, and Markdown lint is clean.
Windows x64 GNU and macOS ARM64 host/GUI Clippy checks passed; these compile
checks do not claim native runtime coverage on those systems.

Native X11 acceptance clicked Fix, cancelled it, fixed two free points, and
observed DOF 4 to 2 to 0 with two canonical Fix constraints. It checked unchanged
geometry and identities, undo/redo, save, and fresh-process reopen. MCPBridge
provided read-only inspection and screenshots. The rendered ribbon and DOF-zero
sketch were inspected. Line picking and grouped failure/veto rollback are owner
tests; the live input test covers point Fix.

The general analytic sketch-region boundary remains blocked on Martensite#390.
This command does not complete the cube or general Extrude journey.
