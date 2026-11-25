# Tools Options expansion rows

The old flyout used text-width buttons with no icons and left blank space
below Pin. Tools Options now uses the shared full-width 28 px rows with
16 px leading icons and sizes the window to the rows and footer.
[Before](before.png) · [After](after.png)

The new native test failed on the old row pitch. It now verifies both icon
widths and the 32 px pitch, clicks the disabled iLogic row at its right edge,
then opens Reset UI Layout from the same column and cancels the confirmation.
The existing X11 reset test and the native Modify flyout test also passed.
[Reset confirmation](reset-confirmation.png)

Five published Ribbon Panel Expansion scenarios passed all 56 steps.
The GUI library’s ribbon filter passed 132 tests.

- [Open Tools Options](case-0.json)
- [Cancel layout reset](case-1.json)
- [Pin and minimize](case-2.json)
- [Highlight New menu](case-3.json)
- [Modify keyboard launch](case-4.json)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`a6dc2840f` with the flyout change and pending local filesystem and Ribbon
Customization edits. [Published input hashes](current-inputs.sha256)
