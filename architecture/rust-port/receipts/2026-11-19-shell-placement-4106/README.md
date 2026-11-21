# Shell ribbon placement

The native X11 scenarios for moving Shell between ribbon areas and retaining
its placement across tabs passed 10 of 10 steps each on 2026-11-19. The
feature SHA-256 was
`645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2`.
The tested branch was committed as `1ccfa88e8`; tests used Xvfb and Vulkan
lavapipe. Native tests also covered launching Shell from the expanded area,
showing the kernel tolerance error, and cancelling the command.

These focused results are not a full-feature run or verified build provenance.
Successful Shell creation remains blocked by Martensite #554. See issue #4106
and issue #3978.
