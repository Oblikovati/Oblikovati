@application @window @wireframe_gap
Feature: Main application window frame
  The MCAD Software integrates desktop window controls into its application bar.

  Scenario Outline: Display the application bar as the top of the main window
    Given the MCAD Software is open in a restored window showing <context>
    And the quick access toolbar is configured to appear above the ribbon
    Then the application bar forms the top of the window with no separate system title bar
    And quick access command icons sit at the leading side of the application bar
    And the application or active document title occupies the same bar
    And Minimize, Maximize and Close sit at the trailing edge in that order
    And the controls are distinguishable by their icons and accessible names

    Examples:
      | context        |
      | Home           |
      | an active part |

  Scenario: Keep the ribbon below the application bar
    Given a part document is active in the MCAD Software
    And the quick access toolbar is configured to appear above the ribbon
    And the ribbon is docked at the top in its expanded Normal appearance
    Then File and the ribbon tabs appear on a separate row beneath the application bar
    And the active tab's command panels are below the ribbon tab row
    And the window controls remain in the application bar

  Scenario Outline: Minimize and return from the taskbar
    Given a part document is active in the MCAD Software
    And the main window is <window state>
    When the user selects Minimize in the application bar
    Then the main window is minimized and its taskbar entry remains available
    When the user activates the application from the taskbar
    Then the main window returns in the <window state> state
    And the same part and ribbon tab are active
    And the displayed part retains its view orientation

    Examples:
      | window state |
      | restored     |
      | maximized    |

  Scenario: Maximize and restore using the application bar
    Given a part document is active in the MCAD Software
    And the restored main window occupies a known size and position within the desktop work area
    When the user selects Maximize in the application bar
    Then the window fills the monitor's work area and respects the taskbar
    And Restore replaces Maximize in the same position
    And the ribbon and viewport lay out within the enlarged window
    When the user selects Restore in the application bar
    Then the window returns to its previous size and position
    And Maximize replaces Restore in the same position
    And the same part remains active

  Scenario: Move the main window using the application bar
    Given a part document is active in the MCAD Software
    And the main window is restored within the desktop work area
    When the user drags an empty region of the application bar within the work area
    Then the entire window follows the drag without changing size
    And the same part remains active in the viewport

  Scenario: Double-click the application bar to maximize and restore
    Given the MCAD Software is open in a restored window
    When the user double-clicks an empty region of the application bar
    Then the main window is maximized
    And the window controls show Restore
    When the user double-clicks an empty region of the application bar again
    Then the main window returns to its previous size and position
    And the window controls show Maximize

  Scenario Outline: Resize the restored main window from its perimeter
    Given a part document is active in the MCAD Software
    And the main window is restored with space to grow within the desktop work area
    When the user moves the pointer onto the <handle> of the main window
    Then the pointer shows a <direction> resize cursor
    When the user drags that resize handle outward and releases it
    Then the window grows along the <direction> axes of that handle
    And the opposite boundary stays fixed
    And the viewport and visible ribbon fit within the resized window
    And all application window controls remain reachable

    Examples:
      | handle              | direction           |
      | left edge           | horizontal          |
      | right edge          | horizontal          |
      | top edge            | vertical            |
      | bottom edge         | vertical            |
      | top left corner     | northwest-southeast |
      | bottom right corner | northwest-southeast |
      | top right corner    | northeast-southwest |
      | bottom left corner  | northeast-southwest |

  Scenario: Keep the application controls reachable at the minimum window size
    Given the MCAD Software is open in a restored window with an active part
    When the user drags the bottom right corner inward past the supported minimum size
    Then the window stops shrinking at its minimum size
    And Minimize, Maximize and Close remain fully visible and usable
    And the active part remains displayed in the resized viewport

  Scenario: Keep the maximized bounds when dragging a resize edge
    Given the main window of the MCAD Software is maximized
    When the user attempts to resize it from the left boundary
    Then the window retains its maximized bounds
    And Restore remains available in the application bar

  Scenario: Route the custom Close control through the unsaved changes flow
    Given the MCAD Software has an open part document with unsaved changes
    When the user selects the application window's Close control
    Then a save-changes prompt identifies the part document
    When the user chooses Cancel
    Then the main window remains open and the part retains its unsaved changes

  Scenario: Keep the window Close action available to the desktop
    Given the MCAD Software has an open part document with unsaved changes
    When the user invokes the desktop window Close shortcut
    Then a save-changes prompt identifies the part document
    When the user chooses Cancel
    Then the main window remains open and the part retains its unsaved changes
