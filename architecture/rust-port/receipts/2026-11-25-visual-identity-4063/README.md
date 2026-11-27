# Oblikovati visual identity on the native desktop

The refactor was exercised on Linux with a private X11 display, Openbox,
and Vulkan Lavapipe. The build started from `19bfb6d47` and included the
local visual-identity changes. The worktree also contained unrelated local
document file and ribbon customization changes.

## Handoff inputs

| File | SHA-256 |
| --- | --- |
| `Design/tokens/design-tokens.json` | `f21ff3b4547fd0ea7e0586eefcd36150e147bb245b10fc4f795c964bf1894a06` |
| `Design/tokens/oblikovati-tokens.css` | `3b89a79f7fc73bafb2791a7d3a5da8f06c1cd684642f764a6734f033d7da6a5e` |

## Navigation scenarios

The native runner clicked the navigation controls using their rendered glyph
positions. The three current scenarios passed all 38 steps.

| Scenario | Steps | Receipt |
| --- | ---: | --- |
| Fit a distant block using Zoom All | 11 | [Cucumber](zoom-cucumber.json) |
| Select and exit constrained orbit | 11 | [Cucumber](orbit-cucumber.json) |
| Open the full navigation wheel and close it from its menu | 16 | [Cucumber](wheel-cucumber.json) |

The scenario runner used the published MCAD-Scenarios directory with the
fixture arguments preserved alongside each result. These results cover the
navigation refactor before the final handle and Extrude preview colors.

The final build passed all 15 steps in
[Reduce the full ribbon to compact group buttons](ribbon-cucumber.json) and all
10 steps in [Open the file menu while a part is active](file-menu-cucumber.json).

## Native captures

[Dark](Dark.png), [Light](Light.png), and [High Contrast](High-Contrast.png)
show the same Part on the three studio backdrops. The native test sampled the
navigation surface and dock tab after each palette switch.

[Extrude](Extrude.png) shows the two property sections, aligned fields,
teal preview, and action footer. The final native tests covered a disabled
OK on invalid distances, recovery with valid distances, and preview updates
after direct and parameter-driven edits.

## Additional checks

- The GUI library suite passed 808 tests.
- The selected model, automation, and persistence suites passed 840 tests.
- Workspace Clippy passed for all targets and features.
- The workspace gate reported 2,183 scenarios missing required R0 ledger
  evidence. Its coverage stage was still running when this receipt was written.
