# Scroll to a file in the native picker

The published **Open a saved document from Home** scenario passed all seven
steps on Wayland and X11 against the full shared fixture tree. Previously it
stopped with `single-occurrence.oad` below the visible rows. The binding now
uses wheel input to bring the target into the file list's visible area, then
clicks the row. The scenario confirms that the Assembly workspace opens with
its linked part loaded and Home remains available.

The native File Properties Escape regression and the Choose Component File
placement test also passed on each backend.

[Wayland report](wayland.xml) · [X11 report](x11.xml) · [Published input hashes](current-inputs.sha256)

The build used base `24e8de7cb66e5558416855ef87d2bb8961065103`, this test-runner
change, and existing local GUI filesystem and test edits. Wayland used headless
Weston with Pixman. X11 used Xvfb with Openbox. Both used Vulkan Lavapipe and
frame-loop input.
