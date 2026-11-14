Feature: Visual check of ribbon docking

  Scenario: Visual right ribbon
    Given a part is active in the MCAD Software
    And the model browser is docked on the left
    And the ribbon is docked at the top with the 3D Model tab selected
    When the user right-clicks the active 3D Model tab and chooses Left from Docking Positions
    Then the ribbon tabs are arranged vertically along the left edge
    And the 3D Model command groups are stacked beside the tabs
    And the model browser appears to the right of the ribbon
    When the user right-clicks the active 3D Model tab and opens Docking Positions
    Then Left is checked
    When the user chooses Right from the ribbon Docking Positions menu
    Then the ribbon tabs and command groups are arranged vertically on the right
    And the model browser is visible on the left of the viewport
