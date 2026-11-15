<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OS-input tab overflow check

On 2026-11-15, the standalone desktop ran on Linux Xvfb with Mesa Lavapipe at
application commit `0022d245a`. The root README had an unrelated local edit.
The executable SHA-256 was
`6fdb02dea16eb4a142307a748e9ba787b8f4499c700fcd26a692c6d373dbc86a`;
the MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.

MCPBridge created 14 unsaved part documents as setup. XTest then clicked the
left navigation arrow at `(1689,266)`, moved the pointer away, clicked the
visible first tab at `(620,266)`, and moved away again. The 1900 × 1100 window
was captured through MCPBridge at each state. `scrolled.png` shows neutral
visible tabs while LivePart14 remains active in the browser. `selected.png`
shows LivePart12 highlighted and its browser active; a read-only document list
also identified LivePart12 as active.

The scrolled capture additionally shows LivePart11 after LivePart13 instead
of before LivePart12. That ordering defect is tracked in #3993. This check
validates OS input for the highlight fix (#3992); it is not a BDD acceptance
receipt. Build provenance remains open (#3978). The application and Xvfb
were closed after the run. The retained scripts use run-specific paths and
coordinates; the document creation setup used the MCP `create_document` tool.
