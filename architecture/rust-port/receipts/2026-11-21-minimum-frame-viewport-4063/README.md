# Main Window Frame at minimum size

Linux X11, Openbox, tint2 and Vulkan Lavapipe ran the published minimum-size
scenario on the local change based on `4c800ef89`. Its [Cucumber report](cucumber.json)
shows one scenario and five passed steps. The published feature SHA-256 was
`382162b5c11e33a252989ae65fee05e83d2e53a9ec3cac2f8ccad9ee05d7fe55`.
The saved native Part fixture SHA-256 was
`cac3df2175ff0f191abda7152700a4542903f2281d4a1668f09315d48dc66664`.

The [original minimum window](before.png) kept the solid visible in a
32-pixel strip. The [red regression](red.log.gz) measured edges 361 and 393
in the 640-pixel window. The [updated window](after.png) leaves at least
40 percent of its width for the viewport. The native regression also
grew the window and checked that its previous dock widths returned.
The [normal-size test](dock-at-normal-size.log.gz) passed docking, resizing,
closing and reopening Material Browser.

The scenario uses an unchanged copy of the saved Part. It checks that the
three icons draw, uses Maximize and Minimize, restores from the taskbar
and checks that the Part is still displayed. A [separate native click](close.log.gz)
proves that Close at minimum size exits with the same clean fixture and
shows zero save prompts. This keeps the scenario's final viewport check
in the same session.
