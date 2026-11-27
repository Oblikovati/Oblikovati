<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Rejected Move endpoint diagnostic

On 2026-11-17, the Rust desktop change based on `c37206976` was tested on
Linux with Xvfb at 1920 × 1080 and Vulkan Lavapipe. The native test created an
ellipse fixture, selected it using X11 XTEST clicks, chose the same base and
end point, and inspected the resulting dialog. The attached capture shows the
error and the original ellipse. XTEST Cancel and Escape each returned Move to
geometry selection. Ending Move through the owner also closed the stale dialog.

The owner regression checked that rejection and cancellation retained the
sketch and undo label, then accepted a nonzero move. The current live move-19
scenario passed 10 steps in a frame-loop diagnostic before the final internal
stage rename. It used generated geometry and an unversioned scenario source.
The selected Move owner and native tests and head Clippy passed. A GUI suite
run before the final cleanup had 557 passes and one failure in the UI-scale
case tracked by #3850. An unrelated owner fixture also timed out while creating
a document; this was observed on the working tree and not bisected.

This change covers zero-displacement recovery only. It does not add the
constraint settings or policy behavior required by move-12 and later cases.
The pinned scenario was not run. Native build provenance remains open in #3978;
these diagnostics are not verified BDD acceptance. The full gate was not run.

A follow-up checked Escape sent directly to the desktop owner, without the GUI
modal dispatching its Cancel action. Before the fix it ended Move; after the fix
it returned to geometry selection and retained the undo label. The owner
regression and the X11 native dialog test passed. The source still needs the
constraint-policy contract and complete scenario acceptance described above.
