# Home Recent at short window heights

On Linux X11 with Xvfb and Vulkan Lavapipe, the native regression seeds
24 entries, resizes the client to 1454 × 540, and scrolls through Recent
with the mouse wheel. It measures from the final filename to the top of
the document tab strip. The previous layout measured 12 px. Adding 12
logical px of scroll-end padding increased the measured clearance to
28 px, including the layout's item spacing. List and Grid share the
padding. [Previous frame](before.png) · [Updated frame](after.png)

The published Home navigation scenario passed all ten steps with an
active sketch and the Part tab available. The published minimum-frame
scenario passed all six steps on Openbox with tint2, including visible
window controls, taskbar restore, and the same Part at 640 × 480.
[Home navigation](home-navigation/cucumber.json) ·
[Minimum frame](minimum-frame/cucumber.json)

The build used base revision `512a7f6d8` with the Home padding and native
test changes, alongside local filesystem and Ribbon Customization work.
The original Windows clipping observation still needs a Windows retest.
