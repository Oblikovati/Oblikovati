# Favorites and search in the browser dock

The [pinned reproduction](before/pinned/cucumber.json) stopped when the
test searched the rendered text for a plus sign. The add control is
drawn as an icon. The updated harness locates its dock tab item, confirms
the pointer target, and clicks it through the native window. The
published pane case keeps its named saved Part active.

| Native scenario | Steps passed | Receipt |
| --- | ---: | --- |
| Open and closes an additional browser pane, pinned | 12 | [Run](after/pinned/open-close/cucumber.json) |
| Open and closes an additional browser pane, published | 13 | [Run](after/published/open-close/cucumber.json) |
| Change the Favorites pane view mode, pinned | 14 | [Run](after/pinned/view-mode/cucumber.json) |
| Search the Model tree from another active dock tab, published | 11 | [Run](after/workspace-search/cucumber.json) |
| Close and restore the model browser while editing a sketch, pinned | 11 | [Run](after/workspace-launcher/cucumber.json) |

The search case uses the same dock hit test for the search icon and gives
the Model tab time to receive focus before typing. Its
[snapshot](workspace-search-snapshot.json) includes the named point
fixture. The [Model Browser snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the saved block used by the published Favorites case.
The sketch case verifies that the standalone plus button restores Model
when the dock tab bar is gone.

The runs used application base `7b9c015dc` with the local harness changes
and existing file-browser edits, X11, Xvfb, Openbox, and Vulkan Lavapipe.
Both pinned Model Browser scenarios gained R0 records.
See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
