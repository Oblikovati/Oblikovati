# Shell properties at the minimum window size

At 640 × 480, the old Shell panel put Thickness after the value field
and left about 110 pixels of blank space below Cancel. Shell now uses
the shared left label column, outlined numeric field, floating panel
border and shadow, and compact action footer.
[Before](before.png) · [After](after.png)

The new native test failed on the old label placement and passed with
the new layout. It verifies containment, footer clearance, disabled
OK before face selection, and cancellation without creating a feature.
The main-ribbon selected-face refusal test, the placement round trip,
the owner test for a rejected preview, and five focused GUI tests
also passed.

Two published Ribbon Panel Expansion scenarios passed all 21 steps.
They cover moving Shell between the main and expanded panels and
retaining its placement across ribbon tab switches.
[Command placement](command-placement.json) · [Tab switches](tab-switch.json)
· [Published input hashes](current-inputs.sha256)

The published launch scenario still fails at the preview it expects
before face selection. The existing native expanded-panel test sees
zero selected faces after moving and launching the command, so it never
reaches its expected tolerance refusal. Both conditions reproduced with
the old Shell layout. [Launch scenario](preview-scenario.json)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`620741659` with the Shell layout change and pending local filesystem
and Ribbon Customization edits.
