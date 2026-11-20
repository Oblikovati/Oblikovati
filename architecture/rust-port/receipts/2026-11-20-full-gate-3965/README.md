# Full gate attempt

On 20 November 2026, the application at commit `0c6943c4a` ran
`env -u XMODIFIERS VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json make gate-xvfb BUILD_CACHE=off`.
The user's modified `README.md` and untracked review file remained in the
checkout. The command exited with status 2 at the released dependency check.

The application locks `oblikovati-api` version `0.159.0` to commit
`57ae73ca46bfcadcc09e3ec1d0c424d803cadc44`. The gate requires the released
`v0.159.0` commit `f0943c724a6e07ec49533769c9e680137bb3d55c`. The latest
published API release is still `v0.159.0`, and the application's commit is not
an ancestor of the API `develop` branch. Issue #3965 tracks the release and pin.
The gate stopped before workspace tests, coverage, and repository checks. It is
not a passing gate.

A separate `python3 -B -m unittest discover -s tools/bdd` run passed all 342
tests. The test log contains expected errors from a deliberately corrupted Git
fixture. This separate run does not replace the full gate.
