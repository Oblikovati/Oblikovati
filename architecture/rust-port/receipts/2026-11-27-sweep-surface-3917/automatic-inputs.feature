Feature: Sweep
  Scenario: surface-sweep-automatic-inputs Create a surface with unambiguous inputs
    Given a disposable copy of the native fixture "fixtures/cad/sweep/block-outward-path.opd" is open
    And a circular sketch curve and a straight path are available
    When Sweep is clicked
    And the Sweep is switched to surface output
    Then Profiles shows 1 Curve and Body Name shows Srf1
    When the Sweep is accepted
    Then SweepSrf1 and Surface Bodies(1) appear
    And the Surface Bodies folder contains Srf1
    And Solid Bodies(1) remains
