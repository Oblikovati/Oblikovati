# Live offset work plane

The standalone desktop built from `e2bb3591b` ran on Linux with Xvfb and
Vulkan Lavapipe. Its SHA-256 was
`709011251e54d9442542e971f20aff6d0659cd4e53c9fc10df1d4334e25afa11`.
The MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The desktop and Xvfb server were closed after the run.

The input was a copy of `../2026-11-15-live-parameter-cube/cube.opd`.
XTest sent OS mouse and keyboard events; MCPBridge provided screenshots and
read-only plane queries. The window was 1900 by 1100 pixels.

1. Click Plane in the 3D Model ribbon and select XZ Plane in the browser.
2. Enter `0 mm`. The editor displays a validation error and disables OK.
3. Enter `50 mm` and accept. The browser shows Work Plane1.
4. Undo and redo with Ctrl+Z and Ctrl+Y. The plane disappears and returns.
   Its API reference remains `datum/v1/2`, its origin is `[0, -5, 0]` cm,
   and its offset is 50 mm.

The saved JSON queries, reviewed screenshots, host log and input scripts are
retained here. This is a live check of one Linux configuration, not a full
work-plane or platform qualification. See #3977.
