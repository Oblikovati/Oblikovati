@sketch @needs_fixture
Feature: Move

  @needs_fixture
  Scenario: move-06 Enter a precise ellipse Move endpoint
    Given an editable sketch with eligible geometry
    Given the ellipse is selected in Move with Copy off and Precise Input on
    When the user chooses Base Point and clicks its center
    Then the coordinate toolbar appears and the status requests an endpoint
    When the user types 1 then 0 into X and presses Tab to Y
    And the user types 0 and presses Enter
    Then Move returns to geometry selection without an error dialog
    And the ellipse appears above its original position
    When the user clicks Done
    Then Move closes and the displaced ellipse remains

  @documented_observation @needs_fixture @precise_history
  Scenario: move-07 Undo and redoes the accepted precise endpoint
    Given an editable sketch with eligible geometry
    Given the precise ellipse Move has been accepted and Move closed
    When the user undoes
    Then the ellipse returns to its original screen position
    When the user redoes
    Then the ellipse returns to its displaced screen position
    When the user undoes
    Then the original fixture position is restored again

  @copy_mode @documented_observation @needs_fixture
  Scenario: move-08 Retain the source when Copy is enabled
    Given an editable sketch with eligible geometry
    Given Move has reopened with Precise Input checked and Copy unchecked
    When the user enables Copy and selects the original ellipse
    And the user chooses its center as Base Point and clicks the displaced endpoint
    Then Move returns to geometry selection
    And a displaced ellipse and the original ellipse are both visible
    When the user clicks Done
    Then both ellipse outlines remain

  @copy_history @documented_observation @needs_fixture
  Scenario: move-09 undo and redoes a copied ellipse
    Given an editable sketch with eligible geometry
    Given Move Copy has created the displaced ellipse and Move is closed
    When the user undoes
    Then only the original ellipse remains in the pair
    When the user redoes
    Then the displaced ellipse returns
    When the user undoes
    Then the trial copy is removed again

  @documented_observation @needs_fixture @single_selection
  Scenario: move-10 automatically advance to Base Point after one selection
    Given an editable sketch with eligible geometry
    Given Move has Copy and Precise Input off
    When the user enables Optimize for Single Selection and clicks the ellipse perimeter
    Then Base Point activates without a separate button click
    And the status reads Select base point
    When the user presses Escape before choosing a base point
    Then Move closes and the ellipse remains in its original position

  @documented_observation @endpoint_cancel @needs_fixture
  Scenario: move-11 Cancel after choosing a base point
    Given an editable sketch with eligible geometry
    Given Move reopens with Optimize for Single Selection checked
    And Copy and Precise Input are unchecked
    When the user selects the ellipse and clicks its center as base point
    Then the status requests an endpoint for Move
    When the user presses Escape before accepting an endpoint
    Then Move closes and the ellipse remains in its original position

  @documented_observation @needs_fixture @same_point
  Scenario: move-12 reject an identical canvas base and endpoint in this fixture
    Given an editable sketch with eligible geometry
    Given the ellipse is selected with optimization on and Copy and Precise Input off
    When the user chooses a canvas location as base point
    And the user chooses the same location again as endpoint
    Then Move reports that the selected geometry could not be moved
    And the message refers to Constraint Settings for Sketch Edit Tools
    When the user clicks Cancel
    Then Move returns to geometry selection
    When the user expands options
    Then both constraint policies show Prompt
    When the user clicks Done
    Then the ellipse has no visible displacement

  @coordinate_modes @documented_observation @needs_fixture
  Scenario: move-13 Choose Distance and X axis angle fields
    Given an editable sketch with eligible geometry
    Given the selected ellipse is at the precise Base Point stage
    When the user opens the coordinate-mode dropdown
    Then four coordinate and angle combinations are listed
    When the user chooses Distance and X axis angle
    Then the toolbar exposes Distance in mm and Angle in degrees

  @documented_observation @needs_fixture @triad_reset
  Scenario: move-14 reset the triad without placing geometry
    Given an editable sketch with eligible geometry
    Given the ellipse is selected in precise Base Point mode
    And the triad remains at its prior visible position
    When the user clicks Reset to Origin
    Then the visible triad moves to the sketch axes intersection
    And the selected ellipse remains at its original position
    And the Distance and Angle fields appear blank
    When the user presses Escape
    Then Move and its toolbar close without an accepted endpoint

  @documented_observation @needs_fixture @polar_numeric
  Scenario: move-15 accept a numeric distance-and-angle endpoint
    Given an editable sketch with eligible geometry
    Given Move reopens with Precise Input and single-selection optimization on and Copy off
    When the user selects the ellipse
    Then Distance and X axis angle mode is retained
    When the user clicks its center as base point
    And the user focuses Distance, selects its text and types 2 then 0
    And the user presses Tab to Angle, types 0 and presses Enter
    Then Move returns to geometry selection without an error dialog
    And the ellipse remains at the lower preview position
    When the user clicks Done
    Then Move closes
    When the user undoes
    Then the ellipse returns to its original position

  @coordinate_reference @documented_observation @needs_fixture
  Scenario: move-16 Inspect Relative and Absolute coordinate modes
    Given an editable sketch with eligible geometry
    Given the selected ellipse is in precise Base Point mode
    When the user opens the left toolbar dropdown
    Then Relative and Absolute choices are visible
    When the user chooses Absolute
    Then the visible triad is at the sketch axes intersection
    And the selected ellipse retains its recorded position
    When the user reopens the dropdown and chooses Relative
    Then the toolbar icon changes
    And the triad remains at the axes intersection

  @documented_observation @needs_fixture @toolbar_close
  Scenario: move-17 close the mini-toolbar while retaining Move
    Given an editable sketch with eligible geometry
    Given the precise toolbar is visible in Move Base Point mode
    When the user opens its right menu
    Then Pin Mini-Toolbar Position, Auto Fade and closes are listed
    When the user chooses Close
    Then the toolbar and triad disappear while Move remains open
    And Precise Input remains checked
    When the user presses Escape
    Then Move closes without accepting placement

  @needs_fixture
  Scenario: move-18 Never policies permit observed coupled line displacement
    Given an editable sketch with eligible geometry
    Given the recorded fixture has two horizontal lines and their 25 +/-1 separation annotation
    And Copy and Precise Input are off and Optimize for Single Selection is on
    When both constraint policies are Never
    And the upper line is moved from the base point to the endpoint
    Then Move returns to geometry selection without an error or relaxation question
    And both horizontal lines visibly shift screen-up while the separation annotation remains
    When both policies are restored to Prompt and Done is clicked
    And the user presses Undo
    Then the two lines return to their recorded original screen positions

  @needs_fixture
  Scenario: move-19 Mixed X coordinate and angle endpoint accepts
    Given an editable sketch with eligible geometry
    Given Move has Copy off and Precise Input and Optimize for Single Selection on
    And the ellipse is selected with X coordinate and X axis angle mode
    When its recorded center is chosen as the base point
    And individual keys enter X=10 and Angle=45
    And Return is pressed
    Then Move returns to geometry selection without an error or question
    And one ellipse remains at the up-left position
    When Done and undoes are used
    Then the ellipse returns to its original visible position

  @needs_fixture
  Scenario: move-20 Mixed Y coordinate and angle endpoint accepts
    Given an editable sketch with eligible geometry
    Given reopening Move retains the previously selected mixed X/angle mode
    When Y coordinate and X axis angle is selected
    And the ellipse center is chosen as the base point
    And individual keys replace Y with 5 and Angle with 45
    And Return is pressed
    Then one ellipse remains at the up-left position
    And Move returns to geometry selection
    When Done and undoes are used
    Then the original visible ellipse position is restored

  @needs_fixture
  Scenario: move-21 Mini-toolbar Pin check state toggles
    Given an editable sketch with eligible geometry
    Given the precise-input mini-toolbar is visible at Move Base Point stage
    And Pin Mini-Toolbar Position is unchecked
    When Pin Mini-Toolbar Position is clicked and Settings is reopened
    Then Pin Mini-Toolbar Position is checked
    When Pin is later clicked again and Settings is reopened
    Then Pin Mini-Toolbar Position is unchecked

  @needs_fixture
  Scenario: move-22 Auto Fade reduces toolbar and hover restores it
    Given an editable sketch with eligible geometry
    Given the mini-toolbar is visible with Pin checked and Auto Fade unchecked
    When Auto Fade is enabled and the pointer remains away
    Then the mode selector and lower toolbar row disappear while numeric fields remain
    When the pointer hovers over the Y input
    Then the full toolbar reappears
    And reopened Settings shows Auto Fade checked
    When Auto Fade is disabled and Settings is reopened
    Then Auto Fade is unchecked

  @needs_fixture
  Scenario: move-23 Never policies reject the white-curve selection
    Given an editable sketch with eligible geometry
    Given both Move constraint policies are Never and precise input is off
    And the recorded white-curve fixture is present
    When the white-curve selection and base and endpoint clicks are supplied
    Then the Move constraint-settings failure is displayed
    And the original white outline remains
    When Cancel is clicked
    Then Move returns to geometry selection

  @needs_fixture
  Scenario: move-24 Geometric Always changes the outcome for the same clicks
    Given an editable sketch with eligible geometry
    Given the failed trial has returned to geometry selection
    When only Break Geometric Constraints is changed to Always
    And the same recorded selection and base and endpoint clicks are supplied
    Then Move returns to geometry selection without a question or error
    And a displaced purple point appears while the broad curve remains in its original location
    When both policies are restored to Prompt and Done is clicked
    Then Ready returns
    When Undo is pressed
    Then the displaced point disappears and the white outline is restored

  @needs_fixture
  Scenario: move-25 Accept the Always dimensional policy trial
    Given the recorded rectangle fixture is present
    And Move has Copy off, Precise Input off and optimization on
    And dimensional relaxation is Always and geometric breaking is Never
    When the user selects the rectangle left edge and its lower endpoint as base
    And the user clicks the endpoint 40 pixels left
    Then Move returns to geometry selection without a question or error
    And the rectangle widens with linked geometry shifting
    When the user clicks Done and measures the moved lower endpoint
    Then its position is (0.000, -174.642, -202.352) mm
    When the user closes Measure and undoes
    Then the earlier rectangle width and linked geometry return

  @needs_fixture
  Scenario: move-26 Accept the same placement under If No Equation
    Given the earlier Always trial was undone
    When the user reopens and expands Move
    Then Always and Never remain selected
    When the user selects If No Equation for dimensional relaxation
    And repeat the same edge base and endpoint clicks
    Then the placement accepts without a question or error
    And the same visible coupled deformation appears

 
  Scenario: move-27 Answer Yes to prompted dimensional relaxation
    Given both Move constraint policies are Prompt
    When the user selects the moved rectangle left edge and its lower endpoint as base
    Then the MCAD Software asks whether existing dimensions should be relaxed if necessary
    When the user clicks Yes and supplies the endpoint 20 pixels further left
    Then Move returns to geometry selection without another warning
    And the loop widens further with linked geometry shifted
    When the user clicks Done
    Then Move closes and the accepted placement remains

 
  Scenario: move-28 Measure the final accepted endpoint
    Given the recorded conditional and prompted rectangle placements are retained
    When the user opens Measure and selects the final lower-left rectangle endpoint
    Then its position is (0.000, -174.642, -186.644) mm

 
  Scenario: move-29 Retain the measured placement after actual disk reopen
    Given the recorded sketch is finished and its part saved
    When the user closes and reopens the part in the same session
    And edit the saved sketch and measures the lower-left rectangle endpoint
    Then its position remains (0.000, -174.642, -186.644) mm
    When the user clicks Done
    Then Measure closes and the saved geometry remains

  @needs_fixture
  Scenario: move-30 Select and move two sources without optimization
    Given Copy and Precise Input are off and optimization is off
    When the upper line and lower hourglass edge are selected by separate ordinary clicks
    Then both are highlighted and selects remains active
    When Base Point and the canvas base and endpoint are supplied
    Then Move returns to geometry selection without a question or error
    And both selected lines move up while connected hourglass diagonals deform

  @needs_fixture
  Scenario: move-31 Undo and redoes the two-source operation together
    Given the two-source Move has been accepted and Done clicked
    When Undo is pressed once
    Then both selected lines and the hourglass return to their original positions
    When Redo is pressed
    Then the displaced pair and deformed hourglass return
    When Undo is pressed again
    Then the original fixture is restored

  @needs_fixture
  Scenario: move-32 Reject invalid text and recover with signed numeric input
    Given the upper line is selected with precise X and Y endpoint entry active
    When q is typed into X and Return is pressed
    Then the red invalid input remains at endpoint selection without an accepted placement
    When separate keys replace X with -5 and Y with 0
    And Return is pressed
    Then Move returns to geometry selection with the displaced line
    When precise input is disabled and optimization enabled and Done clicked
    And the user presses Undo
    Then the upper line returns to its original position



