<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Move zero-vector recovery diagnostic (#4020)

The pinned `move-12` scenario reached the same base and endpoint through the
native frame-loop runner on Linux, Xvfb and Vulkan Lavapipe. Five steps passed;
the next step failed because the rendered message did not refer to Constraint
Settings for Sketch Edit Tools. The status said `Command failed: move vector
[0, 0]: expected distinct base and end points`. Move remained at Select end
point and had no preview. The capture was reviewed. The scenario stopped at
that failure, so the later Cancel and policy checks were not executed.

`diagnostic.patch` records the temporary test bindings used for this failed run;
they are not acceptance bindings. The first failure step records the current
specific status for diagnosis, not a requirement that future wording match it.
The scenario source is the local pinned snapshot
`e76cb749ed43c73a91c24854057da7c890041f51`. The application test tree was
uncommitted. This is not OS-level input or verified acceptance evidence (#3978).

The current public `sketch.transform` request has no policy or prompt-decision
fields. Sketch settings expose broad relax-mode booleans, but the API excludes
selective geometric constraint removal. The session transform solves the
existing constraints without this policy. API issue #304 tracks the contract.
Go `app/sketch_transform_tools.go` has a Move tool but no matching policy UI.
Do not treat a displayed Prompt label as implemented constraint handling.
