# Generic Appearance Editor

The Appearance Browser creates and selects a document-owned Generic
appearance. A separate floating editor shows its shaded preview and
provides name, RGB color, glossiness, and highlight controls. The editor
uses the application's existing basic and custom color palettes.
Creation and Apply record separate document history edits.

The native flow selects RGB 0, 128, 192, checks the live preview, applies
the name Custom Ocean Blue, and finds the applied asset in the document
browser. It verifies Generic and Uncategorized in the list view and an
empty library result. A close request with unapplied color opens Save
Edits. Choosing Cancel leaves the editor and its pending draft visible.

This receipt covers four published editor scenarios and 42 steps per
backend. Six Appearance Browser regression scenarios also passed 39
steps per backend. Image editing, additional Generic effects, metadata
editing, preview settings, and other appearance types remain open.

The build used base `61c979fa5`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The [source and fixture hashes](published-inputs.sha256) matched
before and after testing. Strict provenance remains open under #3978.

| Scenario | Wayland | X11 |
| --- | --- | --- |
| Create a generic appearance | [Report](create-wayland.json), [arguments](create-wayland-arguments.json) | [Report](create-x11.json), [arguments](create-x11-arguments.json) |
| Choose a color | [Report](color-wayland.json), [arguments](color-wayland-arguments.json) | [Report](color-x11.json), [arguments](color-x11-arguments.json) |
| Apply and find | [Report](apply-wayland.json), [arguments](apply-wayland-arguments.json) | [Report](apply-x11.json), [arguments](apply-x11-arguments.json) |
| Cancel closing with pending changes | [Report](close-wayland.json), [arguments](close-wayland-arguments.json) | [Report](close-x11.json), [arguments](close-x11-arguments.json) |
| Appearance Browser regression | [Report](browser-wayland.json), [arguments](browser-wayland-arguments.json) | [Report](browser-x11.json), [arguments](browser-x11-arguments.json) |

- [Editor at creation](create-wayland.png)
- [Live blue preview](color-wayland.png)
