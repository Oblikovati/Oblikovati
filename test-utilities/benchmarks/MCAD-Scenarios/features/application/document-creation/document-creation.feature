@application @documents
Feature: Create a document from a template

  @needs_review
  Scenario: Inspect metric and inch part templates
    Given a document is open in the MCAD Software
    When I press Control+N
    Then the new-document template chooser opens
    And templates are grouped by document type
    When I select the metric template folder
    And I select the standard metric part template
    Then the template details identify a part with millimeter units
    When I select the English template folder
    And I select the standard inch part template
    Then the template details identify a part with inch units

  @needs_review
  Scenario: Cancel the template chooser
    Given a document is open in the MCAD Software
    And the new-document template chooser is open
    When I cancel the chooser
    Then the chooser closes
    And the previously active document remains displayed

  @needs_review
  Scenario: Create a standard metric part
    Given the new-document template chooser is open in the MCAD Software
    When I select the metric template folder
    And I select the standard metric part template
    And I choose Create
    Then a new part document becomes active
    And its viewport contains no visible model geometry
    And its model browser displays an origin and an end-of-part marker
    And a new document tab is visible

  @needs_review
  Scenario: Create a standard metric assembly
    Given the new-document template chooser is open in the MCAD Software
    When I select the metric template folder
    And I select the standard metric assembly template
    Then the template details identify an assembly with millimeter units
    When I choose Create
    Then a new assembly document becomes active
    And its viewport contains no visible components
    And its model browser shows representations, relationships and origin entries
    And the assembly ribbon offers commands to place and create components
    And a new document tab is visible

  @needs_review @needs_fixture
  Scenario: Create a drawing from a sheet template
    Given the new-document template chooser is open in the MCAD Software
    And a drawing template with a border and title block is available
    When I select the drawing template
    And I choose its Use Template preview
    And I choose Create
    Then a new drawing document becomes active
    And its viewport displays a sheet with a border and title block
    And the drawing browser lists the sheet and its border
    And the ribbon offers drawing view commands
    And a new document tab is visible

  @needs_review
  Scenario: Create a metric sheet-metal part
    Given the new-document template chooser is open in the MCAD Software
    When I select the metric template folder
    And I select the metric sheet-metal part template
    Then the template details identify a part with millimeter units
    When I choose Create
    Then a new part document becomes active
    And its viewport contains no visible model geometry
    And the browser shows a folded model and its origin
    And the sheet-metal ribbon is active
    And a new document tab is visible

  @needs_review
  Scenario: Collapse and expand a template group
    Given the new-document template chooser is open in the MCAD Software
    And the metric part template group is expanded
    And a sheet-metal part template is selected
    When I collapse the part template group
    Then its template tiles are hidden
    And the assembly and drawing groups remain visible
    And the selected template's details remain displayed
    When I expand the part template group
    Then its template tiles are visible again
    And the sheet-metal template remains selected

  @needs_review
  Scenario: Change template folders before selecting a new template
    Given the new-document template chooser is open in the MCAD Software
    And the metric sheet-metal template is selected
    And its details show millimeter units
    When I select the English template folder
    Then the gallery shows the English folder's templates
    And the details still identify the previously selected metric sheet-metal template
    When I select the inch sheet-metal template
    Then the template details identify a part with inch units

  @needs_review
  Scenario: Create a presentation and cancel initial model insertion
    Given the new-document template chooser is open in the MCAD Software
    When I select the metric presentation template
    And I choose Create
    Then a new presentation document becomes active
    And an Insert dialog opens for an assembly file
    When I cancel the Insert dialog
    Then the presentation remains open with an empty viewport
    And the browser shows a scene
    And the workspace displays a storyboard timeline and snapshot views panel
    And the presentation ribbon offers Insert Model

  @needs_review
  Scenario: Display templates as details
    Given the new-document template chooser is open in the MCAD Software
    When I open the template display menu
    Then the menu offers icon sizes, List, Details, and Grouped by Type
    When I choose Details
    Then the template gallery displays rows with Name, Size, and Type columns
    And the template details pane remains visible

  @needs_review
  Scenario: Toggle template grouping by document type
    Given the new-document template chooser is open in the MCAD Software
    And the metric template folder is selected
    And a standard metric part template's details are displayed
    And templates are grouped by type
    When I clear Grouped by Type in the template display menu
    Then the template gallery displays a flat collection without type group headings
    And the details pane continues to display the metric part template information
    When I select Grouped by Type in the template display menu
    Then the gallery displays part, assembly and drawing group headings
    And the details pane continues to display the metric part template information

  @needs_review
  Scenario: Select an assembly template in list view
    Given the new-document template chooser is open in the MCAD Software
    And the metric template folder is selected
    And templates are grouped by type
    When I choose List in the template display menu
    Then the gallery displays compact template entries under type headings
    And the template details pane remains visible
    When I select the standard metric assembly template in the list
    Then the details pane identifies an assembly with millimeter units
    When I choose Medium icons in the template display menu
    Then the gallery displays medium-sized template icons
    And the details pane continues to identify the metric assembly template

  @needs_review
  Scenario: Navigate backward and forward through template folders
    Given the new-document template chooser is open in the MCAD Software
    When I select the English template folder
    And I select the metric template folder
    And I choose Back in the chooser navigation bar
    Then the English folder is selected
    And the gallery displays its inch templates
    When I choose Forward in the chooser navigation bar
    Then the metric folder is selected
    And the gallery displays its metric templates

  @needs_review
  Scenario: Reopen the chooser after canceling a selected template
    Given the new-document template chooser is open in the MCAD Software
    And the metric folder and standard metric assembly template are selected
    When I cancel the chooser
    And I reopen the new-document chooser during the same session
    Then the metric folder is selected
    And the standard metric assembly template is selected
    And its details identify an assembly with millimeter units

  @needs_review @needs_fixture
  Scenario: Navigate home from a template folder
    Given the new-document template chooser is open in the MCAD Software
    And the metric template folder is selected
    And the standard metric assembly template details are displayed
    And the template root has no templates directly in it
    When I select Home in the chooser navigation bar
    Then the template root is selected in the folder tree
    And the gallery type sections contain no template entries
    And the metric assembly template details remain displayed
    When I choose Back in the chooser navigation bar
    Then the metric template folder and its gallery are displayed again

  @needs_review
  Scenario: Change template icon size
    Given the new-document template chooser is open in the MCAD Software
    And the metric folder is selected and templates are grouped by type
    And the standard metric assembly template details are displayed
    When I choose Extra large icons in the template display menu
    Then the template gallery displays extra-large tiles
    And the metric assembly template details remain displayed
    When I choose Small icons in the template display menu
    Then the template gallery displays small icons
    And the metric assembly template details remain displayed
    When I choose Medium icons in the template display menu
    Then the template gallery displays medium-sized icons
    And the metric assembly template details remain displayed

  @needs_review
  Scenario: Create a part by double-clicking its template
    Given a part document is open in the MCAD Software
    And the new-document template chooser is open
    And the metric template folder is selected
    When I double-click the standard metric part template
    Then the chooser closes
    And a new part document becomes active
    And its viewport contains no visible model geometry
    And its model browser displays an origin and an end-of-part marker
    And a new document tab is visible

  @needs_review
  Scenario: Create a selected part template with Enter
    Given the new-document template chooser is open in the MCAD Software
    And the metric template folder is selected
    When I select the standard metric part template
    And I press Enter
    Then the chooser closes
    And a new empty part document becomes active
    And its model browser displays an origin and an end-of-part marker
    And a new document tab is visible

  @needs_review
  Scenario: Dismiss the template chooser with Escape
    Given a part document is active in the MCAD Software
    And the new-document template chooser is open
    When I press Escape
    Then the chooser closes
    And the original part remains active
    And its viewport and browser are displayed
    And no new document tab is created
