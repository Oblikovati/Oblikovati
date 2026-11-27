# Capturing the saved drawing sheet

Active Window capture now renders the current drawing sheet and its projections
using the same drawing renderer as the document workspace. Rendering offscreen
keeps the Save As controls out of the captured image.

Both current drawing scenarios passed again with the stricter assertion that
both saved preview payloads contain paper pixels. The first scenario captured
and reopened saved sheet previews after returning to Home. Its
[Open dialog capture](saved-drawing-sheet-in-open-dialog.png) shows the sheet in
the picker. The second scenario also saved real base views and reopened a
drawing beside its Part. The pinned Part active-window capture scenario passed
all ten steps on the same application build.

The runs used application base `509928314`, the changes in this commit, and
existing local GUI filesystem, opening and saving edits. The published
[input hashes](current-inputs.sha256) matched the isolated copies. Issue #3978
tracks strict build and runtime provenance.
