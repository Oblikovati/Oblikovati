@application @workspace
Feature: Workspace layout and panel visibility
  The MCAD Software presents document content and navigation controls in a shared workspace.

  @needs_review
  Scenario: Switch the ribbon to the View tab
    Given a part document is open in the MCAD Software
    And the 3D Model ribbon tab is active
    When I select the View ribbon tab
    Then the ribbon displays view, appearance, window and navigation controls
    And the active document remains displayed

  @needs_review
  Scenario: Hide and restore the model browser
    Given a part document is open in the MCAD Software
    And the model browser is visible at the left of the workspace
    When I open the User Interface menu from the View ribbon tab
    And I clear the Model Browser option
    Then the model browser is hidden
    And the drawing area extends into the space it occupied
    And the Model Browser option is unchecked
    When I select the Model Browser option again
    Then the model browser returns to the left of the workspace
    And the Model Browser option is checked

  @needs_review
  Scenario: Float and redock the model browser
    Given a part document is open in the MCAD Software
    And the model browser is docked to the left edge of the workspace
    When I drag the model browser by its header into the viewport
    Then the model browser becomes a floating panel over the viewport
    And the viewport occupies the space previously used by the docked browser
    When I drag the floating browser by its header to the left edge of the workspace
    Then the model browser docks at the left edge
    And the viewport occupies the remaining workspace

  @needs_review
  Scenario: Move the model browser between side docks
    Given a part document is open in the MCAD Software
    And the model browser is docked to the left edge of the workspace
    When I drag its header to the right edge of the workspace
    Then the model browser docks at the right edge
    And the viewport occupies the workspace to its left
    When I drag its header to the left edge of the workspace
    Then the model browser docks at the left edge
