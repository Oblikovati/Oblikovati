# Highlight New over expanded Options

The focused scenario passed all 12 steps on both the published source and
the pinned `b576445b98cffd686dcc05aa8c017de2a591c8c8` snapshot. The
menu opens above the Options additional commands, Escape closes the menu
alone, and clicking Tools closes the expansion while the Part stays active.
The menu currently displays an empty state for release highlights.

- Published [Cucumber](published/cucumber.json), [JUnit](published/junit.xml),
  and [source report](published/source-revision.json).
- Pinned [Cucumber](pinned/cucumber.json), [JUnit](pinned/junit.xml), and
  [source report](pinned/source-revision.json).
- [Menu over Options](published/highlight-new-over-options.png) and
  [Options after Escape](published/options-after-highlight-new-escape.png).

The run used Pop!_OS 24.04 with Xvfb, Openbox, and Vulkan Lavapipe. The
runner opened Highlight New with X11 pointer input. The build used the
changes in this commit on `d2b6f0834`, with pre-existing local GUI filesystem,
document-opening, and document-saving edits present. The Martensite lock
resolved to `bef25f47548431c8f128f08f66a1f719bcee4982`.

The feature path was
`features/application/ribbon-panel-expansion/ribbon-panel-expansion.feature`.
The published feature and Part fixture hashes matched the
[previous input snapshot](../2026-11-25-ribbon-panel-expansion-current-3856/current-inputs.sha256).
The pinned feature hash is
`645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2`.

```sh
OBK_BDD_AREA=application OBK_BDD_FEATURE=ribbon-panel-expansion \
  OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios \
  OBK_BDD_SCENARIO='Open a main-panel menu while additional commands are visible' \
  cargo test --locked --offline -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --test-threads=1 --nocapture
```

The pinned run omitted `OBK_BDD_ROOT`. Shell previews remain open work
under [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

## Layout-reset regression

The confirmation warning wraps after `to` in the current design. The
assertion now joins the rendered runs before checking the complete warning.
The published reset cancellation scenario passed all ten steps. The separate
`x11_options_layout_reset_can_be_cancelled` test passed with native pointer
input through the expansion, confirmation, and No button.
[Cucumber](reset-regression/cucumber.json), [JUnit](reset-regression/junit.xml),
[source report](reset-regression/source-revision.json), and
[confirmation capture](reset-regression/tools-options-reset-confirmation.png).
