@application @lifecycle
Feature: Close the application

  Scenario: Cancel closing the application when a document has unsaved changes
    Given the MCAD Software has an open part document with unsaved changes
    When the user selects the application window's Close control
    Then a save-changes prompt identifies the part document
    And the prompt offers Yes, No and Cancel
    When the user chooses Cancel
    Then the save-changes prompt closes
    And the part document remains open
    And the MCAD Software remains open

  @needs_fixture @needs_capture
  Scenario: Close the application when the open document has no unsaved changes
    Given the MCAD Software has an open saved document with no unsaved changes
    When the user selects the application window's Close control
    Then the MCAD Software closes without a save-changes prompt

  @needs_capture
  Scenario: Close the application when no document is open
    Given the MCAD Software is open with no document open
    When the user selects the application window's Close control
    Then the MCAD Software closes without a save-changes prompt
