# Live viewport probing (import → render → capture → analyse)

How to drive the **running** application headlessly to import a CAD file, frame it, switch on a
debug render, capture the viewport framebuffer as a PNG, and inspect it — the loop for diagnosing
tessellation/render defects (back-faces, cracks, wrong face sense) without a human at the screen.

It runs over the **MCP bridge** add-in, which serves the `api/wire` surface as MCP tools on
`http://127.0.0.1:7800/mcp`. The relevant tools (all forward to wire methods):

| tool | wire method | what it does |
|---|---|---|
| `import_file` | `documents.import` | import STEP/STL/OBJ/3MF into the active part |
| `execute_command {id:"View.Home"}` | `commands.execute` | frame the model, default isometric (`View.ZoomAll` = fit, keep angle) |
| `set_camera {eye,target,up,fov}` | `view.setCamera` | exact angle — **keep fixed across probes for comparable captures** |
| `set_normal_debug {on}` | `viewport.setNormalDebug` | front-facing **green** / back-facing **red** (winding / flipped-normal defects) |
| `set_face_colors {on}` | `viewport.setFaceColors` | every B-rep face a distinct color = `faceDebugColor(globalFaceIndex)` — map a region back to a face index |
| `capture_viewport {path}` | `viewport.capture` | write the framebuffer to a PNG (host writes it the next frame, **atomically**) and return it as an MCP image |

## The loop

1. **Build the app** (the head binary links the kernel changes you're testing):

   ```sh
   cd Oblikovati/head && OBK_ADDINS_DIR=$(pwd)/addins go build -o oblikovati-head ./cmd/oblikovati-head
   ```

2. **Build + install the MCP bridge** into the app's add-in folder (`make install` builds the
   c-shared `.so` and copies it + `manifest.json` to `Oblikovati/head/addins`):

   ```sh
   cd Oblikovati.AddIns/oblikovati-mcp-bridge && make install
   ```

3. **Restart the app** so it loads the rebuilt head + the new `.so`. It serves MCP on `:7800`:

   ```sh
   pkill -f '[o]blikovati-head'; sleep 2
   cd Oblikovati/head && setsid env OBK_ADDINS_DIR=$(pwd)/addins ./oblikovati-head >/tmp/obk-head.log 2>&1 </dev/null &
   sleep 10 && ss -ltn | grep -q :7800 && echo "MCP up"
   ```

4. **Import + frame + debug-render + capture** in one shot with the `mcpshot` driver (it closes all
   docs, creates a part, imports, frames, sets the debug mode, captures):

   ```sh
   cd Oblikovati.AddIns/oblikovati-mcp-bridge && go build -o mcpshot ./cmd/mcpshot
   # normal-debug (default): front green / back red
   ./mcpshot --file /path/EDF.STEP --out /tmp/oblikovati-capture.png
   # face-index colors instead:
   ./mcpshot --file /path/EDF.STEP --faces=true --normals=false
   # a FIXED angle for repeatable comparison across probes:
   ./mcpshot --file /path/EDF.STEP --eye=120,-120,120 --target=25,0,25
   ```

5. **Analyse** the PNG — read `/tmp/oblikovati-capture.png` (it is a real local file, atomically
   written, so a reader never sees it mid-write).

## Notes

- **Consistent angles.** The default frames with `View.Home`; for A/B comparisons across code
  changes pass the same `--eye/--target` every time so the captures are pixel-comparable. Record the
  angles you settle on in the probe command (or a tiny wrapper script) so a later session reproduces them.
- **Camera too close on a bare import.** A programmatic `import_file` does NOT auto-frame (the GUI's
  File▸Import does); always frame with `--home` (or `set_camera`) before capturing.
- **The viewport tessellation cache** (`head/ui/viewport_cache.go`, keyed on `ModelGeometryVersion`
  + style + visible bodies + face-index flag) means an expensive tessellation runs once, not per
  frame — so a slow capture points at the tessellation cost itself, not the cache. (A pathological
  mesher can still freeze the one tessellation on import / on any cache-invalidating change.)
- **Headless E2E vs live.** The bridge's `go test ./bridge` E2E spins up the host in-process and is
  the place to assert *model* behaviour; the *render* (framebuffer) only exists in the live windowed
  app, so `capture_viewport` must be probed against a running `oblikovati-head`, not the E2E test.
