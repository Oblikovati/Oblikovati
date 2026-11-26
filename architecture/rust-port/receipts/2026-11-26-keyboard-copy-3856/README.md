# Copy the Keyboard command table in display order

Copy to Clipboard exports all rows of the Keyboard draft as a
tab-separated table. It uses the same case-insensitive command-name order
as the Keyboard list. The native test searches for Undo, presses Copy,
reads the text through the desktop clipboard channel, and checks the four
columns, complete catalog, and display order. Apply remains disabled.

The baseline on both backends copied the complete table in registration
order and failed on the order check. The build used `5f3cd5f19` with the
clipboard test and existing local filesystem edits. The final build
added the sort change.

X11 used private Xvfb with Openbox. The Wayland window ran in Weston
13 with the X11 backend, fullscreen kiosk shell, and Pixman. Weston
ran on a separate 1440 × 900 Xvfb display with Openbox. An XTest click
on the nested output reached the Wayland application as a compositor
pointer event. Both native application runs used Vulkan Lavapipe.

Run the focused test with the private display and compositor set up.
Keep the nested X11 `DISPLAY` available when using Wayland so the test
can click its output window.

```sh
cargo test --locked --offline -p oblikovati-head --lib \
  keyboard_clipboard_copies_the_complete_catalog_in_display_order \
  -- --ignored --test-threads=1
```

[Test results](test-results.txt)
