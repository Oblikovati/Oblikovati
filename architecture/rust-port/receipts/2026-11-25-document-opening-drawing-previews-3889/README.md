# Native drawing previews and opening

Current Document Opening cases four and five passed all 25 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. Both cases used writable copies of
the published Part and two distinct Drawing packages.

Case four saved the sheets through Save As using Active Window and Capture.
Selecting a drawing in the Open dialog switched its filename, preview,
metadata, and options. Cancelling opened no drawing. The initial run found
that file inspection completed while Home was active, but the desktop omitted
the result when no document was active. With the result available in the Home
state, the saved sheet preview appeared in the picker.

Case five placed a Base View of the staged Part on each sheet and saved each
drawing. Reopening Drawing A restored projected sheet geometry and its model
browser view while keeping the original Part tab available.

The runs used application base `5b9a9ba93`, the corresponding changes in
this commit, and existing local GUI filesystem, opening and saving edits.
The published [input checksums](current-inputs.sha256) matched the isolated
copies. See the [inventory](scenario-inventory.json) and per-case Cucumber
and JUnit reports. Issue #3978 tracks strict build and runtime provenance.
