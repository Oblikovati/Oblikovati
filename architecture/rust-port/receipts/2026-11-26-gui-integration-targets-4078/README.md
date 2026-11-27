# Current GUI integration checks

All 104 checks from the four groups investigated in #4078 passed on the
current checkout.

| Group | Passed |
| --- | ---: |
| Document tabs | 47 |
| Engine | 4 |
| File dialog | 4 |
| Ribbon click | 49 |

The original report named a standalone document tab test target. Its
current checks live in the GUI library, so the rerun selected
`document_tabs` from that test binary. The three standalone targets ran with
`cargo test --locked --offline -j 2 -p oblikovati-gui --test engine --test file_dialog --test ribbon_click --no-fail-fast -- --test-threads=1`.
Both runs used Xvfb and Vulkan Lavapipe at `ba720f07d` with existing local GUI
filesystem edits.
