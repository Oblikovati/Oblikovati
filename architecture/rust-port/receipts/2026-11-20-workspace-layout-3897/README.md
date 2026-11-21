# Workspace Layout desktop run

The full Linux desktop run passed 53 cases and 523 steps on
`7f735741ed6c768c89162752f2cd09e14912ba52`. The inventory counts 52 scenario
declarations because one outline has two examples. The feature, support
files and native Part and Drawing fixtures matched the recorded hashes
before and after the run.

The suite includes browser docking and restore, viewport and frame spacing,
the View checklist, browser tab selection, keyboard search, and selection of
saved tree entries. The saved Drawing capture shows its browser after restore.

A separate foreground run opened the saved Part under Openbox on a
1600 × 1000 Xvfb display. MCPBridge
`597a3d74477b96418c57b2e55f946ddc7129b7de` used API v0.160.0.
Native X11 clicks on Rule Log and Rule Automation left both menu boxes
unchecked. The Mini-Toolbar click cleared its check mark. The captured
pixels confirmed the changes, and MCPBridge listed the same open Part
throughout. Alt+F4 closed the window. See the three `mcp-*.png` files.

The desktop also passed a Windows GNU cross-compile. The live scenario source
had no Git revision. Strict verification still requires a current pinned
scenario source and bounded build and runtime inputs under #3978.

```sh
export OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios
export OBK_BDD_FEATURE=workspace-layout
export OBK_BDD_REPORTS=$(mktemp -d)/reports
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
export VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers"
export XMODIFIERS=@im=none
xvfb-run -a -s '-screen 0 1600x1000x24' \
  cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```
