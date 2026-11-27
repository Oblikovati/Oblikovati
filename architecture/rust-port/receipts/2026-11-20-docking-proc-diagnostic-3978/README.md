# Docking restart diagnostic with private proc

The test runner and desktop were built offline inside Bubblewrap from the
recorded application commit and then pinned for execution. The selected local
scenario passed one scenario and four steps with native pointer input. The
runtime mounted a private `/proc` for desktop worker discovery; this is a
diagnostic exception, not the sealed runtime policy.

The first launch failed because the runtime lacked a desktop mount point. Later
attempts found missing runtime libraries and the `/proc` requirement. These
failures are retained in the logs. After runtime repair, the scenario passed;
the report check initially failed because it dropped the screenshot folder.
The corrected checker passed against the retained report afterward.

This is not verified evidence: the runtime and build inputs were mutable and
unauthenticated, the Git vendor checks were incomplete, and the runner was
retried separately from the build observation. Issue #3978 remains open.
