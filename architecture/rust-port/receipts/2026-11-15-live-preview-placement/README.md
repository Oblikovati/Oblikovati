<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OS-input tab preview placement

On 2026-11-15, the Rust desktop at
`c80e95c7f7e520bce96caab7126ee6b5828e58b0` ran under Linux Xvfb with Mesa
software Vulkan. The root README had an unrelated local edit. The application
binary SHA-256 was
`7e25f1068a1cf9771b833f2c1cd0777c55b80cbdd9cfd442283fee257c8cb71d`;
the MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.

XTest clicked New Part, minimized the ribbon and hovered over the part tab.
MCPBridge captured a preview below the tab without overlap or recursive image
content. Moving the pointer away dismissed it. Both captures were inspected.
The application window was resized to 1900 × 1000 on a 1920 × 1080 virtual
display. Coordinates in the input script are specific to this run. The
application and Xvfb server were closed afterward.

The native tests also cover placement above the tab. GUI tests cover viewport
edges, wrapping and the small-window fallback. Build provenance remains open
(#3978); this observation is not BDD acceptance evidence.
