# Complete Document Tabs reruns

The native test processes started from application commit
`fc44e5d73ead0b7a6f8fccdd79ded35947152239` with Xvfb, Openbox and
Vulkan Lavapipe. Both runs include the native X11 [reorder checks](../2026-11-21-document-tabs-reorder-3900/README.md).

| Source | Feature SHA-256 | Passed scenarios | Passed steps | Report |
| --- | --- | ---: | ---: | --- |
| Pinned | `2738bff62d9968e32a59d426e9a3e762938981a2bdb7e074f77bdf59c5d4928f` | 38 | 334 | [Cucumber](pinned/cucumber.json) |
| Published | `ee2022e13728cfe8fcda1ef61e60b25f01c308816e6860da91b6dceac3430de9` | 39 | 343 | [Cucumber](published/cucumber.json) |

The pinned checkout stayed at `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The published [feature and support hashes](published-inputs.sha256)
matched before and after the run. Each run produced one final PNG for
every scenario and passed its cleanup hooks. The retained
[pinned](pinned/separate-views.png) and [published](published/separate-views.png)
captures show the final independent-view scenario.

The [earlier baseline](../2026-11-21-document-tabs-r0-3900/README.md)
has the Go comparison. Strict build and runtime provenance remains open
under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
