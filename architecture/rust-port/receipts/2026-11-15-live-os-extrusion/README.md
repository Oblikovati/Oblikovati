# Live OS-input extrusion check

On 2026-11-15, Linux Xvfb and Vulkan Lavapipe ran the desktop built from
`6aa2c4909ca96eade3f24bc95e613d00c397714e`. The binary SHA-256 was
`d3a42e76ca7b160791c384438c686afa3c396504ac538fafa66d4f97062388af`.
The real MCPBridge shared library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The application, bridge, and Xvfb server were closed after the check.

XTest sent operating-system key and mouse events to the application window.
The window had a root-screen origin of `(100, 100)`. The sequence was:

1. Press Ctrl+N, select **Standard metric part**, and click **Create**.
2. Click **Extrude**, select the XY plane in the viewport, draw a rectangle,
   and click **Finish Sketch**. The preview displayed one profile and 10 mm.
3. Click **OK**, then press Ctrl+Z and Ctrl+Y. `get_model_tree` reported zero
   features and bodies after undo, then `Extrusion1` and one body after redo.
4. Press Ctrl+S, type `/tmp/obk-live-os-input/test.opd`, and click **Save**.
   Press Ctrl+W, then Ctrl+O, type that path, and click **Open**.
5. After reopen, `get_model_tree` reported one extrusion and one body;
   `list_sketches` reported one XY sketch with four DOF. `get_reference_keys`
   returned face keys. Measuring the first returned face by its key returned
   `2962.962962962962 mm²`.

The copied `input.py.txt`, `probe.go.txt`, and `run.sh.txt` show the tools and
launcher used. The PNGs were captured by the MCPBridge `capture_window` tool
and reviewed. `test.opd` is the saved test artifact; `keys.json` is the key
query after reopening. This is a manual live check, not a generated BDD receipt.
It does not prove that a key survived replay: no pre-save key was compared.
The rectangle has four DOF and no size parameter, so this is not the requested
fully constrained 10 mm cube journey. Solve precision qualification remains
open in #3928.
