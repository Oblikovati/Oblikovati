@application @documents
Feature: First save of a new part

  @needs_review
  Scenario: Open and cancel the first save
    Given a new unsaved part is active in the MCAD Software
    When I press Control+S
    Then a Save As dialog opens
    And a proposed document name is selected in the file name field
    And a location, file type and preview options are displayed
    When I cancel the dialog
    Then the Save As dialog closes
    And the unsaved part remains active

  @needs_review
  Scenario: Save and reopen a newly created empty part
    Given a new unsaved empty part is active in the MCAD Software
    And the proposed file name does not already exist in the selected folder
    When I press Control+S
    And I accept the proposed file name in the Save As dialog
    Then the dialog closes
    And the active document tab displays the saved file name
    When I close that document tab
    And I open the saved file from the same folder
    Then the saved part becomes active
    And its viewport contains no visible model geometry
    And its model browser displays the origin and end-of-part marker

  @needs_review
  Scenario: Save changes to an existing part and reopen its sketch
    Given a saved empty part is open in the MCAD Software
    When I create a planar sketch with a rectangular profile
    And I finish the sketch
    And I press Control+S
    Then the existing document remains open without a Save As dialog
    When I close the document tab
    And I reopen the saved part in the same session
    Then the model browser contains the sketch
    And the rectangular sketch profile is visible in the viewport

  @needs_review
  Scenario: Toggle the preview picture controls before saving
    Given the Save As dialog is open for a new part in the MCAD Software
    And Save Preview Picture is selected
    When I clear Save Preview Picture
    Then the preview source choices are hidden
    And the preview area displays a placeholder
    When I select Save Preview Picture again
    Then the preview source choices reappear
    When I choose Active Window on Save
    Then that preview source is selected
    And Active Component Iso View on Save is not selected
    When I choose Active Component Iso View on Save
    And I cancel the dialog
    Then the unsaved part remains active

  @needs_review
  Scenario: Browse for a preview image and cancel the image chooser
    Given the Save As dialog is open for a new part in the MCAD Software
    And Save Preview Picture is selected
    When I choose Import From File as the preview source
    Then a browse control appears beside that option
    And the preview area displays a placeholder
    When I choose the browse control
    Then an image file chooser opens above the Save As dialog
    When I cancel the image file chooser
    Then the Save As dialog remains open
    And Import From File remains selected
    And the preview area still displays a placeholder

  @needs_review
  Scenario: Inspect file type choices when saving a part
    Given the Save As dialog is open for a new part in the MCAD Software
    When I open the file type list
    Then the list offers the native part file type and an All Files filter

  @needs_review
  Scenario: Cancel closing a modified unsaved part
    Given an unsaved part containing a finished sketch is active in the MCAD Software
    And save-change prompts are configured to always show
    When I close the document tab
    Then a save-changes prompt offers Yes, No and Cancel
    When I choose Cancel
    Then the part remains active
    And its sketch remains in the model browser

  @needs_review
  Scenario: Cancel the first save requested while closing a part
    Given an unsaved part containing a finished sketch is active in the MCAD Software
    And save-change prompts are configured to always show
    When I close the document tab
    And I choose Yes in the save-changes prompt
    Then the Save As dialog opens
    When I cancel Save As
    Then the part remains active
    And its sketch remains in the model browser

  @needs_review
  Scenario: Inspect save-change prompt frequency choices
    Given an unsaved modified part is active in the MCAD Software
    And save-change prompts are configured to always show
    When I close the document tab
    And I open the prompt-frequency list
    Then the list offers always show, once per operation, suppress for this session and suppress permanently
    When I dismiss the list without changing its selection
    And I cancel the save-changes prompt
    Then the part remains active

  @needs_review
  Scenario: Save a new modified part while closing it
    Given an unsaved part containing a finished sketch is active in the MCAD Software
    And save-change prompts are configured to always show
    And the proposed file name does not exist in the selected folder
    When I close the document tab
    And I choose Yes in the save-changes prompt
    And I accept the proposed file name in Save As
    Then the document closes
    And another open document becomes active
    When I reopen the newly saved part in the same session
    Then the part becomes active
    And its model browser contains the finished sketch
