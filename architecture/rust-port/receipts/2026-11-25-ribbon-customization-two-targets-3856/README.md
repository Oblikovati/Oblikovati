# Native Tools and Manage drafts

The first case passed sixteen steps. Staging 2D Sketches on Tools left
it available on Manage. Both targets then held their own staged command
and pending marker. Returning to Tools preserved its draft.
[Independent drafts](case-066/cucumber.json)

The second case passed nineteen steps. It staged 2D Sketches on Tools
and 3D Annotations on Manage, then canceled. After reopening Customize,
both targets were empty, both commands were available, and the target
selector had no pending marker. [Discard and reopen](case-103/cucumber.json)

The isolated inputs matched the published fixture and profile. The
build used `e3c89c815` with this commit's scenario bindings and existing
local GUI filesystem, opening, and saving edits. The runs used Linux
X11 with Xvfb, Openbox, and Vulkan Lavapipe.
[Input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)
