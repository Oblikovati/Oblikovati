@sketch @inherits_common_sketch_viewport
Feature: The MCAD Software sketch constraint settings

  @acceptance
  Scenario: constraint-settings-01 Commit the intended inference settings
    Given `fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited
    And Infer constraints and Persist constraints are checked
    When the user open Constraint Settings to Inference
    And the user clear Infer constraints and accept
    And the user reopen the Inference tab
    Then Infer constraints is unchecked
    And Persist constraints is unchecked and disabled
    When the user enable Infer constraints
    Then Persist constraints becomes available and remains unchecked
    When the user check Persist constraints and accept
    Then both settings are restored

  @edge_case
  Scenario: constraint-settings-02 Control constraint glyph visibility
    Given `fixtures/cad/constraint-settings/constraint-settings-line.opd` is open with Sketch1 being edited
    And a horizontal relation glyph is displayed on its line
    When the user right-click the glyph and choose Hide
    Then its glyph disappears and the line remains visible
    When the user press F8 for Show All Constraints
    Then the line glyph and other sketch constraint glyphs are displayed
    When the user right-click the test glyph and choose Hide All Constraints
    Then the displayed glyphs disappear across the sketch
    And visible geometry and dimension annotations remain

  @cancel
  Scenario: constraint-settings-03 Cancel changes to inference configuration
    Given `fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited
    And Infer constraints, Persist constraints and all nine inference types are checked
    And Parallel and perpendicular priority is selected
    When the user open Constraint Settings to Inference and click Clear All
    Then all nine inference types are unchecked
    When the user click Select All and clear Infer constraints
    Then Persist constraints, the checked inference types and bulk-selection buttons are disabled
    And the priority choices remain available
    When the user click Cancel and reopen Constraint Settings to Inference
    Then Infer constraints, Persist constraints and all nine inference types are checked
    And Parallel and perpendicular priority remains selected

  @cancel @relax_mode
  Scenario: constraint-settings-06 Cancel Relax Mode configuration changes
    Given `fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited
    And Relax Mode is disabled with twelve removal choices
    And Preserve dimensions with equation is checked and disabled
    When the user open Constraint Settings to Relax Mode and enable Relax Mode
    Then the removal choices, bulk-selection buttons and equation-preservation option become available
    When the user click Clear All and wait for settled painting
    Then all twelve removal choices are unchecked
    When the user click Select All
    Then all twelve removal choices are checked
    When the user clear Preserve dimensions with equation
    Then the twelve removal choices remain checked
    When the user click Cancel and reopen Constraint Settings to Relax Mode
    Then Relax Mode is disabled and the original removal choices are restored
    And Preserve dimensions with equation is checked and disabled

  @documented_observation
  Scenario: constraint-settings-07 Inspect restored settings through Application Options
    Given `fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited and the Tools ribbon visible
    When the user open Application Options to Sketch and Constraint Settings
    Then Edit dimension when created and Create dimensions from input values are checked
    And Warn of overconstrained condition is selected
    When the user choose Inference
    Then Infer constraints and Persist constraints and all nine types are checked
    And Parallel and perpendicular priority is selected
    When the user choose Relax Mode
    Then Enable Relax Mode is unchecked and its dependent controls are disabled
    When the user click Cancel and close Application Options
    Then the sketch returns to Ready

  @documented_observation
  Scenario: constraint-settings-08 Compare glyphs with inference disabled and enabled
    Given `fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited
    And Precise Input is active in Y-coordinate and X-axis-angle mode
    And Infer constraints is unchecked
    When the user create a near-horizontal line using the recorded two pointer clicks
    And exit Line and select the segment with Show constraints for selected objects enabled
    Then no constraint glyph is visible in that selection frame
    When the user measure the segment
    Then its recorded displayed length is 19.597 mm
    When the user exit Measure and undo the segment
    And restore Infer constraints and Persist constraints
    And the user create a replacement line using the same pointer clicks
    Then a constraint glyph appears during creation and after Escape
    And the selection popup identifies a Constraint

  @documented_observation
  Scenario: constraint-settings-09 Retain Relax Mode after accepting and restore it
    Given `fixtures/cad/constraint-settings/constraint-settings-line.opd` is open with Sketch1 being edited
    And Relax Mode is disabled with the recorded removal choices
    And Preserve dimensions with equation is checked
    When the user open Constraint Settings to Relax Mode, enable Relax Mode and accept
    And the user reopen Constraint Settings to Relax Mode
    Then Enable Relax Mode remains checked
    When the user clear Enable Relax Mode and accept
    Then the original disabled preference is restored

  Scenario: constraint-settings-10 Inspect initial Relax Mode removal choices
    Given `fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited
    And Relax Mode is initially disabled
    When the user open Constraint Settings to Relax Mode and enable Relax Mode
    Then Coincident, Tangent, Smooth(G2) and Symmetric are unchecked
    And Collinear, Concentric, Horizontal, Vertical, Parallel, Perpendicular, Equal and Fix are checked
    And Preserve dimensions with equation is checked
