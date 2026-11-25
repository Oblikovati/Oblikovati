# Isolate detached drawing from concurrent GUI unit tests

The GUI unit suite at `ceaceb153` failed on the template chooser's first
frame with texture ID zero. The chooser passed on its own. Running it
with the detached-drawing DPI test on two threads failed on all six
baseline attempts. Failures included an occupied context, a missing
current context, and segmentation faults.

The detached-drawing context guard now takes the shared test lease before
reading or parking the active ImGui context. The guard restores the
primary context before releasing the lease. This covers detached drawing
creation, rendering, destruction, and sheet previews in GUI unit tests.
The lease is compiled into test builds.

The contention regression holds the lease while another thread attempts
to create a detached drawing. It verifies that creation waits for the
lease and succeeds after release. The [test results](test-results.txt)
also record the parallel pair, full GUI unit suite, and context ownership
checks. The build used `ceaceb153` with this change and existing local
filesystem edits.
