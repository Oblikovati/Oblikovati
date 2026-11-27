# Inactive views and inline renaming on the saved blank Part

The initial published runs opened the saved blank metric Part and stopped
at the copy steps for [BrowserView](before/lock/cucumber.json) and
[Capture View](before/rename-cancel/cucumber.json). Both setups now use
the Primary row's Copy command. The inactive copy leaves Primary active.
The rename cases use the active Capture View, adding a sibling named Front
through the same browser command when the scenario calls for it.

| Native scenario | Published | Pinned |
| --- | ---: | ---: |
| Toggle the lock marker on a copied view representation | [21 steps](published/lock/cucumber.json) | [19 steps](pinned/lock/cucumber.json) |
| Toggle camera autosave for an inactive copied view | [19 steps](published/autosave/cucumber.json) | [17 steps](pinned/autosave/cucumber.json) |
| Cancel a duplicate view representation name | [12 steps](published/rename-cancel/cucumber.json) | [10 steps](pinned/rename-cancel/cucumber.json) |
| Correct a duplicate view representation name | [12 steps](published/rename-correct/cucumber.json) | [10 steps](pinned/rename-correct/cucumber.json) |
| Submit an empty view representation name | [10 steps](published/rename-empty/cucumber.json) | [8 steps](pinned/rename-empty/cucumber.json) |
| Cancel an inline view rename with Escape | [10 steps](published/rename-escape/cucumber.json) | [8 steps](pinned/rename-escape/cucumber.json) |
| Use a view name differing only in letter case | [10 steps](published/rename-case/cucumber.json) | [8 steps](pinned/rename-case/cucumber.json) |

The lock marker check samples a single captured frame. All 94 published
and 80 pinned steps passed. Six pinned scenarios gained R0 records. The inventory fell from
[87 unresolved steps](../2026-11-24-browser-solid-view-fixture-3896/published-bindings.json)
to [80](published-bindings.json).

The runs used application base `d9ae91fbd`, the local test changes,
and existing file-browser edits on Linux X11, Xvfb, Openbox, and Vulkan
Lavapipe. The
[published snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the saved blank Part.

See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
