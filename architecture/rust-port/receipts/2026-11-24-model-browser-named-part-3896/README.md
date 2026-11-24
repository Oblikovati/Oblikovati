# Model Browser cases retain the named native Part

The published nested-sketch case stopped at the
[unbound fixture step](before/published/nested/cucumber.json). The grouped
Find case opened the named file, then its second setup step opened a
generated replacement. Both cases now retain the named Part and resolve
the extrusion and sketch rows from its browser state. The sketch case
uses the existing Visibility action to prepare the hidden profile before
checking whether selection calls it into view.

| Native scenario | Published | Pinned |
| --- | --- | --- |
| Inspect a nested sketch from the model browser | [11 steps passed](after/published/nested/cucumber.json) | [10 steps passed](after/pinned/nested/cucumber.json) |
| Group search criteria for the same feature property | [17 steps passed](after/published/find/cucumber.json) | [16 steps passed](after/pinned/find/cucumber.json) |

The runs used application base `76796031d` with the local harness changes
and existing local file-browser edits on Linux X11 with Xvfb, Openbox,
and Vulkan Lavapipe. The [published input snapshot](snapshot.json) includes
the native Part, and the pinned source was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature-wide inventory fell
from [149 unresolved steps](before/published-bindings.json) to
[127 unresolved steps](after/published-bindings.json) because other cases
also use the newly bound fixture step.

See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
