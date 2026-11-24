# Empty Part viewport and model-state captions

The [pinned reproduction](before/pinned/representations/capture.png)
shows the fixed background rectangle extending into the docked
Appearance browser. The check now derives its rectangle from the
document viewport and verifies that the Part has no bodies, features,
or sketches. The [gradient tests](gradient-tests.log) also passed.

Both published cases keep the named blank metric Part active. The
[first model-state run](before/published/states/cucumber.json) revealed
that the caption check assumed the document was called Part1. The
updated check locates the browser root and document tab independently
and reads the active member in each caption.

| Native scenario | Pinned | Published |
| --- | --- | --- |
| Inspect representation branches in an empty part | [17 steps passed](after/pinned/representations/cucumber.json) | [18 steps passed](after/published/representations/cucumber.json) |
| Create and switch model states in an empty part | [19 steps passed](after/pinned/states/cucumber.json) | [20 steps passed](after/published/states/cucumber.json) |

The passing runs used application base `e7994401f` with local test
changes and the existing file-browser edits on X11, Xvfb, Openbox, and
Vulkan Lavapipe. The pinned cases gained two R0 records. The
[published snapshot](../2026-11-24-model-browser-named-part-3896/snapshot.json)
contains the named blank fixture. Its [binding inventory](published-bindings.json)
lists 99 unresolved steps across Model Browser, down from 127.

The standard metric document-creation scenario uses the same assertion.
Its source is tagged `@capture_host_only` and the local runner excluded it.
See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
