# File menu rerun

On 2026-11-15, the pinned File menu feature passed eight scenarios and 59
steps at application commit `5738199d932b19d5112413eb04e8c65bba94bb74`.
The live feature hash matched the pinned file after the run. The Home capture
was inspected. The native harness injects input into the frame loop, so this
run does not prove OS-level keyboard or mouse input. The existing local
`README.md` edit did not affect the build.

```sh
OBK_BDD_ROOT=/tmp/obk-filemenu-rerun-20261115 \
OBK_BDD_REPORTS=/tmp/obk-filemenu-rerun-report-20261115 \
OBK_BDD_FEATURE=file-menu \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
