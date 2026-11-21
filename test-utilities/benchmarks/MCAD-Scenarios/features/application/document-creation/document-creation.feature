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

  @needs_review @needs_fixture
  Scenario: Inspect projects from the template chooser
    Given the new-document template chooser is open in the MCAD Software
    And a project is active
    When I choose Projects in the chooser
    Then a Projects dialog opens above the chooser
    And it lists project names and locations
    And the active project is marked
    And the dialog displays a project settings tree
    When I choose Done without changing the project
    Then the Projects dialog closes
    And the template chooser is displayed again
    And the previously selected template details remain displayed

  @needs_review @needs_fixture
  Scenario: Reverse template size sorting in Details view
    Given the new-document template chooser is open in the MCAD Software
    And the metric template folder is selected
    And the gallery is grouped by template type and displayed in Details view
    And the standard metric part template details are displayed
    And the Size column is not the active sort column
    When I click the Size column header
    Then the visible part templates are ordered by increasing file size
    And the standard metric part template details remain displayed
    When I click the Size column header again
    Then the visible part templates are ordered by decreasing file size
    And the standard metric part template details remain displayed
    And the gallery remains grouped by template type

  @needs_review @needs_fixture
  Scenario: Retain template sort order when reopening the chooser
    Given the new-document template chooser is open in the MCAD Software
    And the metric gallery is grouped by template type in Details view
    And the visible part templates are sorted by decreasing file size
    When I choose Medium icons in the template display menu
    And I cancel the chooser
    And I reopen the chooser in the same session
    Then the metric template folder and medium icon view are displayed
    When I choose Details in the template display menu
    Then the visible part templates are ordered by decreasing file size
    And the Size column indicates descending order

  @needs_review @needs_fixture
  Scenario: Create a part after navigating to an empty template root
    Given the new-document template chooser is open in the MCAD Software
    And the standard metric part template is selected
    And the template root contains no direct template entries
    When I choose Home in the template chooser
    Then the root is selected and its template groups are empty
    And the standard metric part template details remain displayed
    When I choose Create
    Then the chooser closes
    And a new empty part document becomes active
    And its model browser displays an origin and an end-of-part marker
    And a new document tab is visible

  @needs_review @needs_fixture
  Scenario: Enable creation by selecting a template
    Given the new-document template chooser is open in the MCAD Software
    And no template is selected
    And the selected template root has no direct template entries
    Then Create is disabled
    When I select the metric template folder
    Then template tiles are displayed
    And Create remains disabled
    When I select the standard metric assembly template
    Then its assembly details are displayed
    And Create is enabled

  @needs_review @needs_fixture
  Scenario: Create the retained template after changing folders
    Given the new-document template chooser is open in the MCAD Software
    And the standard metric assembly template is selected
    When I select the English template folder without selecting another template
    Then the gallery displays the English folder templates
    And the details still identify the metric assembly template
    When I choose Create
    Then the chooser closes
    And a new empty assembly document becomes active
    And its browser displays representations, relationships and origin entries
    And a new document tab is visible

  @needs_review @needs_fixture
  Scenario: Cancel component selection for a drawing layout
    Given a part document is active in the MCAD Software
    And the new-document template chooser is open
    And an inch drawing template with a populated assembly layout preview is available
    When I select the inch drawing template
    Then the chooser displays the template sheet and available layout previews
    When I select the populated assembly layout preview
    And I choose Create
    Then the template chooser closes
    And a component-selection dialog opens for a drawing view
    And the dialog displays a document name field and a browse control
    When I cancel component selection
    Then the component-selection dialog closes
    And the original part document remains active
    And no new drawing tab is displayed

  @needs_review @needs_fixture
  Scenario: Create a populated drawing layout with a component that needs updating
    Given a part document is active in the MCAD Software
    And a drawing template provides a populated layout with a parts list and a view
    And an available part component requires an update
    When I select the drawing template and populated layout
    And I choose Create
    And I confirm that component in the component-selection dialog
    Then a message identifies the component as needing an update
    And the message says the drawing reflects the model's non-updated state
    When I acknowledge the message
    Then a new drawing document becomes active
    And its sheet displays a border, title block and parts list
    And the drawing browser lists a parts list and a view referencing the component
    And a new document tab is visible

  @needs_review @needs_fixture
  Scenario: Cancel browsing for a drawing-layout component
    Given a populated drawing layout is being created in the MCAD Software
    And the component-selection dialog displays a document name
    When I choose Browse beside the document name
    Then a file-selection dialog opens above component selection
    And it displays a file list, a file name field and Open and Cancel controls
    When I cancel the file-selection dialog
    Then the component-selection dialog is displayed again
    And its previous document name remains displayed
    When I cancel component selection
    Then the original document remains active
    And no new drawing tab is displayed

  @needs_review @needs_fixture
  Scenario: Browse to a different component for a populated drawing layout
    Given a populated drawing layout is being created in the MCAD Software
    And the component-selection dialog displays an initial component
    And a different part file that needs updating is available
    When I choose Browse beside the document name
    And I select the different part file and choose Open
    Then the component-selection dialog displays the selected file name
    When I confirm the selected component
    Then a message identifies the selected component as needing an update
    When I acknowledge the message
    Then a new drawing becomes active
    And its parts list identifies the selected component
    And its browser contains a view referencing the selected component
