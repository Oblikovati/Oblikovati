# Template chooser navigation regressions

Three native regressions passed on 2026-11-15 against application commit
`5db7c5ba3c0ce7b8d8f0059ff9b25be45d4f27e5`. They check vertical gallery
scrolling with selection retained, creation from a collapsed group, and Enter
after moving the folder highlight. The built-in metric template is used; these
tests do not replace the scenarios that require the missing named catalog.

The native harness injects input into the frame loop; OS-level input delivery
was not tested. The existing local `README.md` edit did not affect the build.
The test ran from an isolated target after the assembly presenter merge. An
earlier build started before that merge and was stopped without being counted.

```sh
CARGO_TARGET_DIR=/tmp/obk-template-merged-target XMODIFIERS=@im=none \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  chooser_navigation_regressions -- --test-threads=1
```
