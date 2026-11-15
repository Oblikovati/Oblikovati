<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OS-input tab preview check

On 2026-11-15, XTest sent mouse and keyboard events to the Rust desktop under
Linux Xvfb with Mesa software Vulkan. MCPBridge captured the application
window. The application was built from `6c0822e46e37e171f07965cf23a57bb5c11b802b`
with an unrelated modified root README. Its SHA-256 was
`2b34a99470313a1df927cadf0c64d0be8bdb4d9ff6faee583a9b69e460311841`;
the MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.

The test created a part, used the View ribbon to open a second window, hid the
document tabs, dismissed the menu, pressed Ctrl+F4, restored the tabs and
hovered the survivor. The captures show two window tabs before the close, one
part tab afterward and a preview of the surviving view. The first Ctrl+F4
attempt occurred while the menu was open and did not close a window; the
recorded successful sequence dismissed the menu before retrying.

The window was moved to the origin and resized to 1900 × 1000 through Xlib;
the display was 1920 × 1080. Input coordinates are specific to this run. The
preview overlapped the tab because there was insufficient space above it; this
separate layout issue is tracked in #3988. The application and Xvfb server
were closed after the run. Cache cleanup and identity behavior are established
by the owner and GUI regression tests, not by the screenshot alone. Build
provenance remains open (#3978), so this is diagnostic evidence.
