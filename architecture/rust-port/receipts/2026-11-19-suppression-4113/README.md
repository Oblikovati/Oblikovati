# Model-state suppression check

The focused native scenario passed all 11 steps on 2026-11-19. The tested
worktree was committed as `627a847cd`. X11 input switched between model states;
the scene contained zero visible bodies when suppressed and one when computed.
The dock-tab occlusion test also checks that a real model-colored patch is
still detected.

This is a focused result, not a full-feature run or verified build provenance.
See issue #4113 and issue #3978.
