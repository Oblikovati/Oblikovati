@application @ribbon
Feature: Ribbon panel visibility
  The MCAD Software allows individual ribbon panels to be shown or hidden.

  @needs_review
  Scenario: Show and hide a ribbon panel
    Given a part is active in the MCAD Software
    And the 3D Model ribbon is expanded
    And the Parameters panel is hidden
    When I open the ribbon context menu and expand Show Panels
    Then Parameters is unchecked
    And visible panels including Create and Modify are checked
    When I select Parameters in Show Panels
    Then the Parameters panel appears in the ribbon
    When I reopen Show Panels
    Then Parameters is checked
    When I select Parameters again
    Then the Parameters panel disappears from the ribbon
    And the part remains active

  @needs_review
  Scenario: Retain panel visibility while switching tabs and parts
    Given two part documents are open in the MCAD Software
    And the first part is active with the 3D Model ribbon expanded
    And the Parameters panel is visible
    When I select the View ribbon tab
    And I return to the 3D Model ribbon tab
    Then the Parameters panel is still visible
    When I activate the second part document
    Then its 3D Model ribbon also shows the Parameters panel
    When I return to the first part document
    Then its Parameters panel remains visible

  @needs_review
  Scenario: Hide and restore a View ribbon panel
    Given a part is active in the MCAD Software
    And the View ribbon is expanded with the Navigate panel visible
    When I open the View ribbon context menu and expand Show Panels
    Then Visibility, Appearance, Windows and Navigate are checked
    When I select Navigate in Show Panels
    Then the Navigate panel disappears from the View ribbon
    When I switch to 3D Model and return to View
    Then the Navigate panel remains hidden
    When I reopen Show Panels on the View ribbon
    Then Navigate is unchecked
    When I select Navigate in Show Panels
    Then the Navigate panel reappears on the View ribbon

  @needs_review
  Scenario: Hide a panel while the ribbon shows compact group buttons
    Given a part is active in the MCAD Software
    And the 3D Model ribbon shows compact group buttons
    And the Parameters group button is visible
    When I open the compact ribbon context menu and expand Show Panels
    Then Parameters is checked
    When I select Parameters in Show Panels
    Then the Parameters group button disappears
    And the other compact group buttons remain visible
    When I return the ribbon to its expanded display
    Then the Parameters panel is absent from the expanded ribbon

  @needs_review
  Scenario: Hide and restore a panel on the side-docked ribbon
    Given a part is active in the MCAD Software
    And the 3D Model ribbon is docked on the left
    And the Modify group is visible
    When I open the ribbon context menu and expand Show Panels
    Then Modify is checked
    When I select Modify in Show Panels
    Then the Modify group disappears from the side-docked ribbon
    And the remaining ribbon groups reflow vertically
    When I reopen Show Panels
    Then Modify is unchecked
    When I select Modify in Show Panels
    Then the Modify group returns to the side-docked ribbon

  @needs_review
  Scenario: Preserve a hidden panel when changing ribbon docking position
    Given a part is active in the MCAD Software
    And the 3D Model ribbon is docked at the top with Modify visible
    When I hide Modify through Show Panels
    Then the Modify panel disappears from the ribbon
    When I dock the ribbon on the left
    Then the ribbon is docked on the left
    And the Modify group remains hidden
    When I open Show Panels on the left-docked ribbon
    Then Modify is unchecked
    When I select Modify
    Then the Modify group returns to the left-docked ribbon

  @needs_review
  Scenario: Restore a hidden panel in the undocked ribbon
    Given a part is active in the MCAD Software
    And the 3D Model ribbon is docked at the top with Modify visible
    When I hide Modify through Show Panels
    And I undock the ribbon
    Then the floating ribbon does not show the Modify group
    When I open Show Panels on the floating ribbon
    Then Modify is unchecked
    When I select Modify
    Then the Modify group appears in the floating ribbon
    When I dock the ribbon at the top
    Then the Modify panel is visible in the top ribbon

  @needs_review
  Scenario: Keep a panel hidden when redocking the floating ribbon
    Given a part is active in the MCAD Software
    And the floating 3D Model ribbon shows the Modify group
    When I hide Modify through Show Panels
    Then the floating ribbon no longer shows the Modify group
    When I dock the ribbon at the top
    Then the top ribbon does not show the Modify panel
    When I open Show Panels on the top ribbon
    Then Modify is unchecked
    When I select Modify
    Then the Modify panel is visible again

  @needs_review
  Scenario: Set visibility of two ribbon panels independently
    Given a part is active in the MCAD Software
    And the expanded 3D Model ribbon shows Modify and Pattern
    When I hide Modify through Show Panels
    And I hide Pattern through Show Panels
    Then neither Modify nor Pattern is visible in the ribbon
    When I reopen Show Panels
    Then Modify and Pattern are both unchecked
    When I select Modify
    Then Modify appears in the ribbon
    And Pattern remains hidden
    When I reopen Show Panels
    Then Modify is checked and Pattern is unchecked
    When I select Pattern
    Then both panels are visible in the ribbon

  @needs_review
  Scenario: Retain separate panel choices on different ribbon tabs
    Given a part is active in the MCAD Software
    And Modify is visible on the 3D Model ribbon
    And Navigate is visible on the View ribbon
    When I hide Modify on 3D Model through Show Panels
    And I switch to View and hide Navigate through Show Panels
    And I return to 3D Model
    Then Modify remains hidden
    When I restore Modify through Show Panels
    And I switch to View
    Then Navigate remains hidden
    And Navigate is unchecked in Show Panels
    When I select Navigate
    Then the Navigate panel appears on the View ribbon

  @needs_review
  Scenario: Dismiss the panel list without changing visibility
    Given a part is active in the MCAD Software
    And the 3D Model ribbon shows Create and Modify
    And Parameters is hidden
    When I open the ribbon context menu and expand Show Panels
    Then Create and Modify are checked and Parameters is unchecked
    When I press Escape
    Then the Show Panels submenu closes
    And the ribbon context menu remains open
    When I press Escape again
    Then the ribbon context menu closes
    And Create and Modify remain visible
    And Parameters remains hidden
