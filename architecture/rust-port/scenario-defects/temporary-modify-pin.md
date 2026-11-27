# Temporary Modify expansion has contradictory source expectations

The pinned and published versions of **Minimize to tabs after pinning the
Modify expansion** have the same actions. Both pin Modify in the full
ribbon, switch to tabs-only mode, reopen the temporary 3D Model ribbon and
Modify, and click empty viewport space.

| Source | Expected after the click | Native result |
| --- | --- | --- |
| Pinned `b576445b98cffd686dcc05aa8c017de2a591c8c8`, line 534 | Temporary ribbon and pinned Modify remain visible | [11 steps passed, then the assertion failed](../receipts/2026-11-24-ribbon-modify-pin-source-3856/pinned/cucumber.json) |
| Published snapshot `dfbe3564d8e78386834e955d833ad0f6d5c06ea5`, line 535 | Temporary ribbon and Modify close | [All 19 steps passed](../receipts/2026-11-24-ribbon-modify-pin-source-3856/published/cucumber.json) |

The focused runs used the same Head binary on Linux X11 with Xvfb, Openbox
and Vulkan Lavapipe. The build used `7de9a444a`, the exact pinned assertion,
and the existing local file-browser edits. The
[pinned capture](../receipts/2026-11-24-ribbon-modify-pin-source-3856/pinned/capture.png)
shows the closed temporary ribbon. The
[published input manifest](../receipts/2026-11-24-ribbon-expansion-full-3856/snapshot.json)
records the revised feature and supporting files.

The pinned declaration is blocked pending reconciliation with the
published source. See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
