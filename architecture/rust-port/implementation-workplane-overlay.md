<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Work-plane overlay

The desktop presents healthy, visible authored work planes from the current
part replay. It converts the saved display corners through each evaluated plane
frame and the document's working scale. Fill and border colors come from the
active theme. A selected browser feature uses the selected-plane border color.
A bad display rectangle is reported in the window title and does not hide other
healthy planes. The reference overlay is separate from solid meshes and does
not change picking or model topology.

The overlay is hidden during sketch editing. It refreshes as the current part,
history, plane visibility, selection, or theme changes. It uses depth-tested
Martensite structures and removes them when the view closes. This first part
view implementation does not add viewport picking of planes or assembly datum
presentation. The public `workFeatures.setVisible` route remains blocked by
the cross-document identity issue #3983; see #3982.

The native renderer tests check blending, solid occlusion, overlay removal, and
validation. The native head tests create a plane from the ribbon and compare
its projected center before and after creation, undo, redo, save and reopen.
The head tests inject frame-loop input. OS-level input and the full gate remain
open. These test results are diagnostic, not verified BDD receipts (#3978).
