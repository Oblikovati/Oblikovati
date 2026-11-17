Feature: Cancel a Chamfer preview
  Scenario: Cancel a visible Chamfer preview
    Given the MCAD Software is displaying a part in the solid modeling workspace
    And Chamfer is open in equal-distance mode with a valid distance
    When the user clicks an eligible block edge
    Then a bevel preview appears
    When the user cancels the chamfer preview
    Then the chamfer preview is discarded without changing the solid
