# Ribbon customization

A part document can open Customize User Commands from a ribbon panel title or
from the Tools tab. Use Show Panels or the ribbon panel selector to hide or
show User Commands. Add commands to a tab's User Commands panel, reorder them,
and choose their size and text display on each row. New commands start with
Large and Text clear. A removed command keeps its options if it is added back
before Apply. Double-click a command to transfer it between the lists. Select a row and use Shift with Up or Down to select
multiple rows for transfer. Ctrl+A selects visible source rows or all target
rows when that list has focus. Apply keeps the dialog open; OK applies
and closes it. Cancel, Escape, and the title-bar close control discard edits
since the last Apply. Escape closes an open target or category list before it closes the dialog. A target
remains marked as changed until Apply, OK, or Cancel, even if an edit returns
the list to its earlier state.

The command list includes sketch geometry, constraint, and dimension commands.
Other commands remain limited to the entries shown in the dialog. Some
commands are unavailable until their host operations exist. Apply and OK save
settings in the user configuration directory. The desktop loads those settings
on startup. A save failure leaves the draft open and the applied ribbon unchanged.
Right-click a User Commands button to move it to the expanded panel or back.
The move is saved immediately and does not apply other pending edits. A side or
floating ribbon shows both areas in its User Commands group.

Export writes the current draft as versioned Oblikovati XML to a new file.
Version 2 records expanded-panel placement; version 1 files remain readable.
Import replaces the draft and requires Apply or OK.
The export chooser refuses to overwrite an existing file. If an import file is
missing, dismiss the warning to edit the selected filename.

Keyboard and marking-menu customization and other document types remain open work.
