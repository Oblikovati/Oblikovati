# Open an existing part: native input diagnostic

The filtered pinned scenario passed nine steps. The source was the local
snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`. The scenario has a
`needs_fixture` tag and native build provenance remains open (#3978). This
is diagnostic evidence, not verified acceptance.

A separate native test passed under Xvfb with Vulkan Lavapipe. The host created
and saved a part, then opened a drawing. X11 XTEST sent Control+O and clicked
the part row and Open. The test entered the fixture folder through injected
frame-loop input. It checked the active document identity, the unchanged list
of document windows, and the retained drawing tab. Both captured frames were
inspected. The test closed its window. It does not exercise a physical keyboard
or mouse, nor does it establish native text entry for the folder path.

The Go reference at `aa8297d1b` routes the file dialog to `OpenDocument`, then
`Workspace.Open`. The workspace returns and activates an existing document
when the full document name matches. This source review does not establish UI
or file-format parity. The current Rust dependency pin is Martensite
`bef25f47548431c8f128f08f66a1f719bcee4982`.

The shared X11 Escape helper was checked by the native Move preview and cancel
scenario: nine steps passed. The X11 keyboard utility uses the display's key
mapping rather than fixed keycodes.
