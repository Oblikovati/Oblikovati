Feature: Edit a Chamfer preview
  Scenario: Cancel and accept a Chamfer distance edit
    Given the MCAD Software is displaying a part in the solid modeling workspace
    And a committed chamfer is open for editing
    When the user changes the chamfer distance to 2 mm
    Then the chamfer edit preview does not change the committed distance
    When the user cancels the chamfer preview
    Then the committed chamfer distance is still 3 mm
    When the user reopens the chamfer editor
    And the user changes the chamfer distance to 2 mm
    And the user accepts the chamfer edit
    Then the committed chamfer distance is 2 mm
