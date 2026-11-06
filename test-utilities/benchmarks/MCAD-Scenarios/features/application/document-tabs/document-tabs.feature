@application @documents
Feature: Open document tabs

  @needs_review @needs_fixture
  Scenario: Switch between open parts
    Given the MCAD Software has two open part documents
    And the first part contains an empty sketch
    And the second part contains a rectangular sketch profile
    And the first part is active
    When I select the second part's document tab
    Then the second part becomes active
    And its rectangular profile appears in the viewport
    And the model browser shows the second part
    When I select the first part's document tab
    Then the first part becomes active
    And its model browser and viewport are displayed
    And the second part remains available as an open tab

  @needs_review
  Scenario: Inspect a document tab's context menu
    Given a part document is active in the MCAD Software
    When I open the document tab's context menu
    Then the menu offers Close, Close Tabs, Close All, Save and Properties
    And it displays shortcuts for Close and Save
    When I dismiss the menu
    Then the document remains active

  @needs_review
  Scenario: Hide and restore the document tab strip
    Given multiple documents are open in the MCAD Software
    And document tabs are visible
    When I clear Document Tabs in the View tab's User Interface menu
    Then the document tab strip disappears
    And the active document remains displayed
    When I select Document Tabs in the same menu
    Then the document tab strip reappears
    And the open documents are available as tabs

  @needs_review
  Scenario: Open the extended window list when tabs are hidden
    Given multiple documents are open in the MCAD Software
    And document tabs are hidden
    When I open the Switch menu on the View tab
    Then the menu lists open windows and offers More Windows
    When I choose More Windows
    Then a Select Window dialog displays a scrollable list of open documents
    And the dialog offers OK and Cancel
