# File Menu visual contract

All nine amended File Menu scenarios passed on Wayland and X11, with
73 steps on each backend. The new scenario opened the File menu above
the published solid Part. It checked the brand fill with the pointer
away, measured the 32 px command and footer row pitch, and verified
the aligned leading icons and text indent.

The Recent ordering accent stayed visible after switching between
By Ordered List and By Access Date. Framebuffer samples against the
closed-menu baseline showed the shadow fading into the workspace to the
right and below the popup. The Wayland capture was also checked visually.

The native build used `50e184c49` and existing local GUI filesystem
edits. Wayland used nested Weston 13 and X11 used Openbox, with Xvfb
and Vulkan Lavapipe on both. The published input hashes matched before
and after the runs. Issue #3978 tracks strict build and runtime provenance.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)
