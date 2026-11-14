@application @ribbon @docking
Feature: Ribbon docking
  The MCAD Software allows the ribbon to be positioned around the modeling workspace.

  @needs_review
  Scenario: Inspect available ribbon docking positions
    Given a part is active in the MCAD Software
    And the ribbon is docked at the top
    When I open the ribbon context menu
    And I open Docking Positions
    Then the menu offers Top, Left and Right
    And Top is checked

  @needs_review
  Scenario: Move the ribbon from the top to both sides and back
    Given a part is active in the MCAD Software
    And the model browser is docked on the left
    And the ribbon is docked at the top
    When I choose Left from the ribbon Docking Positions menu
    Then the ribbon tabs are arranged vertically along the left edge
    And the 3D Model command groups are stacked beside the tabs
    And the model browser appears to the right of the ribbon
    When I open the ribbon Docking Positions menu
    Then Left is checked
    When I choose Right from the ribbon Docking Positions menu
    Then the ribbon tabs and command groups are arranged vertically on the right
    And the model browser is visible on the left of the viewport
    When I open the ribbon Docking Positions menu
    Then Right is checked
    When I choose Top from the ribbon Docking Positions menu
    Then the ribbon tabs and command groups are arranged horizontally above the viewport
    And the same part remains active

  @needs_review
  Scenario: Undock the ribbon and return it to the top
    Given a part is active in the MCAD Software
    And the ribbon is docked at the top
    When I open the ribbon context menu
    And I choose Undock Ribbon
    Then the ribbon floats over the workspace in a vertical panel
    And its tabs and 3D Model group buttons remain visible
    When I open the floating ribbon context menu
    Then Undock Ribbon is unavailable
    When I choose Top from Docking Positions
    Then the ribbon returns to a horizontal position above the viewport
    And the same part remains active

  @needs_review
  Scenario: Open a command group in the floating ribbon
    Given a part is active in the MCAD Software
    And the ribbon is floating with the 3D Model tab selected
    When I select the Create group button
    Then a temporary panel opens beside the ribbon
    And the panel displays Create commands including Extrude and Revolve
    When I click an empty area of the viewport outside the panel
    Then the temporary panel closes
    And the ribbon remains floating

  @needs_review
  Scenario: Switch command groups in the left-docked ribbon
    Given a part is active in the MCAD Software
    And the ribbon is docked on the left with the 3D Model tab selected
    When I select the Create group button
    Then a temporary Create command panel opens beside the ribbon
    And the panel includes Extrude and Revolve
    When I select the Modify group button
    Then the Create panel closes
    And a temporary Modify command panel opens beside the ribbon
    And the panel includes Hole and Fillet
    When I click an empty area of the viewport
    Then the Modify panel closes
    And the ribbon remains docked on the left

  @needs_review
  Scenario: Scroll a floating ribbon to access a lower group
    Given a part is active in the MCAD Software
    And the floating ribbon shows the 3D Model groups at the top of its scroll area
    And the Work Features group is partially visible at the bottom
    When I scroll down in the floating ribbon group area
    Then the Work Features group becomes fully visible
    When I select the Work Features group
    Then a temporary panel shows Plane, Axis and Point commands
    When I click an empty area of the viewport
    Then the temporary panel closes
    And the ribbon remains floating

  @needs_review
  Scenario: Switch tabs with a floating command panel open
    Given a part is active in the MCAD Software
    And the ribbon is floating with the 3D Model tab selected
    When I select the Annotate tab on the floating ribbon
    Then the ribbon displays annotation groups including Notes
    When I select the Notes group
    Then a temporary panel displays note commands
    When I select the 3D Model tab
    Then the Notes panel closes
    And the floating ribbon displays its 3D Model groups again
    And the part remains active

  @needs_review
  Scenario: Keep the ribbon docked while switching document types
    Given a part, an assembly and a sheet-metal part are open in the MCAD Software
    And the part is active with the ribbon docked on the left
    When I activate the assembly document
    Then the ribbon remains docked on the left
    And it displays assembly tabs and commands
    When I activate the sheet-metal part
    Then the ribbon remains docked on the left
    And it displays the Sheet Metal tab and its commands
    And the sheet-metal part is active
