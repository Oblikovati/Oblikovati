# Native File and Home recent-document checks

The Linux desktop ran under Xvfb, Openbox, and Vulkan Lavapipe. The build
included the File and Home changes, the titlebar refactor, and existing
local document-dialog edits.

- All eight File menu scenarios and 67 steps passed, including Home,
  dismissal, recent document controls, and the New and Save As submenus.
  [Cucumber](file-menu-cucumber.json) · [JUnit](file-menu-junit.xml) ·
  [Scenario arguments](file-menu-scenario-arguments.json)
- A long recent filename was ellipsized before its pin control. Clicking
  the pin toggled the original full path. [File menu](File-menu.png)
- The published Home metadata scenario passed all nine steps. It checked
  the default columns, dates, and distinct rendered Part and Drawing icons.
  [Home list](Home-list.png) · [Cucumber](home-cucumber.json) ·
  [JUnit](home-junit.xml) · [Scenario arguments](home-scenario-arguments.json)
- Native input switched between List and Grid and verified that the
  highlighted control tracked the view. [Home grid](Home-grid.png)
