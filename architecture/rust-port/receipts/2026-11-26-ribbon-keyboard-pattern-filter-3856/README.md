# Keyboard command search for Pattern

The published scenario **Filter keyboard commands by keyword and clear the
filter** opened the named Part on Wayland and stopped at the undefined
Keyboard tab setup step. Its later assertions require Pattern matches across
multiple command categories. The desktop shortcut catalog currently has no
Pattern commands. The published visual reference includes Sketch, Features,
Annotation, and Sheet Metal matches.

The existing native keyboard filter regression passed on Wayland and X11.
It verifies typing a keyword, filtering out a nonmatching command, and
clearing the search to restore the command list. It also exercises staging
and applying a shortcut reset.

The test binary used application revision `cc6469b35` plus the viewport test
changes and existing local GUI filesystem edits. The desktop runs used nested
Weston 13 and Openbox under Xvfb, with Vulkan Lavapipe on both.

[Published diagnostic](published-wayland.json) · [Native filter results](filter-regression.txt) · [Input hashes](inputs.sha256)
