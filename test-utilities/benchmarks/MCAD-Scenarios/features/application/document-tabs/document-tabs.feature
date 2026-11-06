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

  @needs_review
  Scenario: Switch to another open document with the keyboard
    Given multiple part documents are open in the MCAD Software
    And one part document is active
    When I press Control+Tab
    Then another open part document becomes active
    And its model browser and viewport are displayed
    And the previously active part remains open

  @needs_review @needs_fixture
  Scenario: Inspect properties of an inactive document from its tab
    Given the MCAD Software has two open part documents
    And the first part is active
    And the second part has been saved
    When I open the second part's document tab context menu
    Then the first part remains active in the workspace
    When I choose Properties from the context menu
    Then the properties dialog identifies the second part
    And its General page displays the file name, location, size and timestamps
    When I close the properties dialog without making changes
    Then the first part remains active in the workspace
    And the second part remains open

  @needs_review @needs_fixture
  Scenario: Close a saved inactive document with its tab close button
    Given the MCAD Software has two open part documents
    And the first part is active
    And the second part is saved and has no unsaved changes
    When I click the close button on the second part's document tab
    Then the second part's tab disappears
    And the first part remains active in the workspace
    And no save prompt is displayed

  @needs_review @needs_fixture
  Scenario: Close a newly created empty active part
    Given a part document is active in the MCAD Software
    And other documents are also open
    When I create a new empty part from a standard metric template
    Then the new part becomes active and has its own document tab
    When I click the close button on the new part's tab without editing it
    Then the new part's tab disappears without a save prompt
    And the previously active part is displayed again
    And its model browser and viewport are restored

  @needs_review @needs_fixture
  Scenario: Cancel closing a part while editing its sketch
    Given a new part is open in the MCAD Software
    And I am editing a sketch in the part
    And the environment-exit message is enabled
    When I click the close button on the part's document tab
    Then a dialog asks whether to exit the sketch environment and continue
    And the dialog offers OK and Cancel
    When I choose Cancel
    Then the part remains open
    And the sketch editing environment remains active

  @needs_review @needs_fixture
  Scenario: Cancel the save prompt after leaving sketch editing to close
    Given a new part contains an unsaved sketch in the MCAD Software
    And I am editing that sketch
    And the environment-exit message is enabled
    When I click the close button on the part's document tab
    And I choose OK to exit the sketch environment and continue
    Then a save-changes prompt identifies the part
    And the prompt offers Yes, No and Cancel
    When I choose Cancel at the save-changes prompt
    Then the part remains open
    And its sketch remains listed in the model browser
    And the part modeling environment is active

  @needs_review @needs_fixture
  Scenario: Cancel the first save while closing an unsaved part
    Given a new part with an unsaved sketch is active in the MCAD Software
    And the part is in the modeling environment
    When I click the close button on the part's document tab
    Then a save-changes prompt identifies the part
    When I choose Yes to save changes
    Then a Save As dialog opens with a suggested file name
    When I cancel the Save As dialog
    Then the part remains open and active
    And its sketch remains listed in the model browser

  @needs_review @needs_fixture
  Scenario: Save a new part while closing its tab
    Given a new part with an unsaved sketch is active in the MCAD Software
    And another part was active before the new part was created
    When I click the close button on the new part's document tab
    And I choose Yes in the save-changes prompt
    And I accept the suggested file name in the Save As dialog
    Then the part file exists at the selected location
    And the new part's tab disappears
    And the previously active part is displayed
    When I reopen the saved part file
    Then the reopened part is active
    And its sketch is listed in the model browser

  @needs_review @needs_fixture
  Scenario: Close an inactive saved document from its tab menu
    Given multiple part documents are open in the MCAD Software
    And the first part is active
    And the second part is saved and has no unsaved changes
    When I open the second part's document tab context menu
    And I choose Close
    Then the second part's tab disappears
    And the first part remains active in the workspace
    And no save prompt is displayed

  @needs_review @needs_fixture
  Scenario: Switch between a drawing and a part being edited
    Given a drawing and a part are open in the MCAD Software
    And the part has an active sketch editing environment
    And the drawing is active with its drawing resources branch collapsed
    When I select the part document tab
    Then the part sketch viewport and model browser are displayed
    And the sketch ribbon is active
    When I select the drawing document tab
    Then the drawing sheet and drawing browser are displayed
    And the drawing ribbon is active
    And the drawing resources branch is collapsed
    And the part remains available as an open tab
    When I select the part document tab again
    Then the part sketch viewport and model browser are displayed
    And the sketch editing environment remains active

  @needs_review @needs_fixture
  Scenario: Activate a document from the tab-strip window list
    Given a drawing and a part are open in the MCAD Software
    And the part is active
    And document tabs are visible
    When I open the document list at the end of the tab strip
    Then the list contains the open part and drawing
    And the active part is marked in the list
    And the menu offers Arrange, Horizontal Tile, Vertical Tile and Restore All
    And the menu offers Close Tabs and Close All
    When I select the drawing from the list
    Then the drawing becomes active
    And its sheet and drawing browser are displayed
    And the part remains open

  @needs_review @needs_fixture
  Scenario: Reorder an active drawing tab
    Given a drawing and several other documents are open in the MCAD Software
    And the drawing is active and its tab is visible
    When I drag the drawing tab left across several neighboring tabs
    Then the drawing tab appears at its new position among the document tabs
    And the drawing remains active
    And its sheet and browser remain displayed
    When I drag the drawing tab back to its earlier position
    Then the drawing tab returns to its earlier position
    And the drawing remains active

  @needs_review @needs_fixture
  Scenario: Reorder an inactive part tab
    Given a drawing and several parts are open in the MCAD Software
    And the drawing is active
    And an inactive part tab is visible
    When I drag the inactive part tab left across several neighboring tabs
    Then the part tab appears at its new position among the document tabs
    And the part becomes active
    And the part viewport and model browser are displayed
    And the drawing remains open

  @needs_review @needs_fixture
  Scenario: Detach an active drawing from the tab strip
    Given a drawing and a part are open in the MCAD Software
    And the drawing is active in the main document tab strip
    When I drag the drawing tab from the strip into the workspace
    Then the drawing is displayed in a separate window with its own document tab
    And the drawing sheet remains visible in that window
    And the part remains open in the main workspace
    When I select another open part tab in the main workspace
    Then that part becomes active in the main workspace
    And the drawing remains open in its separate window

  @needs_review @needs_fixture
  Scenario: Inspect the detached drawing tab context menu
    Given a drawing is open in a separate window in the MCAD Software
    And the drawing has a document tab in that window
    When I open the detached drawing tab context menu
    Then the menu offers Close, Close Tabs, Close All, Save and Properties
    And it displays shortcuts for Close and Save
    When I dismiss the menu
    Then the drawing remains open in its separate window

  @needs_review @needs_fixture
  Scenario: Switch shared controls between a detached drawing and a part
    Given a drawing is open in a separate window in the MCAD Software
    And a part is open in the main workspace
    And the part context is active in the main workspace
    When I click the detached drawing sheet
    Then the main application browser displays the drawing tree
    And the main application ribbon displays drawing-context tabs
    When I select the part tab in the main workspace
    Then the main application browser displays the part tree
    And the main application ribbon displays part-context tabs
    And the detached drawing remains open

  @needs_review @needs_fixture
  Scenario: Switch backward and forward between neighboring part tabs
    Given two part tabs are adjacent in the MCAD Software
    And the part on the right is active
    When I press Control+Shift+Tab
    Then the part on the left becomes active
    And its viewport and model browser are displayed
    When I press Control+Tab
    Then the part on the right becomes active again
    And its viewport and model browser are displayed
