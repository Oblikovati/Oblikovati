@application @menus
Feature: File menu

  @needs_review
  Scenario: Open the file menu while a part is active
    Given a part document is active in the MCAD Software
    When I select File
    Then a menu opens with New, Open, Save, Save As, Export, Print and Close entries
    And a recent-documents region is visible
    And the model workspace remains visible behind the menu
