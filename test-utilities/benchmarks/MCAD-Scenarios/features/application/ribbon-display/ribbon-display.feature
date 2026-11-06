@application @ribbon
Feature: Ribbon display modes
  The MCAD Software provides access to grouped commands while allowing the ribbon to occupy less workspace.

  @needs_review
  Scenario: Reduce the full ribbon to compact group buttons
    Given a part document is open in the MCAD Software
    And the ribbon display control is configured to cycle through all modes
    And the View ribbon tab displays its full groups of commands
    When I activate the ribbon display control once
    Then the View ribbon shows compact buttons for Visibility, Appearance, Windows and Navigate
    And the viewport has more vertical space
    When I select the Windows group button
    Then a temporary panel displays that group's commands

  @needs_review
  Scenario: Open a group from the title-only ribbon
    Given the View ribbon displays compact group buttons
    When I activate the ribbon display control once
    Then the View ribbon displays group titles in a narrow strip
    When I select the Windows group title
    Then a temporary panel displays the Windows commands

  @needs_review
  Scenario: Access a ribbon tab in tabs-only mode
    Given the View ribbon displays its group titles
    When I activate the ribbon display control once
    Then the ribbon shows its tabs without the group strip
    When I select the View tab
    Then the View commands appear in a temporary panel below the tabs
    When I activate the ribbon display control again
    And I click in an empty area of the viewport
    Then the expanded View ribbon remains visible

  @needs_review
  Scenario: View available ribbon minimization choices
    Given a part document is open in the MCAD Software
    When I open the ribbon display control menu
    Then the menu offers minimization to tabs, panel titles and panel buttons
    And it offers a cycle-through-all choice

  @needs_review
  Scenario: Choose tabs-only mode and switch ribbon tabs
    Given a part document is open in the MCAD Software
    And the View ribbon is expanded
    When I open the ribbon display control menu
    And I choose Minimize to Tabs
    Then the command groups are hidden and the ribbon tabs remain visible
    When I select the 3D Model tab
    Then its command groups appear temporarily below the tabs
    When I click an empty area of the viewport
    Then the temporary command groups close
    And the 3D Model tab remains selected
    When I open the ribbon display control menu
    Then Minimize to Tabs is checked

  @needs_review
  Scenario: Cycle drawing ribbon display by double-clicking its active tab
    Given a drawing document is active in the MCAD Software
    And the ribbon display control is configured to cycle through all modes
    And the Place Views tab shows its full command groups
    When I double-click the active Place Views tab
    Then the ribbon shows compact Create, Modify, Sketch and Sheets group buttons
    When I double-click the active Place Views tab again
    Then the ribbon shows the group titles in a narrow strip
    When I double-click the active Place Views tab again
    Then the ribbon shows tabs without the group strip
    When I double-click the active Place Views tab again
    Then the full Place Views command groups are visible
    And the drawing document remains active

  @needs_review
  Scenario: Keep compact ribbon display when switching document types
    Given a drawing and a part are open in the MCAD Software
    And the drawing is active with its ribbon in compact group-button mode
    When I activate the part document
    Then the part ribbon displays its command groups as compact buttons
    And those buttons include Sketch, Create and Modify
    When I return to the drawing document
    Then its ribbon displays compact Create, Modify, Sketch and Sheets group buttons

  @needs_review
  Scenario: Switch between temporary compact ribbon groups
    Given a drawing document is active in the MCAD Software
    And the Place Views ribbon shows compact group buttons
    When I select the Create group button
    Then a temporary Create panel displays drawing-view commands
    When I select the Modify group button
    Then the Create panel closes
    And a temporary Modify panel displays modification commands
    When I click an empty area of the drawing outside the panel
    Then the temporary Modify panel closes
    And the compact group buttons remain visible

  @needs_review
  Scenario: Switch ribbon tabs while a compact group panel is open
    Given a drawing is active in the MCAD Software
    And the Place Views ribbon shows compact group buttons
    And the temporary Create group panel is open
    When I select the Annotate tab
    Then the temporary Create panel closes
    And the ribbon displays compact Annotate group buttons
    And those buttons include Dimension, Text and Symbols
    And the drawing remains active

  @needs_review
  Scenario: Choose panel titles as the ribbon minimization mode
    Given a drawing is active in the MCAD Software
    And the Annotate ribbon is expanded
    When I open the ribbon display control menu
    And I choose Minimize to Panel Titles
    Then the Annotate command groups are reduced to a narrow strip of titles
    When I activate the ribbon display control
    Then the full Annotate command groups are visible
    When I activate the ribbon display control again
    Then the ribbon displays the group-title strip
    When I open the ribbon display control menu
    Then Minimize to Panel Titles is checked

  @needs_review
  Scenario: Choose panel buttons as the ribbon minimization mode
    Given a drawing is active in the MCAD Software
    And the Annotate ribbon is expanded
    When I open the ribbon display control menu
    And I choose Minimize to Panel Buttons
    Then the Annotate command groups appear as compact buttons
    When I activate the ribbon display control
    Then the full Annotate command groups are visible
    When I activate the ribbon display control again
    Then the Annotate command groups appear as compact buttons
    When I open the ribbon display control menu
    Then Minimize to Panel Buttons is checked

  @needs_review
  Scenario: Identify the control for restoring the full ribbon
    Given a part is active in the MCAD Software
    And the ribbon displays tabs without permanent command groups
    When the pointer rests over the ribbon display control
    Then a tooltip identifies the action as Show Full Ribbon
    And the part viewport remains visible

  @needs_review
  Scenario: Toggle between tabs-only and full ribbon display
    Given a part is active in the MCAD Software
    And Minimize to Tabs is the selected ribbon display mode
    And the 3D Model ribbon is expanded
    When I activate the ribbon display control
    Then the permanent command groups disappear and the ribbon tabs remain visible
    When I activate the ribbon display control again
    Then the full 3D Model command groups are visible
    When I activate the ribbon display control again
    Then the ribbon returns to tabs-only display
    And the part remains active
