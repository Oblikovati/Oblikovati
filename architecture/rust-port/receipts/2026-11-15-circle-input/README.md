# Center Point Circle diagnostic

Application test revision: `fc7dc4ea5` (the run used the same local edits before commit). Pinned scenario snapshot: `05317bbb2d0809a532b29b7ff6f726a5722b865f`, verified before and after the runs. Xvfb used Lavapipe and `XMODIFIERS=@im=none`.

The full circle run passed 8 of 9 scenarios and 68 of 69 executed steps. The remaining scenario, circle-center-point-10, stops at the zero-radius observation; later steps were not run. The setup enters the recorded center through Precise Input, then moves the pointer to the nearest available screen pixel. The HUD reads `0.491 mm`; the source does not specify the input method, pointer state, field, or view alignment for its `0.000 mm` observation. A clarification is in the feature's `BLOCKERS.md`. The displayed value was not changed or treated as a pass.

The retained evidence also includes a passing seven-step invalid-coordinate Precise Input scenario. The circle-08 setup constructs its recorded geometry through Precise Input before undo and redo; it does not establish that its original circle was mouse-created. Screenshots for circle-10 and circle-11 were reviewed. The original baseline stopped in five scenarios, four at the old Precise Input interaction and one at an unbound shortcut step.

These are frame-loop input diagnostics, not OS-input acceptance. Build and runtime provenance remains open in #3978, so they are not verified receipts. Tracking: #3910.
