# Full-width Home at startup

Model, Material Browser, and Appearance Browser are suspended while Home
is active, including when the app starts without a document. Their saved
visibility and docking preferences are applied when a compatible document
is active.

The native X11 check verified the full workspace width, the New Part
action, and the return of the Model and shared right asset dock after
creating a part. [Startup capture](startup-home.png)

The published startup observation expects the empty Model and asset
browsers to remain visible. It fails at the first of those expectations
with this change. Its Get Started and New Part checks still pass.
[Legacy startup scenario result](legacy-startup/cucumber.json)
