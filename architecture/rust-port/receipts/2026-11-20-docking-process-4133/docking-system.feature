@application @workspace @cross_cutting @docking @implementation_gap
Feature: Dockable workspace panels
  The MCAD Software lets users arrange workspace panels around the active document.

  @wireframe_gap @expected_contract
  Scenario: Dock a floating panel to an edge with a visible target preview
    Given a part document is active in the MCAD Software
    And a dockable panel is floating over the workspace
    When the user drags the panel title bar toward the left workspace edge
    Then the left docking target is indicated before the panel is released
    When the user releases the panel over that target
    Then the panel docks along the left edge of the workspace
    And the viewport resizes to occupy the remaining workspace
    And the active part remains visible

  @wireframe_gap @expected_contract
  Scenario: Dock a panel to each supported workspace edge
    Given a part document is active in the MCAD Software
    And a dockable panel is floating over the workspace
    When the user docks the panel to the right workspace edge
    Then the panel is attached along the right edge
    And the viewport occupies the workspace to its left
    When the user docks the panel to the top workspace edge
    Then the panel is attached below the ribbon
    And the viewport occupies the workspace below the panel
    When the user docks the panel to the bottom workspace edge
    Then the panel is attached above the document tabs and status area
    And the viewport occupies the workspace above the panel

  @wireframe_gap @expected_contract
  Scenario: Resize a docked panel from its shared boundary
    Given a part document is active in the MCAD Software
    And a dockable panel is docked beside the viewport
    When the user drags the panel boundary away from the viewport
    Then the panel becomes wider
    And the viewport becomes narrower by the corresponding amount
    And the panel contents and active part remain visible
    When the user drags the boundary back
    Then the panel and viewport return to their previous proportions

  @observed_live @fixture_specific
  Scenario: Dock and resize Appearance Browser on the right
    Given a part document is active in the MCAD Software
    And Appearance Browser is floating over the workspace
    When the user drags the Appearance Browser title bar to the far right workspace edge
    Then Appearance Browser docks along the right edge
    And the Model Browser remains visible on the left
    And the viewport occupies the space between both browser panels
    And the active part document remains open
    When the user drags the shared boundary toward the viewport to widen Appearance Browser
    Then Appearance Browser becomes wider
    And the viewport becomes narrower
    And the view orientation control remains visible in the viewport
    When the user restores the boundary to its earlier position
    Then the docked panel and viewport return to their earlier proportions

  @observed_live @fixture_specific
  Scenario: Tab, switch, close, and restore docked browser panels
    Given a part document is active in the MCAD Software
    And Appearance Browser is docked on the right
    And Material Browser is floating over the workspace
    When the user drags the Material Browser title bar onto the Appearance Browser tab strip
    Then both browsers occupy one right-side dock region as tabs
    And one browser is active while the other remains available
    And the viewport remains visible to the left of the dock region
    When the user selects the Appearance Browser tab
    Then Appearance Browser contents become visible in the dock region
    When the user selects the Material Browser tab
    Then Material Browser contents become visible in the dock region
    When the user selects Appearance Browser and closes it from its panel controls
    Then Appearance Browser is removed from the tab group
    And Material Browser remains visible in the right dock region
    When the user opens View and then User Interface
    And the user selects Appearance Browser
    Then Appearance Browser returns as a tab in the right dock region
    And its contents are visible

  @wireframe_gap @expected_contract
  Scenario: Join panels in one dock region as selectable tabs
    Given a part document is active in the MCAD Software
    And two dockable panels are open in separate workspace regions
    When the user drags one panel onto the center of the other panel's dock region
    Then the destination region indicates a tab insertion target
    When the user releases the panel over that target
    Then both panels occupy the same dock region as tabs
    And one panel is active while the other remains available
    When the user selects the other panel tab
    Then its contents replace the active panel contents in that region
    And the viewport and active document remain available

  @wireframe_gap @expected_contract
  Scenario: Float a docked panel and dock it again
    Given a part document is active in the MCAD Software
    And a dockable panel is docked beside the viewport
    When the user drags its title bar away from the dock targets
    Then the panel floats above the workspace
    And the viewport expands into the space released by the panel
    When the user drags the floating panel to a workspace dock target
    Then the panel docks at that target
    And the viewport adjusts to the resulting workspace layout

  @wireframe_gap @expected_contract
  Scenario: Close a docked panel and restore it from the interface menu
    Given a part document is active in the MCAD Software
    And a dockable panel is the active tab in a dock region
    When the user closes the panel
    Then the panel is removed from the visible workspace
    And the remaining dock regions and viewport use the freed space
    When the user opens the interface visibility menu
    And the user selects the closed panel
    Then the panel is restored to the workspace
    And its contents are visible

  @wireframe_gap @expected_contract @needs_live_capture
  Scenario: Retain a panel layout while switching active documents
    Given two documents are open in the MCAD Software
    And a dockable panel has been moved, resized, or tabbed in the workspace
    When the user switches to the other document
    Then the docked panel arrangement remains in place
    And the active document changes in the viewport
    When the user returns to the first document
    Then the panel arrangement remains available

  @wireframe_gap @expected_contract @needs_live_capture
  Scenario: Restore the saved panel arrangement after restarting the application
    Given the user has changed the workspace panel arrangement
    And the MCAD Software has saved that arrangement
    When the user closes and restarts the MCAD Software
    Then the saved panel positions, sizes and tab groups are restored
