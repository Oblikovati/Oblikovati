# Docking System with the Studio desktop

All 11 published scenarios and 111 steps passed across two native X11
runs on a 1440 × 900 display. The suite report contains the ten scenarios
that passed all 105 steps, plus a restart failure in the old verifier.
The separate restart report records all six steps passing with the
updated verifier. [Suite report](suite/cucumber.json) ·
[Restart report](restart/cucumber.json)

The old restart driver used recorded toolbar positions. Those clicks
missed the browser commands in the Studio layout. The updated driver
takes the saved profile produced by the scenario's X11 drags. It launches
two standalone desktop processes with that profile and checks the
original and restored dock IDs, tab order, positions, and sizes. The
captured frames were byte-for-byte identical and the Part remained visible.
[Before restart](restart/before.png) · [After restart](restart/after.png) ·
[Measurements](restart/restart-summary.json)

The environments used Xvfb, Openbox, and Vulkan Lavapipe. The suite used
the float-and-redock bindings pushed in `264c8aad0`. The restart run
included the updated driver and the same desktop behavior. Both builds
included pending local filesystem and Ribbon Customization edits.
[Scenario and main Part hashes](current-inputs.sha256)
