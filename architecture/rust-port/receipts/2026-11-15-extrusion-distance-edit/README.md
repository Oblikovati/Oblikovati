# Extrusion distance edit

Issue: [#3968](https://github.com/Oblikovati/Oblikovati/issues/3968). This is a focused native-window regression, not an acceptance result for extrusion-07: taper remains unsupported and is tracked by [Martensite #463](https://github.com/Oblikovati/Martensite/issues/463).

The test used application source commit `2b1ca6b4f` and pinned Martensite `d051ff699982faba4ec2df3ee9a47e14f38dedbd`, on Linux with Vulkan Lavapipe under Xvfb. The final test ran on the matching worktree contents immediately before the application commit. It sent pointer and keyboard events through the native frame loop; it did not use OS-level XTest input.

```sh
OBK_EDIT_RECEIPT=1 OBK_BDD_REPORTS=/tmp/obk-edit-3968-receipt \
  VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  xvfb-run -a -s '-screen 0 1920x1080x24' \
  cargo test --locked -p oblikovati-head --lib \
  browser_distance_edit_previews_cancels_commits_and_reopens -- --nocapture
```

The run passed one test. It used the browser context menu to open a literal-distance extrusion, changed 10 mm to 20 mm, checked the private preview without committing, cancelled and checked the original, then reopened and accepted the edit. It checked undo, redo, save/reopen, and named-edge keys across the edit and reopen. The window closed at test end. An independent owner test checks that the preview replaces the original scene and that Cancel restores its triangles and outline. The shared-parameter and named-face behavior is covered by the typed host tests.

The screenshot shows the orange candidate preview and the distance draft. The viewport is front-facing; the image alone does not show the edited depth. `native-test.log` contains the test output. Their SHA-256 values are:

- `extrusion-distance-edit-preview.png`: `fa450619fe72a9e09f158cb6948f114bab10423165a50fd8f3591140041e8615`
- `native-test.log`: `cb135634f62af3ede2c6aa134e614783addd83fa26cd41395f2e61f3c2f751ed`

Sketch solve precision qualification remains open under [#3928](https://github.com/Oblikovati/Oblikovati/issues/3928).
