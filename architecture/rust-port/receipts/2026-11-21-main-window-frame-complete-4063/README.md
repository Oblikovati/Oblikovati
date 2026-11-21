# Main Window Frame full Linux run

All 11 published scenario declarations passed. The outlines expanded the
run to 20 cases and 149 steps. The [Cucumber report](cucumber.json) and
[JUnit report](junit.xml) match the published feature SHA-256
`382162b5c11e33a252989ae65fee05e83d2e53a9ec3cac2f8ccad9ee05d7fe55`.
The [feature, support and fixture hashes](published-inputs.sha256) matched
before and after the run. The verifier checked all 20 final PNGs against
each case's source line and name.

The application was built from
`0905065a6d9b0da023f00c8713efcd60cb30d4e1`.
The [runner inputs](runner-inputs.sha256) match the private desktop
launcher added in `c6b2621cd`. It used X11, Xvfb, Openbox, tint2, AT-SPI
and Vulkan Lavapipe. Enabling accessibility before startup fixed the two
application-bar outline failures that occurred when the private bus started
disabled. Both cases checked the actual AT-SPI names and hit bounds.

The [active Part](active-part.png) and [minimum-size window](minimum-window.png)
show the application frame and viewport. The minimum-size case keeps the
saved Part unchanged. The [separate Close test](../2026-11-21-minimum-frame-viewport-4063/README.md)
checks that clicking Close at that size exits without a save prompt. The
full run also checked dirty Close and Cancel through both the custom
control and the desktop shortcut.

This receipt covers Linux desktop behavior. Windows runtime evidence for
[#4063](https://github.com/Oblikovati/Oblikovati/issues/4063) and strict
build and runtime provenance under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978)
remain open.
