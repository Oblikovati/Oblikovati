# Exercise the Quick Access Extrude shortcut

Both current invocation scenarios passed individually, with 21 steps in
total. In the supplied empty Part, the added Extrude shortcut opened origin
plane selection. With View active, the shortcut also opened plane selection
and kept the View ribbon displayed. Escape returned to the same Part.

The test inspected the Quick Access shortcut by its rendered control.
The current added Extrude control uses a text caption. The two
reports cover invoking that control and cancelling plane selection.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Invoke an added command from quick access in an empty part | 10 | [Cucumber](invoke-empty-next/cucumber.json) |
| Invoke an added shortcut while another ribbon tab is active | 11 | [Cucumber](invoke-view-next/cucumber.json) |

The runs used the code in `4f3a2c24df73dda0e643fd00bb37dc603140bc30` with
existing local GUI filesystem, opening, and saving edits. The environment
was Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe. The
[input hashes](current-inputs.sha256) matched the shared source and the
isolated copy after the runs. Strict build and runtime provenance remains
open under #3978.
