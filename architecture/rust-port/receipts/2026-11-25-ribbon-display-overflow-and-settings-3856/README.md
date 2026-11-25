# Ribbon overflow, compact groups, and Document Settings

Nine current Ribbon Display scenarios passed all 96 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The View checks use the visibility
command implied by the saved Part's sketch state. The runs verified overflow
selection and dismissal, compact panel changes, and a return to the full
ribbon after maximizing. Document Settings opened from both the compact and
full Tools ribbons. The published solid also stayed available for the Options
panel dismissal check.

The matching builds included existing local GUI filesystem, opening and saving
edits. The published [input checksums](current-inputs.sha256) matched the
isolated copies. See the [scenario inventory](scenario-inventory.json) and
individual reports. Issue #3978 tracks strict build and runtime provenance.
