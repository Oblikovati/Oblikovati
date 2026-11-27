# Cancel an invalid Closed Loop Loft edit

The published `loft-15` scenario reopens a two-circle Loft with a saved
intermediate section. Checking Closed Loop and accepting the edit runs the
native check on the straight guide wire. The warning's Details reports that
the guide has distinct start and end points and must be periodic. Cancel
restores the committed solid and leaves its body identity, document generation
and saved section size intact. The viewport clears the draft annotations and
preview color.

| Backend | Scenario |
| --- | --- |
| Wayland on COSMIC | [Eight steps passed](wayland/junit.xml) |
| X11 under Xvfb and Openbox | [Eight steps passed](x11/junit.xml) |
