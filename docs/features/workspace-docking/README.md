# Workspace docking

Model, Log, and Favorites are independent dockable ImGui windows. The workspace
dockspace places them in the Model dock node, so Dear ImGui owns tab selection,
tab reordering, dragging, and title-bar close behavior. Material Browser and
Appearance Browser are also independent dockable windows; each shows only its
own asset list, with no nested material or appearance tab row.

Every visible dock tab bar has trailing panel-menu and search buttons appended
through Dear ImGui's dock-node tab-bar API. The menu controls which dockable
panels are open. With a document open, search activates the Model tab and
focuses its tree filter. The saved ImGui
layout restores panel placement and active-tab selection.

The window abstraction follows `Window` → `DockableWindow` →
`TabbedDockableWindow`. The dock registry accepts only tabbed dockable windows,
so a registered window outside the inheritance chain fails to compile.
