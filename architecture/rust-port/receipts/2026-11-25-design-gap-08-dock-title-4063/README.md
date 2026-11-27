# Model Browser dock title controls

The docked Model Browser uses its title bar for the tree options and search.
The panel launcher stays separate. The tree menu still offers Autoscroll,
Expand All, Collapse All, and Find where applicable. The floating browser
keeps access to its controls.

Native X11 tests opened the menu, exercised Find, focused the tree filter,
and verified that filtering preserved an open Feature Properties editor.
They also verified that both asset-tab bounds stay fixed when the pointer
moves to the inactive tab's close area.

[Model Browser capture](model-dock-title.png)

Three published workspace scenarios passed with 27 steps. They covered
the Model menu, switching from Log directly into the focused tree filter,
and searching a floating Drawing browser.
[Model menu](menu/cucumber.json) · [Dock search](search/cucumber.json) ·
[Floating search](floating/cucumber.json)

The workspace spacing scenario passed all eight steps. It measured the
dock-title inset and checked the ribbon group gaps with their dividers.
[Workspace spacing](spacing/cucumber.json)
