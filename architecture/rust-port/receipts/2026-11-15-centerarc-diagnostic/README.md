# Center-point arc diagnostic

The focused `arc-center-point-01` run passed all seven steps on application
commit `21d9cff4f` and pinned scenario snapshot
`e76cb749ed43c73a91c24854057da7c890041f51`. The feature SHA-256 was
`35d69dce04958b8bb0b086450fce097d2cc8c99ca367600311c1b6284599469f`.
The source checkout remained clean and unchanged.

The run used Xvfb, Lavapipe, and frame-loop input. It is a focused diagnostic,
not verified BDD or OS-level input evidence; native build provenance remains
open in issue #3978. Capture-only cases 06–14 were not executed.
