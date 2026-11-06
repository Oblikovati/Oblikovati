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
