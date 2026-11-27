# Model browser search and tree styling

At 640 × 480, the Model search combines the query and its × control inside
one outline. The native X11 test types a filter, clears it with the ×,
checks that the full tree returns, then presses × again to close the field.
The old layout failed the assertion for the multiplication-sign glyph.
[Before](before.png) · [After](after.png)

The sketch row uses the same 10 px mono face as the status bar and
centers its DOF badge vertically. The dashed rollback marker uses the
strong border color. Native checks cover the badge’s size and alignment
and confirm the rollback rule still draws dashes above End of Part.
[Model tree](model-tree.png)

Six published search scenarios passed all 69 steps. The origin-axis
check now measures the blue datum overlay. It verifies that the axis
remains highlighted through clearing and closing search, and disappears
after a viewport click.

- [Part open and close](search-0.json)
- [Origin axis selection](search-1.json)
- [Replace a query](search-2.json)
- [Feature-name filter](search-3.json)
- [Floating drawing browser](search-4.json)
- [Sketch browser](search-5.json)

The GUI library’s browser filter passed 99 tests. Verification used
Linux X11, Xvfb and Vulkan Lavapipe. The build used `7223594c1` with the
browser changes and pending local filesystem and Ribbon Customization
edits. [Published input hashes](current-inputs.sha256)
