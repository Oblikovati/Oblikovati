# Center-point circle diagnostic

The focused native diagnostic used application commit `a55445b6f` and pinned
scenario snapshot `e76cb749ed43c73a91c24854057da7c890041f51`. The feature
SHA-256 was `a18dcad10755d0ceb8b49b1d907b3800b1c6847c8b3548eb5d2535487b2be445`.
The pinned checkout was clean and unchanged during the run.

Scenarios `circle-center-point-01`, `-06`, `-07`, `-08`, and `-09` passed in
diagnostic mode. The filter was `circle-center-point-0`; it did not run the
other scenarios in the feature. The report records each executed step.

The `-08` fixture creates its circle with precise input, although the scenario
title calls it mouse-created. Its measurement steps passed, but the fixture
does not establish that part of the premise. The scenario folder asks for
clarification of the intended input and recorded view.

The run used Xvfb, Lavapipe, and frame-loop input with `XMODIFIERS=@im=none`.
It is not verified acceptance or OS-level input evidence. Native build
provenance remains open in issue #3978. Circle scenario 10 has a separate
source clarification request in its feature folder.
