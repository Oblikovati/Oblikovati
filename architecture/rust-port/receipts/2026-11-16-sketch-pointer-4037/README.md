# Pointer-driven sketch creation diagnostic

This is diagnostic evidence, not verified acceptance. The source is unversioned
and the scenario is tagged `needs_fixture` and `needs_capture`. The live feature
SHA-256 before and after this run was
`f5e24ccd8a393e2ce319b45405b40bc78a1304f3b74b7df99652af5493ef765e`.
Native build provenance remains open in #3978.

One filtered scenario passed all nine steps under Xvfb and Vulkan Lavapipe.
The initial part and sketch use the test fixture. X11 XTEST selects Line, clicks
two viewport positions, moves the pointer, and sends Escape. The test checks the
committed line, pending preview, cancellation, and retained geometry. The two
frames were inspected. This does not test a physical mouse or keyboard.

The first run reached the cancellation check but expected the text `Ready`.
Xvfb displayed an input-method warning in that status area. The final check reads
the sketch Ready state instead and also checks the preview and pixels.
