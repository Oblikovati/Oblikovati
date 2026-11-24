# Current native assembly and clean application closes

Three current scenarios passed individually, with 16 steps. Cancelling
the save-changes prompt kept the modified Assembly and desktop open.
Closing an unchanged Assembly and closing from Home both completed
without a save-changes prompt.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Cancel closing the application when an assembly has unsaved changes | 10 | [Cucumber](cancel-dirty-assembly-next/cucumber.json) |
| Close the application when the open document has no unsaved changes | 3 | [Cucumber](clean-assembly-next/cucumber.json) |
| Close the application when no document is open | 3 | [Cucumber](empty-session-next/cucumber.json) |

The successful exits retain a frame taken immediately before Close.
The runs used the code in `e871095fe01915c8528bdcb076b201480f85a2b8` with
existing local GUI filesystem, opening, and saving edits. The environment
was Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe. The
[input hashes](current-inputs.sha256) matched the shared source and
isolated copy after the runs. Strict build and runtime provenance remains
open under #3978.

Together with the [modified Part cancellation](../2026-11-24-application-closing-current-dirty-part-4030/README.md),
four of seven current scenarios have passing native records. The remaining
cases exercise Save and Discard at the existing path and cancelling a
first Save As during exit.
