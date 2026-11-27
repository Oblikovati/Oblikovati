# Document Saving after the viewport fix

All 23 pinned scenarios and 218 steps passed in one
[native run](cucumber.json). The run covered the empty Part reopen,
the existing sketch reopen, first-save previews, subfolder navigation,
and file replacement prompts. It includes the
[empty-view regression](../2026-11-24-empty-part-viewport-3888/README.md).

The run used application code at `8deaf75f2` and the clean benchmark pin
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. It ran on X11 under Xvfb with
Vulkan Lavapipe. The working directory was `crates/head`, matching the
Cargo test invocation. The two subfolder scenarios use the crate's
`src` folder as their initial fixture.

See [#3888](https://github.com/Oblikovati/Oblikovati/issues/3888).
