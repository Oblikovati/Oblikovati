# Sketch Create expansion rows

In Normal ribbon appearance, Create Block and Precise Input now use the
shared 28 px rows, 16 px icons, and compact Pin footer. The old flyout used
unequal text-only buttons and measured 23.5 px between their captions.
The native Line test measures a 32 px row pitch, checks both icons, and
clicks the right side of the Precise Input row to open and close the
existing toolbar. [Before](before.png) · [After](after.png)

Four published Ribbon Panel Expansion scenarios passed all 44 steps.
They cover the Create panel, cancellation from Create Block, commands
outside sketch edit, and the neighboring Constrain expansion. The
existing Tools Options native check and 132 GUI ribbon tests also passed.

- [Open the Create expansion](case-0.json)
- [Cancel Create Block](case-1.json)
- [Create outside sketch edit](case-2.json)
- [Open and close Constrain](case-3.json)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`d22a91b89` with the flyout change and pending local filesystem and
Ribbon Customization edits. [Published input hashes](current-inputs.sha256)
