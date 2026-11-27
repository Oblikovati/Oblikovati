@application @documents
Feature: Create a layout from a current Part
  Scenario: Confirm a current Part without an update warning
    Given a fresh MCAD Software session loads `fixtures/cad/document-creation-layout-candidate.opj` as its active project
    And a disposable copy of `fixtures/cad/document-creation-layout-candidate/English/layout-data/extrusion/extrusion-rect-8mm.opd` is active
    And `fixtures/cad/document-creation-layout-candidate/English/Populated Layout.odd` is available as a drawing layout with a view and parts list
    When the user selects the drawing template and populated layout
    And the user chooses Create
    Then the component-selection dialog displays the proposed component
    When the user confirms that component in the component-selection dialog
    Then a new drawing document becomes active
    And the drawing browser lists a parts list and a view referencing the component
    And a new document tab is visible
