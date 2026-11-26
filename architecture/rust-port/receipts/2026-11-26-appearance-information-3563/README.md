# Appearance Information

The Generic appearance's Information section shows editable Name,
Description, and Keywords fields, the Generic type, and the Document
source. The name above the sections uses the same draft. Description
and keywords are stored in an optional document recipe map keyed by
appearance ID. Empty metadata clears the entry. Existing parts open
with blank values when the metadata is absent.

The published Information scenario passed seven steps per backend.
The Apply and find regression passed thirteen steps per backend.
The direct native check typed description and keywords, applied the
edit, saved and reopened the Part, opened the selected appearance,
and verified both values in the editor. All ten package asset tests
passed, including a round trip with multiline text and metadata removal.

The build used base `4edda3f66`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The [source and fixture hashes](published-inputs.sha256) matched
before and after testing. Strict provenance remains open under #3978.

- [Wayland report](wayland.json), [step arguments](wayland-arguments.json), and [Information screenshot](information-wayland.png)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- Native save and reopen on [Wayland](reopen-wayland.txt) and [X11](reopen-x11.txt)
- Apply and find on [Wayland](apply-wayland.json) and [X11](apply-x11.json)
