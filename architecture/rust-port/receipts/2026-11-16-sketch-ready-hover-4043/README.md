# Ready-state sketch hover diagnostic

This is diagnostic evidence, not verified scenario acceptance. The two related
scenarios are tagged `capture_only` and `needs_fixture`. The source pin is the
local snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`; native build
provenance remains open (#3978).

The native tests passed on Xvfb with Vulkan Lavapipe. X11 XTEST moved the
pointer onto a line endpoint, a line, and empty space. The checks cover the
visible point and line cues, themed line highlight, idle persistence, focus
loss, starting a dimension tool, undo, and preservation of selection and
history. Keyboard input and focus loss were injected into the frame loop. The
host created the fixture. The screenshots were inspected. `red.log.gz`
records the expected missing-cue failure before implementation.

The existing Ready selection, overlap selection, and active-tool hover
scenarios also passed in separate filtered native diagnostics. Their reports
are in `selection`, `overlap`, and `tool`. They are not portable acceptance.

The release frame test used a 200-segment sketch, Xvfb at a checked 1440 × 900
extent, and Lavapipe. After 20 warmups, 120 bounded frame-loop calls had a
39.946 ms median, 44.102 ms p95, 44.817 ms p99, and 45.951 ms maximum. All
120 exceeded 16.67 ms. The test confirms a hover cue exists after the loop.
It injects pointer motion and excludes OS delivery and display scanout. It is
not a production GPU result. Raw samples are in `performance.log.gz`. The
60 fps target remains open (#4032, #4035, Martensite #501 and #505).
