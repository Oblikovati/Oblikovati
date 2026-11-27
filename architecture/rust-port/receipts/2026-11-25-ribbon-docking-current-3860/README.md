# Current native ribbon docking

Twenty-one of the 22 current scenarios passed, with 223 steps. They
cover docking and undocking, the floating title menu, switching compact
groups, Finish Sketch from the left dock, and retaining the left dock
across the published Part, Assembly, and Sheet Metal documents.

The document-switch case exposed a clipped Sheet Metal tab label. The
final run verifies the full label in the left tab column after the width
fix. The [placement retest](placement-retest/cucumber.json) also passed
all 19 steps for docking on both sides and returning to Top.

The active Shell scenario is blocked under Martensite #554. A native
probe opened the same published Part, started Shell, and selected one
visible face. The pinned kernel reported a vertex tolerance of 0.0005
against a limit of 0.0000001 and the desktop rejected the preview body.
The [probe log](shell-probe/native.log), [source](shell-probe/probe.rs.txt),
and [frame](shell-probe/published-shell-refusal.png) record the refusal.
The blocked scenario's [Cucumber report](case-10/cucumber.json) records
the unbound fixture setup from the regular scenario run.

The builds used base revision `a8971f506be6abf0e99a09bbfc408704e868bd16`, the
fixture bindings available for each run, and existing local GUI
filesystem, opening, and saving edits. The native window ran on Linux
X11 with Xvfb, Openbox, and Vulkan Lavapipe. The feature and fixture
hashes matched the shared source and isolated copy before and after
the runs. Strict build and runtime provenance remains open under #3978.
See the [combined report](cucumber.json), [scenario inventory](scenario-inventory.json),
and [input hashes](current-inputs.sha256).
