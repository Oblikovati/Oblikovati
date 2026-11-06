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
