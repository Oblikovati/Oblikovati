# Native Ribbon Panel Visibility fixture and selector checks

Eleven current cases passed 126 steps after the published blank Part setup
was connected to the existing native Open dialog binding. The cases cover
ordinary, compact, minimized, side-docked and floating ribbon states,
the pinned panel expansion, and selector dismissal.

The Linux X11 test build used Xvfb, Openbox and Vulkan Lavapipe. It was based
on `626b53a8b` and included the fixture wording used here, a selector wording
alias, an unused draft sketch alias, and existing local GUI filesystem,
opening and saving edits. The [input hashes](current-inputs.sha256) matched
the isolated source and fixtures. Issue #3978 tracks strict build and
runtime provenance.
