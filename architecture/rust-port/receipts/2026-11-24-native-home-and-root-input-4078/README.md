# Native Home and browser root input

Issue #4078. On `963cdeb6a`, the saved Part was already listed in Home's
Recent table. Its Name column shortened `planar-sketch-empty-xy` to
`plana...`, so the native File menu test could not find the full name. The
test now selects Home from its tab and opens a temporary copy named
`Menu.opd`. It checks the full recent name, muted File menu commands and
unchanged Part bytes.

The browser root test opened the correct context menu and kept the solid's
edge highlight after Escape. Its fixed deselection click then reached the
Appearance Browser. The test now finds empty viewport space from the current
layout and checks that the root selection and edge highlight disappear while
the solid remains visible.

Both X11 File menu tests and the X11 browser-root test passed with Xvfb,
Openbox and Vulkan Lavapipe. Strict Clippy passed for all desktop targets.
The pinned [File menu on Home](pinned-file-home/cucumber.json) passed seven
steps and the pinned [browser root menu](pinned-root/cucumber.json) passed 11.
Both feature files retained their source hashes throughout the runs.
