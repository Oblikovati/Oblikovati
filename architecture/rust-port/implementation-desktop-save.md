# Native Save command — partial R4/R7

The native Linux head now sends `Ctrl+S` to its document owner before the
frame's worker batch. It saves only the active document at its current path
through the released typed `Client.documents().save` method. Script and
worker calls retain the same canonical file owner. There is no second save
implementation, raw wire string, file codec, or geometry evaluation path.

The native X11 boundary recognizes Control+S on press and release. Plain S,
Control+Shift+S, and Alt+Control+S do not become Save. Release does not execute
Save. GUI input capture and configurable shortcuts remain open.

The owner reports a saved path, no active document, a required Save As path,
or a save failure. These notices reach the native title and survive subsequent
scene publication. A failure does not terminate the window, discard a document,
or clear its dirty state. The existing close guard still refuses dirty and
unsaved documents. Save As dialogs and close prompts require the shared GUI
composition and remain open; this change does not claim those controls.

## Local acceptance

```sh
VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers" \
  CARGO_NET_OFFLINE=true make gate
```

Named memory-file tests cover a saved dirty document, a missing destination,
no active document, release inputs, and a staging failure. The failure retains
the previously committed bytes and dirty state; a later Save can succeed.
Native tests run the production frame owner with the script wrapper, edit a
stored prism's width through the typed API, execute Save, check its displayed
destination, and reconstruct the parameter recipe from the actual written
package. X11 tests translate the native keymap with each modifier combination.

The title is current command feedback, not a GUI notification panel. The full
application port, foreground MCPBridge workflow, and complete shell controls
remain open. Tests are limited to this Linux host; no push or PR is included.
