# Native document tab preview checks

Issue #4078. On `0b9370a01`, the Home preview contained visible detail near
the top of its thumbnail. The standalone regression sampled the lower
portion of the image and reported a blank preview. It now uses the pinned
scenario's check, which starts below the rendered caption and samples the
picture interior. The shared check has a separate blank-image regression.

The 2× ribbon-minimized regression also failed before it reached the placement
check. The default asset docks left too little width for the Part tab. The
test now starts with the saved layout that closes those docks, and still
checks the preview against the tab strip and logical viewport at 2×.

The Home, full-ribbon and 2× native preview tests passed on Linux X11 with
Xvfb, Openbox and Vulkan Lavapipe. The scaled capture shows the preview clear
of the tab strip. Strict Clippy passed for all desktop targets.

Both pinned scenarios passed. **Preview Home while a part is active** passed
six steps, and **Preview an inactive document from its tab** passed nine,
including dismissal without activation. The feature SHA-256 stayed at
`2738bff62d9968e32a59d426e9a3e762938981a2bdb7e074f77bdf59c5d4928f`.
See the [Home result](pinned-home/cucumber.json) and
[inactive Part result](pinned-inactive/cucumber.json).
