Feature: Visual check of ribbon docking

  Scenario: Visual left ribbon
    Given a part is active in the MCAD Software
    And the model browser is docked on the left
    And the ribbon is docked at the top with the 3D Model tab selected
    When the user right-clicks the active 3D Model tab and chooses Left from Docking Positions
    Then the ribbon tabs are arranged vertically along the left edge
    And the 3D Model command groups are stacked beside the tabs
    And the model browser appears to the right of the ribbon
