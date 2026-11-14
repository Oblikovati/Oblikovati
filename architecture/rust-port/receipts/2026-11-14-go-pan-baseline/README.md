# Viewport Pan reference baseline

Go reference: `aa8297d1b` (`go-legacy`). Run from the Go `head` module:

```sh
xvfb-run -a go test ./ui -run '^(TestNavGestureMiddleButtonModes|TestApplyNavigationPansWithMiddleDrag|TestApplyNavigationModalFKeys)$' -count=1
```

Result: pass on Linux. The tests cover navigation gesture selection, camera
translation with middle-button pan, and modal function-key navigation. They
do not test the navigation bar highlight, status text, click-to-cancel, or the
context-menu commands required by these scenarios. Those behaviors are checked
by the Rust native scenario steps and captures, not by this Go baseline.
