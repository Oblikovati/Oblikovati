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

  @needs_review
  Scenario: Resize the docked model browser
    Given a part document is open in the MCAD Software
    And the model browser is docked to the left of the viewport
    When I drag the boundary between the model browser and viewport to the right
    Then the model browser becomes wider
    And the viewport becomes narrower
    And the document and model tree remain displayed
    When I drag the boundary back to its original position
    Then the model browser and viewport return to their previous widths

  @needs_review @needs_fixture
  Scenario: Collapse and expand model browser branches
    Given a part with nested model browser branches is open in the MCAD Software
    And some browser branches are expanded
    When I open the model browser menu
    And I choose Collapse All
    Then the expanded branches collapse to their parent rows
    And the model remains displayed in the viewport
    When I open the model browser menu
    And I choose Expand All
    Then nested browser rows are displayed
    And the model remains displayed in the viewport

  @needs_review @needs_fixture
  Scenario: Filter model browser entries by name
    Given a part with multiple work points and other features is open in the MCAD Software
    And the model browser is visible
    When I open the browser search field
    And I enter "Work Point"
    Then matching work point rows are shown in the browser
    And the matching text is highlighted in those rows
    And unrelated visible feature rows are filtered from the browser
    And the model remains displayed in the viewport
    When I clear the search text
    Then the unfiltered browser tree is shown again
    When I close the empty search field
    Then the browser search field is hidden

  @needs_review @needs_fixture
  Scenario: Show an empty browser search result
    Given a populated part is open in the MCAD Software
    And the model browser search field is open
    When I enter a query that matches no browser entry
    Then the browser shows a no-results message instead of feature rows
    And the model remains displayed in the viewport
    When I clear the search query
    Then the unfiltered model browser tree is shown again

  @needs_review
  Scenario: Float and redock the drawing browser
    Given a drawing document is active in the MCAD Software
    And its browser is docked to the left of the drawing
    When I drag the browser header into the drawing area
    Then the browser floats over the drawing sheet
    And the browser displays drawing resources and the sheet tree
    And the drawing remains visible behind the browser
    When I drag the floating browser header to the left edge of the workspace
    Then the browser docks at the left edge
    And the drawing remains visible in the remaining workspace

  @needs_review
  Scenario: Close and restore the drawing browser from its tab area
    Given a drawing is active in the MCAD Software
    And its model browser is docked on the left
    When I select the browser tab's close control
    Then the browser tree is hidden
    And the drawing occupies the space previously used by the browser
    And a plus control remains at the upper left of the workspace
    When I open the plus control
    Then the panel list offers Model Browser
    When I select Model Browser
    Then the browser returns to the left
    And it displays the drawing resources and sheet tree

  @needs_review
  Scenario: Add and switch between browser panel tabs
    Given a drawing is active in the MCAD Software
    And its Model browser is docked on the left
    And an additional log panel is available in the browser's plus menu
    When I add the log panel from the plus menu
    Then the log panel becomes active in the same dock
    And the Model browser remains available as a tab
    When I select the Model browser tab
    Then the drawing resources and sheet tree are displayed
    When I select the log panel tab
    Then the log area is displayed in the dock
    When I close the log panel tab
    Then the Model browser is displayed
    And the drawing remains visible

  @needs_review
  Scenario: Collapse and expand the drawing browser tree
    Given a drawing with drawing resources and a sheet is open in the MCAD Software
    And the browser has expanded branches containing nested entries
    When I open the drawing browser menu
    And I choose Collapse All
    Then the drawing resources and sheet remain visible as collapsed branches
    And their nested entries are hidden
    And the drawing sheet remains visible
    When I open the drawing browser menu
    And I choose Expand All
    Then nested entries under the drawing resources and sheet are displayed
    And the drawing sheet remains visible

  @needs_review @needs_fixture
  Scenario: Filter drawing browser entries while retaining their parent branches
    Given a drawing with border entries in its resources and sheet is open in the MCAD Software
    When I open the drawing browser search field
    And I enter "Border"
    Then matching border entries are displayed beneath their respective parent branches
    And matching text is highlighted
    And unrelated browser entries are hidden
    And the drawing sheet remains visible
    When I clear the search text
    Then the unfiltered drawing browser tree is displayed
    When I close the empty search field
    Then the search field is hidden

  @needs_review @needs_fixture
  Scenario: Show an empty drawing browser search result
    Given a drawing with browser entries is open in the MCAD Software
    And the drawing browser search field is open
    When I enter a query that matches no drawing browser entry
    Then a no-results message is displayed in place of the browser tree
    And the drawing sheet remains visible
    When I clear the search text
    Then the unfiltered drawing browser tree is displayed

  @needs_review @needs_fixture
  Scenario: Switch the selected drawing browser entry
    Given a drawing with a border and title block is open in the MCAD Software
    And the sheet branch is expanded in the drawing browser
    When I select the sheet border entry in the drawing browser
    Then the border entry is highlighted in the browser
    And the drawing sheet remains displayed
    When I select the sheet title block entry in the drawing browser
    Then the title block entry is highlighted in the browser
    And the border entry is no longer highlighted as the active selection
    And the drawing sheet remains displayed

  @needs_review @needs_fixture
  Scenario: Search while the drawing browser floats
    Given a drawing with browser entries is open in the MCAD Software
    And the drawing browser floats over the drawing sheet
    When I open the floating browser search field
    And I enter a query that matches no drawing browser entry
    Then the floating browser displays a no-results message instead of its tree
    And the drawing sheet remains visible behind the browser
    When I clear the search text
    Then the unfiltered drawing browser tree is displayed in the floating panel
    When I close the empty search field
    Then the search field is hidden
    And the browser remains floating over the sheet

  @needs_review
  Scenario: Move the drawing browser between side docks
    Given a drawing is active in the MCAD Software
    And the drawing browser is docked on the left
    When I drag the drawing browser header to the right edge of the workspace
    Then the drawing browser docks on the right
    And its drawing resources and sheet tree remain visible
    And the drawing sheet remains visible to its left
    When I drag the drawing browser header to the left edge of the workspace
    Then the drawing browser docks on the left
    And the drawing sheet remains visible to its right

  @needs_review @needs_fixture
  Scenario: Collapse and reopen one drawing browser branch
    Given a drawing with a sheet and drawing resources is open in the MCAD Software
    And the sheet and drawing resources branches are expanded in the browser
    When I select the expansion control beside the sheet branch
    Then the sheet branch collapses and its child entries are hidden
    And the drawing resources branch remains expanded
    And the drawing sheet remains visible
    When I select the expansion control beside the sheet branch again
    Then the sheet branch expands and its child entries are displayed
    And the drawing resources branch remains expanded

  @needs_review @needs_fixture
  Scenario: Find drawing entries inside a collapsed resource branch
    Given a drawing with a border resource is open in the MCAD Software
    And the drawing resources branch is collapsed in the browser
    When I open the drawing browser search field
    And I enter "Border"
    Then the filtered tree shows the matching border resource under its parent branches
    And the matching text is highlighted
    When I clear the search text
    Then the unfiltered tree is restored
    And the drawing resources branch is collapsed
    And the drawing sheet remains visible
