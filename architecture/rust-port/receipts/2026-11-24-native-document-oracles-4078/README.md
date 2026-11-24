# Native drawing and assembly checks

Issue #4078. Both regressions reproduced at `db8c2a8b9` on Linux X11 with
Xvfb, Openbox, and Vulkan Lavapipe.

- The saved drawing opened on Sheet 2 of 2. A fixed framebuffer coordinate
  intended to read the workspace reached the portrait sheet instead. The
  test now measures the rendered paper along the document viewport's
  centerlines and checks both its surrounding workspace and the A3
  portrait ratio of 297 to 420.
- The saved assembly opened with its two visible occurrences. The test
  then stopped while looking for a text run for Select. It now clicks the
  rendered toolbar icon and uses the same Faces and Components choices,
  occurrence selection, and visibility checks.

Both native regressions passed with these changes. Clippy passed for the
desktop crate and its test targets.

**Open an assembly in express mode and load it fully** passed 13 pinned
steps, including rendering the component and activating Load Full. The
final step failed because the Assemble ribbon did not show Finish.
See the [Cucumber diagnostic](pinned-express/cucumber.json).

The pinned source still marks **Create a drawing from a sheet template**
as capture-host-only. The native portrait regression uses an authored
saved drawing and runs independently of that fixture.

## Read the stale assembly cache warning in the Log Panel

The assembly save regression at `cc07df1d1` raised the expected stale-cache
warning and then failed while looking for the workspace's standalone `+`
launcher. With Model docked, the panel menu is an icon in its tab bar. The
test now opens Log Panel through the existing workspace driver and still
checks the complete warning against the rendered log.

The standalone assembly save test passed. Its executable had debug sections
stripped to shorten hashing of the running image. The test still verifies the
saved producer digest against that executable. Strict Clippy passed for all
desktop targets.

The pinned **Add and switch between browser panel tabs** scenario passed all
13 steps on the same test build with Xvfb, Openbox, and Vulkan Lavapipe. It
opens Log beside Model, switches between both tabs, and closes Log. The
workspace-layout feature stayed at SHA-256
`72f20c5bcf466b9ee01a4d3a547a4412ed9a1ae243c07f256a027e237ed2e554`.
See the [Cucumber result](pinned-workspace-log/cucumber.json) and
[source revision](pinned-workspace-log/source-revision.json).
