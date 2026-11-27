# Full pinned Ribbon Panel Expansion run

The [native benchmark](cucumber.json) passed 57 of 67 scenarios and 643 of
653 executed steps. The other ten cases stopped at an undefined step. The
gaps involve Shell, a Tools Options menu, a temporary ribbon pin, and
the Explore panel. All matched steps passed.

The run used the Head build at `d17002295` with existing local
file-browser edits, Linux X11, Xvfb, Openbox, and Vulkan Lavapipe. The
[source revision](source-revision.json) records a clean pinned checkout at
`b576445b98cffd686dcc05aa8c017de2a591c8c8` before and after the run.
The feature SHA-256 was
`645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2`.

Fifty-four passing declarations gained R0 baseline records. The
[published run](../2026-11-24-ribbon-expansion-full-3856/README.md)
uses a different feature hash and remains recorded separately.
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
