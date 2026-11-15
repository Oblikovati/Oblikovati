# Model browser body visibility diagnostic

Application test revision: `55b5b2c0b`. Scenario snapshot: `05317bbb2d0809a532b29b7ff6f726a5722b865f`. The restored snapshot was verified before the runs.

The original body run stopped at the blue-face check. The retained screenshot shows a blue face while the previous check counted only pixels near the brighter edge color. The revised check requires a blue filled patch outside application windows; its unit cases reject an outline, an occluded fill and a neutral face. Eight visibility tests passed.

At the test revision, the body scenario passed all 13 steps and the Solid Bodies folder scenario passed all 11 steps. Their reports and final screenshots are retained here. These runs inject input through the native window frame loop, not the operating system. The initial screenshot was reviewed for the highlighted face; the final body screenshot was reviewed for the restored shaded body. The full model-browser feature and OS-input journey were not run.

These are diagnostic observations, not verified acceptance receipts: build and runtime provenance remains open in #3978. Tracking: #3896.
