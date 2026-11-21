<!-- SPDX-License-Identifier: GPL-2.0-only -->

# New Sketch native diagnostic

On 2026-11-16, the full pinned New Sketch feature ran in diagnostic mode on
application commit `efcd467287c7c9377c0c2a8817b3d6f584fa0ca2`. The working
branch also had an unrelated modified README. The restored scenario snapshot
was clean at `e76cb749ed43c73a91c24854057da7c890041f51` before and after the
run; the feature SHA-256 was
`e4011485e2cfe78197b73450bea0d26ba2b0dbc54f2307d3b27364ffe9d9a20d`.

Seven scenarios reported all steps passing in diagnostic mode: 01 through 05,
07 and 08. Scenarios 09, 10 and 06 stopped on unbound steps. Later step results
for each of those three are tainted. The report is diagnostic, not acceptance:
it does not bound native build provenance (#3978), establish OS-level input,
or provide reviewed captures. No verified scenario count changes.

The run used Linux Xvfb at 1920 × 1200, Vulkan Lavapipe, `XMODIFIERS=@im=none`,
a task-owned Cargo target, and the following command:

```sh
OBK_BDD_FEATURE=new-sketch OBK_BDD_DIAGNOSTIC=1 \
  cargo test --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```

The application test and Xvfb processes ended. Startup-plane preference work
is tracked in #4009 and look-at behavior in #4010. Scenario 06 (#4011) needs its
accumulated document fixture clarified in the source folder.
