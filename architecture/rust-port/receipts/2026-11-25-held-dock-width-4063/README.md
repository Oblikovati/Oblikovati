# Viewport width during held splitter drags

The previous build could reduce the viewport to 164 px in a 1440 px client
while the Appearance splitter was held past its cap. The width limit now
applies before each dragged frame. Both side splitters moved more than 40 px and
kept at least 792 px for the viewport while held past the cap. Moving the
pointer back without releasing restored both viewport boundaries within
1 px. The captures show the active splitter and the visible Part.
[Model splitter](model-splitter-held-at-cap.png) ·
[Appearance splitter](appearance-splitter-held-at-cap.png)

The native restart and asymmetric workspace resize checks also passed,
along with 74 filtered GUI workspace tests. Both published splitter
scenarios passed using the shared native Part.
[Shared boundary, eight steps](shared-boundary/cucumber.json) ·
[Appearance, 13 steps](appearance/cucumber.json) ·
[Scenario and fixture hashes](current-inputs.sha256)

The builds used Linux X11, Xvfb, and Vulkan Lavapipe at revision
`2820b737a` with the held-width change and pending local filesystem and
Ribbon Customization edits.
