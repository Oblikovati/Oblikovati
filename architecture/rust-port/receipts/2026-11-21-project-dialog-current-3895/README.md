# Project Dialog on both source versions

All eight scenarios passed on application revision `750c2b67f` under Xvfb,
Openbox, and Vulkan Lavapipe. The pinned run passed 101 steps. The published
run passed 100 steps and used the supplied native Part and project files.
Every scenario passed its cleanup hook and produced a final capture.

- Pinned. [Cucumber](pinned-cucumber.json), [JUnit](pinned-junit.xml.gz), and
  [source report](pinned-source.json).
- Published. [Cucumber](published-cucumber.json), [JUnit](published-junit.xml.gz),
  [source report](published-source.json), and [input hashes](published-inputs.sha256).

The Linux runs used native X11 mouse and keyboard events for dialog
interaction. The [library groups capture](library-groups.png) shows the active
project with Appearance Libraries open after the Material Libraries branch
closes. The [Go baseline](go-reference.log) passed four project-file codec
tests. The native scenario results cover the dialog and wizard.

Build and runtime provenance for strict acceptance remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).

## Windows wizard path

The wizard test now checks the selected directory and file name as path
components. The complete GUI test passed on Linux. A separate
Windows-target probe ran the same path-building and folder-label functions
under Wine and passed. The local MinGW linker could not link the complete
GUI test binary because its User32 import library lacks DPI functions used
by a dependency. The Windows GUI test still needs a hosted CI result.
