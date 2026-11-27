<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Document-opening diagnostic

On 2026-11-15, the full native document-opening feature ran on application
commit `c413f5579` and scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f`. The snapshot was verified before
and after the runs. The full run passed 26 of 27 scenarios and 273 of 274
executed steps. The Express-load scenario stopped at an unbound final step.

The focused run used the additional binding committed as `f51954224`. It
passed 13 of 14 steps and failed the final visible-and-enabled check because
the rendered assembly ribbon did not contain Finish. The existing assertion
checks Simplify after Finish, so the failure does not establish Simplify's
state. The missing commands are tracked in #3963. The retained final capture
shows the assembly and its ribbon after Load Full.

Both runs used Xvfb at 1600 × 1200 with Mesa Lavapipe. Input is injected into
the GUI frame loop. These are diagnostic results, not acceptance evidence:
native build provenance remains open (#3978), and OS-level input was not
tested in these runs.
