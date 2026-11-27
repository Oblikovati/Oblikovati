# Model Browser suppression and table rows on Linux

Five published scenarios passed 82 steps on each backend. They cover
switching between suppressed and computed members, accepting or cancelling
a suppression change, and accepting or cancelling an inserted member row.
The test clicks use the active desktop backend. After suppression, the
published scene has zero visible bodies and the sampled face is gone.

- Switching members on [Wayland](switch-wayland.json) and [X11](switch-x11.json)
- Cancelling suppression on [Wayland](cancel-wayland.json) and [X11](cancel-x11.json)
- Applying suppression on [Wayland](apply-wayland.json) and [X11](apply-x11.json)
- Cancelling insertion on [Wayland](cancel-insert-wayland.json) and [X11](cancel-insert-x11.json)
- Applying insertion on [Wayland](apply-insert-wayland.json) and [X11](apply-insert-x11.json)

The build used `47963834b63899b37cc29317962e7e976c056dc1` with the existing
local GUI filesystem test-module visibility edit. Wayland used nested
Weston 13 and X11 used Openbox. Both used Xvfb and Vulkan Lavapipe.
The [input hashes](current-inputs.sha256) matched the shared source before
and after the runs. Strict build and runtime verification remains open
under #3978.
