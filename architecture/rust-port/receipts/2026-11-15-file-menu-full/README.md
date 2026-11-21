# File menu feature run

The pinned file-menu feature passed on 2026-11-15: eight scenarios and 59 steps
passed. The run checks menu presentation, submenus, document context, and
navigation. The final captures were reviewed. The native harness injects input
into the window frame loop; OS-level key and mouse delivery was not tested.

Application commit: `7f1b5cef45fc0ca30441e6befd4f0f2cd09ca267`.
The existing local `README.md` edit did not affect the build. The scenario
checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064` before and
after the run, and the live feature hash matched the pin afterward. The main
worktree had its own Cargo target; the concurrent agent used a separate target.
Go baseline details remain open where not already recorded.

```sh
OBK_BDD_ROOT=/tmp/obk-file-menu-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-file-menu-report-20261115 OBK_BDD_FEATURE=file-menu \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
