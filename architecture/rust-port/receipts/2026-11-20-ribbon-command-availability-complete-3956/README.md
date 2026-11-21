# Ribbon Command Availability and native Sketch Text

At `200d549ef2c372db275009fb3bc21e54b51deada`, the complete live Ribbon
Command Availability feature passed all five scenarios and 39 steps. The
pinned source at `b576445b98cffd686dcc05aa8c017de2a591c8c8` passed all three
scenarios and 29 steps. The live file, support documents, and native CAD
fixtures retained their hashes throughout the run. The pinned and live files
differ because the live source also checks the assembly and drawing ribbons.

- [Live Cucumber report](live/cucumber.json)
- [Pinned Cucumber report](pinned/cucumber.json)
- [Outside-edit Sketch ribbon](live/Inspect-Sketch-commands-outside-sketch-edit.png)

The standalone native regression used X11 mouse and keyboard events to start
Text from the Part ribbon, pick a point, preview text, cancel, then preview and
accept a second entry. The test checked the saved glyph display for declines,
exited placement with Escape, and checked undo and redo. The dockable-window
registration guard also passed.

The focused live `text-08` scenario passed all four steps. It canceled `a`,
then reopened an empty editor with 3.05 mm height, zero rotation, and the system
default font. The [editor capture](text/cancelled-text-fresh-editor.png) shows
that state. The complete standalone Text feature has 38 scenarios. This run
exercised one. Editing existing sketch text, rich formatting, and further placement
workflows are still open under #3907.

The runs used Linux X11 under Xvfb at 1920 × 1080 with Vulkan Lavapipe. The BDD
run for Ribbon Command Availability used the application benchmark runner and
the normal feature filter. The focused Text run used the sketch benchmark
runner and the `text-08` filter. The separate native test used the
`sketch_text_ribbon_accepts_cancels_and_round_trips_in_history` filter.
Strict acceptance still requires bounded build and runtime inputs under #3978.
