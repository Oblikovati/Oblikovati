# Rectangle edge context-menu diagnostic

Base `b973d5975`, scenario snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`,
Martensite `bef25f47548431c8f128f08f66a1f719bcee4982`. Xvfb, 1600 × 1200,
Lavapipe, Linux, X11 XTEST mouse and Escape. The fixture is an 80 × 40 mm
rectangle created by the existing typed-rectangle test helper. It was chosen
to keep the selected sample clear of endpoints and dimension text.

The native test selected an edge, checked its theme-derived selected color,
opened the context menu, dismissed it with Escape, and checked selection was
retained. An empty-space click cleared the selection and restored the sampled
pixels. The screenshot was visually reviewed. This is diagnostic evidence, not
acceptance: the source scenario is `@capture_only` and `@capture_host_only`,
the exact reference fixture is unavailable, and build provenance remains open
(#3978). It does not establish the scenario's exact labels or physical-device
behavior. The test window closed.

The retained failed runs explain test development: run 2 clicked a point on a
small rectangle, run 4 asserted menu configuration as open state, and run 5 had
a compile error in the corrected text assertion. Run 8 is the final passing test.
