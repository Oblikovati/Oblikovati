# Live reference-key replay check

On 2026-11-15, the desktop binary from `6aa2c4909ca96eade3f24bc95e613d00c397714e`
ran under Linux Xvfb and Vulkan Lavapipe with the real MCPBridge. The binary and
bridge hashes, the saved input part, and the XTest and MCP probe scripts are in
the preceding [live extrusion check](../2026-11-15-live-os-extrusion/README.md).

The application opened a copy of that part. MCPBridge `get_reference_keys`
captured `before.json`. XTest sent Ctrl+S, Ctrl+W, and Ctrl+O; it typed the copy's
path into the Open dialog and clicked Open. MCPBridge captured `after.json`.
The application and Xvfb were closed after the run.

The six face keys, 12 edge keys, and eight vertex keys were compared as sets
and matched exactly. After reopening, `analysis_measure` resolved the first
face key from `before.json` and returned `2962.962962962962 mm²`. The reviewed
`replayed.png` was captured by MCPBridge and shows the restored solid and
`Extrusion1`. This checks one saved free-sketch prism on one Linux renderer;
it does not establish the parameter-driven, zero-DOF cube or general naming
coverage across edits. The sketch precision gap remains open (#3928).
