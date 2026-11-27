# Asset selectors in narrow docks

At 780 × 580 on Linux X11, the asset browser now sizes its dropdown after
reserving room for the swatch and caption. The previous build clipped the
Appearance caption at the window edge. The native regression checks both
captions inside their dock bounds and opens each catalog with the X11
pointer. Long asset names can still truncate in the closed preview.
[Appearance before](appearance-before.png) ·
[Appearance after](appearance-after.png) ·
[Material after](material-after.png) ·
[Appearance catalog](appearance-choices.png)

The Normal ribbon geometry check and all six filtered GUI asset tests
passed. The published docked-browser scenario passed all 21 steps again.
[Published scenario](appearance/cucumber.json) ·
[Scenario and fixture hashes](current-inputs.sha256)

The build used revision `b0f317f92` with the picker changes and pending
local filesystem and Ribbon Customization edits.
