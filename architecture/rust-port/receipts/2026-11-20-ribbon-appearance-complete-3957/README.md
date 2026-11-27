# Ribbon Appearance desktop verification

All 18 Ribbon Appearance scenarios and 201 steps passed on
`07a996c9a17f792bd54b814bbd229e86f28dabdf`. The run used the verified scenario
snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`. Its feature file matches
the live feature byte for byte. All pinned support files also match the live
copies. The extra live references and their hashes are in `inputs.sha256`.

The run verifies appearance modes, retained expansion, shared panel titles,
submenu switching, and Reset Ribbon. Reset reopens the Model tab and restores
Team Web visibility while hiding Tutorials and the removed User Commands.
The repeated setup opens the real customization dialog and retains the
applied command.

The foreground application at `2c9c84184` opened the saved Part on X11.
MCPBridge `597a3d74477b96418c57b2e55f946ddc7129b7de` captured Team Web
before and after closing it. The Open in Browser button sent the
documentation URL to an isolated system browser handler. MCPBridge returned
the same active Part before and after. Clicking the main window's X closed
the application. Team Web uses the project documentation as its destination
and the Go desktop's browser handoff.

The filtered native test ran on Linux Xvfb at 1600 × 1200 with Vulkan
Lavapipe. Strict acceptance still requires bounded build and runtime inputs
under #3978.

```sh
scratch=$(mktemp -d)
python3 -B tools/bdd/verify_snapshot.py "$scratch/pin" --restore
export OBK_BDD_ROOT="$scratch/pin"
export OBK_BDD_FEATURE=ribbon-appearance
export OBK_BDD_REPORTS="$scratch/reports"
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
export VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers"
export XMODIFIERS=@im=none
xvfb-run -a -s '-screen 0 1600x1200x24' \
  cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```
