# Typed feature rename checkpoint replay

Issue #4079. The regression opens a Part with Extrude and Fillet, retains
the current part checkpoints, and renames the Fillet via the typed API.
At `e6b6f82e3`, the full replay count rose from one to two during the
rename. Routing the rename through the existing checkpoint edit path keeps
that count unchanged. The reply returns the new name with no health error,
the published checkpoint belongs to the current generation, and the
Fillet's face keys are identical before and after the rename.

The existing typed lifecycle test also covers the first-feature rename
and duplicate-name refusal. A rename that cannot reuse a safe prefix
still reaches the existing full refresh.

All nine Fillet replay consumer tests, the typed lifecycle test and all
77 automation library tests passed. Strict Clippy passed for the automation
and desktop targets. The pinned [Feature Properties scenario](pinned/cucumber.json)
passed 16 steps on Linux X11 with Xvfb, Openbox and Vulkan Lavapipe. The
Model Browser feature stayed at SHA-256
`603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
