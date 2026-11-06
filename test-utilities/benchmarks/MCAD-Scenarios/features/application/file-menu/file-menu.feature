@application @menus
Feature: File menu

  @needs_review
  Scenario: Open the file menu while a part is active
    Given a part document is active in the MCAD Software
    When I select File
    Then a menu opens with New, Open, Save, Save As, Export, Print and Close entries
    And the menu also shows Share, Manage and document properties entries
    And a recent-documents region is visible
    And application options and an exit control appear at the bottom of the menu
    And the model workspace remains visible behind the menu

  @needs_review
  Scenario: Dismiss the file menu with Escape
    Given a part document is active in the MCAD Software
    And the File menu is open
    When I press Escape
    Then the File menu and recent-documents region close
    And the same part document remains active

  @needs_review
  Scenario: Dismiss the file menu by clicking outside
    Given an empty part document is active in the MCAD Software
    And the File menu is open
    When I click an empty area of the viewport outside the menu
    Then the File menu and recent-documents region close
    And the same part document remains active

  @needs_review
  Scenario: Open and dismiss the file menu in a drawing
    Given a drawing document is active in the MCAD Software
    When I select File
    Then the menu shows New, Open, Save, Save As, Export, Share, Manage, document properties, Print and Close entries
    And a recent-documents region is visible
    And the drawing sheet remains visible around the menu
    When I press Escape
    Then the File menu closes
    And the same drawing document remains active
