# Pinned Application Closing scenarios

All three scenarios ran individually against the native desktop. The dirty
part showed a save-changes prompt with Yes, No and Cancel. Cancel dismissed
the prompt and left both the part and the application open. The saved-part
case created and saved its own native part. Both the saved-part and
no-document cases exited with zero confirmation requests.

The successful clean-close tests capture the last visible frame before the
native drawable is destroyed. The dirty-part case records both the prompt
and the window after cancellation.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Cancel closing with unsaved changes | 8 | [Test report](cancel/cucumber.json), [save-changes prompt](cancel/application-closing-dirty-prompt.png) |
| Close with a saved document | 3 | [Test report](saved/cucumber.json), [before close](saved/application-closing-ready.png) |
| Close with no document | 3 | [Test report](empty/cucumber.json), [before close](empty/application-closing-ready.png) |

The scenario source stayed clean at `b576445b98cffd686dcc05aa8c017de2a591c8c8`
with feature hash `d87eaf041fc93701e0de5c911dd3b4c0622386d184f05954a8409400e412c0d0`.
The runner used Linux Xvfb, Openbox, and Vulkan Lavapipe. Application
build inputs included the existing working-tree edits to document-opening
and document-saving. These local runs support the implemented ledger state.
Strict build and runtime provenance remains open under #3978.
[Source report](source-revision.json)
