# Projects docking and close

The native check dragged standalone Projects from Quick Access into
Material Browser's dock. Done and Escape each revealed Material Browser in
the first frame after closure. Reopening Projects restored the same dock.
A new native window and owner loaded the saved profile, reopened Projects
in the same dock, and closed it with Done while the Part stayed active.
[Wayland checks](wayland.txt) · [X11 checks](x11.txt) · [Shared dock](wayland-projects-dock.png)

The chooser check seeded a restored placement for its Projects child.
Cancelling New document closed both windows and revealed Material Browser
on the first frame after closure. The profile initially kept two browser
tabs, so the older assertion named a different sibling from the one that
was actually visible. The revised check starts with one neighboring tab.
[First frame after closing the chooser](wayland-chooser-closed.png)

The published Project Dialog scenario opened and closed Projects with a
selected template. The test now clicks the Projects button within the
chooser even when its glyphs share a draw run with the active-project
caption. Six of the eight published Project Dialog scenarios passed, with
85 steps in the passing cases. Both workspace-folder cases stopped in the
folder browser.
[Combined X11 scenario report](cucumber.json) · [Published input hashes](current-inputs.sha256)

An exploratory run restored the chooser child after restart and revealed
its saved tab. Done could not receive pointer input behind the active
chooser modal on either backend. The experimental focus changes were
removed. This input restriction remains open under #4073.
[Wayland failure](modal-input-wayland.txt) · [X11 failure](modal-input-x11.txt)

The builds used `2485a0512` with the test changes in this commit and
existing local filesystem and sketch-marking test edits. X11 used Xvfb and
XTest. Wayland used isolated headless Weston with Pixman and application
frame-loop input. Both used Vulkan Lavapipe.
