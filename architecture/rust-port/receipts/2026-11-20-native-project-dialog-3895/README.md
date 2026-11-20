# Project Dialog desktop acceptance

All eight scenarios and 100 steps passed on Linux with Xvfb and Vulkan
Lavapipe. Mouse clicks, Ctrl+N, Enter, and the arrow keys used X11 events.
The fixture path in the first Given was entered through the test harness.
The feature used the supplied Part and the three native project fixtures.

The desktop can load `.opj` files at startup before it opens the documents.
The first project is active. The dialog displays the project names and wraps
long paths. A single click inspects the selected project while the original
active project stays marked. The captures show the retained template details
and the independent library branches.

The regular desktop binary opened a Part and two projects in a bounded smoke
run. It reported the path of a missing project file. The benchmark registered
projects through the same desktop owner operation. The library steps observed
the separate default library entries. The fixture's generic library search
paths appear under Libraries.

The feature, support, and native fixture hashes matched before and after the
run. Cucumber and JUnit record the result. The source was unpinned and the
build and runtime inputs were unsealed. The local pass does not meet the
strict verification requirements in #3978.

The runner selected the full `project-dialog` feature with
`OBK_BDD_FEATURE=project-dialog` and `OBK_BDD_ROOT` set to the live source.
It used Xvfb at 1600 × 1200, XMODIFIERS set to `@im=none`, and Lavapipe.

```sh
xvfb-run -a -s '-screen 0 1600x1200x24' cargo test --locked -p oblikovati-head \
  --lib application_scenarios_use_the_native_window -- --nocapture
```
