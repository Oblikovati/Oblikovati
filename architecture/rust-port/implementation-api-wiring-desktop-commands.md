# Native command execution

The desktop handles API 0.166.0 `commands.execute` on the native owner
thread. Home, Zoom All, Look At, and the executable sketch entries from
the marking-menu catalog use the same action dispatcher as the desktop.
The API checks command availability and returns dispatch errors.

Home and Zoom All publish pending model geometry before framing it. That
lets an add-in frame geometry it authored in the same owner poll.

The native test checks that Home changes the active camera and that an
unknown command or a sketch command outside sketch edit returns an error.

The MCP Bridge run on native Wayland verified Home, Zoom All, and
unknown and disabled commands. The original `mcplive --part paramseam`
case passed after previously stopping at `View.Home`. It framed and
captured the parameter-driven part after its dependency checks.

The native typed-client test also exercises the argument-free
`view.getCamera` call used by the Rust API client. The host now defaults
an empty request to the active document.
