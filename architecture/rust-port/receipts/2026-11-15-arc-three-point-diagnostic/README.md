# Three-point arc diagnostic

The `arc-three-point-0` filter ran against application commit `5025f5c23` and
pinned scenario snapshot `e76cb749ed43c73a91c24854057da7c890041f51`. The feature
SHA-256 was `8903f946c37beaed357b59f483346bab8d5d72af3ae86afafc6a8a99ca7dfc44`.
The source checkout remained clean and unchanged.

Cases 01–04 passed in diagnostic mode. Cases 05–09 were capture-only and were
not executed. The report records individual steps and exclusions; the filter
did not cover the rest of the feature.

The run used Xvfb, Lavapipe, and frame-loop input. It is not verified BDD or
OS-level input evidence. Native build provenance remains open in issue #3978.
