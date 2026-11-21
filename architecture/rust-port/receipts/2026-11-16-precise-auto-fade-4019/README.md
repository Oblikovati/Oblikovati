<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Precise Input Auto Fade diagnostic (#4019)

The pinned `move-22` scenario passed nine native frame-loop steps on snapshot
`e76cb749ed43c73a91c24854057da7c890041f51`. It enables Auto Fade while pinned,
checks the reduced controls, restores the toolbar by hovering over Y, reopens
Settings, and disables Auto Fade. The focused native editor test also checks
that the pin position is unchanged and text input remains active when the
pointer moves away. The prior pin-drag regression and GUI unit test passed.

The reduced and restored captures were reviewed. The faded toolbar retains its
expanded width so the restored controls stay reachable; this is an
implementation choice, not a width requirement from the scenario. It retains
numeric inputs and the window close control. No restart persistence or exact
fade delay is established by the scenario.

The full pinned Move diagnostic in `full/` passed 10 of 27 scenarios; 17 failed
or stopped at undefined steps. It includes successful pin and Auto Fade cases.
A failed step stops its scenario and may hide later gaps. The source checkout
was clean before and after the run. The application test tree had local changes.

All native runs used frame-loop input on Linux with Xvfb and Vulkan Lavapipe.
They are diagnostic: OS-level input and bounded build provenance (#3978) remain
unverified. The full `make gate` was not run.
