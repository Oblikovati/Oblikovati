# Temporary ribbon selected-tab behavior

The focused scenario passed all 11 steps on Linux with Xvfb, Openbox and Vulkan
Lavapipe. It used the scenario hash
`645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2` and code
committed as `617b2317d`. A separate X11 mouse test passed. Focused regressions
also passed for outside-click dismissal, Escape and full-ribbon restoration.

This report is not a full feature run and does not update the inventory count.
