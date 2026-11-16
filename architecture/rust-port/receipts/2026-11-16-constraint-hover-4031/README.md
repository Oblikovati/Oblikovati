# Constraint glyph hover diagnostic

This is a diagnostic, not a verified acceptance receipt. The source was
unversioned, absent from the pinned scenario snapshot, and tagged
`needs_fixture`. Its SHA-256 was
`5d30dc6745bfc4bf6415c21558d7d6ff3cae1df3b50862583a87ebd4e38f2926`
before and after the run. Build provenance and receipt reuse remain open in #3978.

The native window test passed one scenario and six steps under Xvfb and
Vulkan Lavapipe. It creates the fixture with host commands and moves the OS
pointer with XTEST. It compares pixel neighborhoods on both lines and checks
the constraint identity after hover. Separate renderer tests verify the
candidate color and that the overlay clears or follows geometry updates.
The compressed run log preserves its original bytes. The screenshot was inspected; its default theme gives subtle visual contrast.
