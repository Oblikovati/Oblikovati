# Complete Navigation Wheel reruns

Both sources passed all 12 scenarios and 143 steps with X11 input, Openbox,
Xvfb and Vulkan Lavapipe. The two test processes started from application
commit `5e9cad7d0c01192457f348fda856b6f0fb0b7724`.

| Source | Feature SHA-256 | Native report |
| --- | --- | --- |
| Pinned | `9793d6a9ad9655ef4412295075dfbf9f6f5a5f97566e5564f2ca9f5a9c68f62a` | [Cucumber](pinned/cucumber.json) |
| Published | `8b6164be54d7a3bd307b4de24d7750079ce4d23b9ada2e6fec7a3b83149e2cfa` | [Cucumber](published/cucumber.json) |

Each report matched every source step and passed all cleanup hooks.
The pinned checkout stayed at `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The published source and fixture matched their [recorded hashes](published-inputs.sha256)
before and after the run. The pinned run used the
[repository fixture](pinned-fixture.sha256), which also matched before and after.
Both replays captured the active Part after the wheel closed.
See the [pinned capture](pinned/repeat.png) and the
[published capture](published/repeat.png).

The [earlier baseline](../2026-11-21-viewport-navigation-wheel-r0-3881/README.md)
has the Go comparison and the open wheel capture. Strict build and runtime
provenance remains open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
