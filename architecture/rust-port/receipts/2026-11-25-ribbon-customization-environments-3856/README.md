# Marking Menu environments

Current case 118 passed all 10 steps in the native X11 window. It opened
the environment selector with 2D Sketch selected, checked the seven
published labels, dismissed with Escape, and verified that the profile
selection and pending state were unchanged. Part opens its own profile
by default, and an active 2D sketch opens the 2D Sketch profile. The five
other labels were visible as disabled choices in this build.

A separate native test staged Fillet in Part and Rectangle in 2D Sketch.
The running profiles kept their original assignments until Apply. One
Apply saved both assignments, and restarting loaded both. The existing
native single-slot Apply and restore test also passed.

The build used base `c273e6447` with the environment editor changes and
existing local GUI filesystem, document-opening, and document-saving
edits. The run used a writable copy of the published Normal preferences.

[Cucumber](case-118/cucumber.json) · [JUnit](case-118/junit.xml) ·
[Scenario arguments](case-118/scenario-arguments.json) ·
[Source revision](case-118/source-revision.json)
