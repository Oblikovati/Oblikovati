# Native cube journey after the forward-ray fix

Application commit: `5556501a721667ecaa933f84f12d6cdc58a277e5`.
Environment: Linux, Xvfb 1920 × 1200, Vulkan Lavapipe. The test sends
keyboard and pointer events to the native window and checks owner state and
rendered output. The window closes at test completion.

```text
$ xvfb-run -a -s '-screen 0 1920x1200x24' env XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json CARGO_NET_OFFLINE=true cargo test --locked -p oblikovati-head --lib native_new_part_button_creates_and_selects_an_unsaved_document -- --nocapture
test native::gui_loop_tests::native_new_part_button_creates_and_selects_an_unsaved_document ... ok
test result: ok. 1 passed; 0 failed; 0 ignored; 0 measured; 862 filtered out
```

The test creates a part and parameter, draws and constrains a rectangle,
creates an extrusion, switches model states, changes the parameter, and
checks undo and redo. It saves and reopens the part, verifies the solid and
its body, face, and topology keys, and resolves a public face reference. This
is a targeted native integration test; it is not a run of the pinned BDD
scenario set or evidence for other operating systems.
