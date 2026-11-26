# Minimize and return from the taskbar

Both published examples passed on Wayland and X11, with 18 steps on
each backend. The application starts with the saved Part, minimizes from
the application bar and returns when the user clicks its taskbar entry.
It returns to the original restored or maximized state with the same
active Part, ribbon tab and camera orientation.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Native trace](test-results.txt)

Wayland used Labwc 0.7.1 and Waybar 0.9.24 on private Xvfb. The test samples
the visible Waybar entry to confirm that it remains present and reports
the minimized state. It clicks the same entry with native pointer input
and waits for its active state. The protocol trace shows the application's
minimize request and Waybar's activation of the corresponding toplevel.
X11 used Openbox and tint2. Both used Vulkan Lavapipe.

The build was based on `a86a1d92b` and included the existing local GUI
filesystem edits. The inputs are identified by content hashes.
[Input hashes](inputs.sha256)
