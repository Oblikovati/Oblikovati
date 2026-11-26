# Sweep Cut and a cancelled no-op edit

The published `sweep-14` scenario joins a circular boss to the block,
reopens the Sweep and changes its operation to Cut. The swept tool
touches the original block and removes no material. OK opens Edit Sweep
Feature with a collapsed Details section. Expanding it shows the
unchanged face-count message. Cancel preserves the joined boss, one
body, and the original history generation.

| Backend | Scenario evidence |
| --- | --- |
| Wayland on COSMIC | [7 steps](wayland/sweep-14/junit.xml) |
| X11 under Xvfb and Lavapipe | [7 steps](x11/sweep-14/junit.xml) |

The [input hashes](current-inputs.sha256) identify the published
scenario and the native fixtures.

The published `sweep-15` fixture declines differently. Martensite
refuses the smaller circular profile in its translated plane during
planar-region certification, before Sweep can evaluate the Cut. The
same local circle certifies at the world origin. The minimal kernel
reproducer and tested proof versions are in
[Martensite #619](https://github.com/Oblikovati/Martensite/issues/619).
