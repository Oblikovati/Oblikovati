# Native Workspace Layout drawing dock and tree

Eleven published cases passed 104 steps on Linux X11 with Xvfb, Openbox
and Vulkan Lavapipe. The drawing now uses the same native drag gesture as the
Part to reach the right edge of the full workspace. With the asset dock open,
the earlier gesture ended at the interior edge of the drawing viewport. The
reruns verified both side docks, the drawing resource tree, the sheet beside
the browser, and the Part docking regression.

The Drawing tree cases also covered expand, collapse, filter, empty
results, selection, and search while floating. One Part case verified
collapse and restore of Origin while Sketch1 was active.

The side-dock checks used `04aeb4deb` plus the test changes in this commit.
The other cases used the test changes in `111dee388`. Both builds included
existing local GUI filesystem, opening and saving edits. The
[input hashes](current-inputs.sha256) matched the isolated source and fixtures.
Issue #3978 tracks strict build and runtime provenance.
