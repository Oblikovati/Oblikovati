# Close the window with a desktop shortcut

The published Main Window Frame case opens a writable copy of the native
Part, creates an unsaved XY sketch, and presses Alt+F4. The save prompt
names the Part. Cancel leaves the same document active and dirty and
preserves the working file byte for byte.

Wayland ran on Labwc 0.7.1 nested on private Xvfb with its own D-Bus
session. The runner binds Alt+F4 to Labwc's Close action. The client trace
records `xdg_toplevel.close()` arriving from the compositor. The X11 run
used Openbox on Xvfb. Both used Vulkan Lavapipe.

At baseline `7201e60a3`, the Wayland test injected a close event into the
application loop. The verification build used the compositor shortcut and
included the existing local GUI filesystem edits.
[Wayland report](wayland.json) · [Input hashes](inputs.sha256)

All seven Main Window Frame steps passed on Wayland and X11. Three
Application Closing scenarios sharing the Alt+F4 step also passed on
Wayland. They cover saving an existing Part, cancelling a first save,
and dismissing the prompt for a dirty Assembly.
[X11 report](x11.json) · [Save and reopen](save.json) · [Cancel first save](first-save.json) · [Cancel Assembly close](assembly.json) · [Protocol trace](test-results.txt) · [Application Closing inputs](closing-inputs.sha256)
