# Native Undo and Redo — partial R4/R7

The native Linux head recognizes Control+Z for Undo and Control+Y or
Control+Shift+Z for Redo. The X11 boundary copies semantic key inputs without
retaining the native event or display address. Release does not execute a
command. Plain letters, Alt combinations, and Control+Shift+Y are not history
commands. Configurable shortcuts and GUI input capture remain open.

The document owner uses the released typed transaction client. Recipe history,
identity restoration, dirty state, generation updates, and recompute remain in
the canonical host/session path. These commands do not operate on the camera
Previous View stack. There is no duplicate history engine or direct kernel edit.

As in Go `app/bindings.go`, an open bounded transaction prevents Undo and Redo.
This is a no-op: no history entry is removed, the transaction stays open, and
no command result replaces the current feedback. Other failures reach native
command feedback without terminating the window. A missing active document is
reported with the expected state.

## Local acceptance

```sh
VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers" \
  CARGO_NET_OFFLINE=true make gate
```

Tests cover key press/release and modifier classification, restoration of a
saved clean state, redo to the edited dirty state, an open transaction, and
missing document ownership. Native tests edit a stored prism parameter, run
Undo and Redo on the script/document owner, and publish through the production
frame loop. Both actions reproduce their expected rendered pixels exactly and
retain the current camera. The title displays the completed action.

The shared native parameter-edit fixture also serves Save acceptance so the
command tests do not implement separate fixture-edit paths. Full GUI history
controls, selection/tool behavior, and foreground MCPBridge acceptance remain
open. Testing is limited to this Linux host; no push or PR is included.
