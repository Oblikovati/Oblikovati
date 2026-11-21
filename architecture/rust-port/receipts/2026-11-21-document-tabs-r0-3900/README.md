# Document Tabs full native run

Linux X11 with Xvfb, Openbox and Vulkan Lavapipe ran both feature versions
on application commit `d2755f6d5673f02d4389f847fc3845777f9cd58d`, API v0.165.0, and Martensite v0.13.1.

| Source | Scenarios passed | Steps passed | Report |
| --- | ---: | ---: | --- |
| Pinned | 38 | 334 | [Cucumber](pinned/cucumber.json) |
| Current | 39 | 343 | [Cucumber](live/cucumber.json) |

The pinned source was restored from snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and remained clean. Its feature SHA-256 was
`2738bff62d9968e32a59d426e9a3e762938981a2bdb7e074f77bdf59c5d4928f`. The current feature SHA-256 was
`ee2022e13728cfe8fcda1ef61e60b25f01c308816e6860da91b6dceac3430de9`. The current [feature and support hashes](live-inputs.sha256)
matched before and after the run.

The scenario setup generates native Part and drawing inputs. The cases
cover tab and Home navigation, document state and preview,
close and save prompts, overflow and tab order, separate drawing windows,
and independent views of one Part. The standalone tests also pass for
closing and restoring a detached drawing, preserving its tab when the main
strip is hidden, and [previewing Home](home-tab-clean-preview.png). The
drawing-detachment tests use Openbox and native X11 events. The earlier [separate window
capture](../2026-11-20-native-drawing-window-3900/README.md) records the
window manager tree and the distinct drawing and Part windows.

To repeat the pinned run, set `OBK_BDD_FEATURE=document-tabs`, select
Lavapipe with `VK_DRIVER_FILES`, start Openbox on the Xvfb display, then
run the test below. To run the current source, also set `OBK_BDD_ROOT`
to the MCAD-Scenarios directory.

```sh
cargo test --locked --offline -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The [Go reference tests](../2026-11-14-go-tab-baseline/README.md) cover
history isolation, background close and tab-label identity. The native
Rust runs check the complete document switching and window journeys.
All pinned declarations have R0 baseline evidence. Strict verification
remains open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
