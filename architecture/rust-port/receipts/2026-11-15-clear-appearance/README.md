# Clear body appearance checks

On 2026-11-15, two native tests passed on merged application commit
`b629b41c31ccab8807de3374bf597031d5df483c`. They check Cancel and OK, the
selected body's override, retained part material and appearance, undo and redo,
and rejection of a stale selection in the panel and owner. The source branch
also passed the GUI action test, DockableWindow guard and strict Clippy; their
logs are retained here.

This increment supports one body selected before the command starts. Selection
after activation, multiple objects, faces and occurrences remain in
[issue #3962](https://github.com/Oblikovati/Oblikovati/issues/3962). The
native harness injects input into the frame loop; it does not prove OS-level
keyboard and mouse delivery. These tests do not complete the quick-access BDD
feature.

```sh
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  quick_access_toolbar::clear_appearance -- --test-threads=1
```
