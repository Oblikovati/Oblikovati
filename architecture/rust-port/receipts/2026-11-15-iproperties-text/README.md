# iProperties text editor checks

Three native tests passed on 2026-11-15 against application commit
`eed22e9e74a21a2aff9b1b5dc920ee02b46fd3d5`. They check text property editing,
undo and redo, save and reopen, and closing the editor when the active document
changes. The editor rejects non-text and parameter-owned replacements; full
typed editing remains open in
[issue #3958](https://github.com/Oblikovati/Oblikovati/issues/3958).

The native harness injects input into the window frame loop; OS-level input was
not tested. The existing local `README.md` edit did not affect the build. The
main worktree used its own Cargo target; concurrent agents used separate targets.

```sh
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib iproperties -- --test-threads=1
```
