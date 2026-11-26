# MCP sketch rectangle and profile wiring

This addresses the `sketch.rectangle` and `sketch.profiles` findings in the
2026-11-26 MCP Bridge live report for Oblikovati.API v0.166.0. These missing
routes were the first failure in 12 and 9 of the 118 modeling cases,
respectively. A case can still fail further downstream.

The rectangle command resolves both dimensions in the active document's
parameter context and creates a closed polyline from the sketch origin. It
computes the profile count after solving the candidate, inside the same
transaction. A failed solve or failed region certification rolls the edit back.

The profile query returns certified bounded regions in extrusion selection
order. It reports areas in square centimetres, subtracts holes from the
surrounding region, and also lists an enclosed cell when it can be selected
independently. Open chains and curves outside the current certified region
admission remain gaps.

The desktop owner regression exercises the released typed API, including an
empty sketch, rectangle dimensions from an expression, a hole, a disjoint
region, rejection of a coincident rectangle, and extrusion by a listed index.

A disposable native Wayland host with the API v0.166.0 MCP Bridge also
accepted both routes over loopback HTTP. The bridge returned the structured
profile payload and the selected perforated profile produced the expected
volume, `(12 − π/4) × 5,000 mm³`. The three baseline bridge integration
scenarios also passed on the rebuilt host.

## Live modeling rerun

The Wayland rerun completed all 118 cases across two disposable hosts. It
passed seven cases, up from four in the baseline. Spacer, hex nut, and
polygon ring now passed their analytic volume checks. All four previous
passes held, and zero cases stopped because either new sketch route was
unavailable.

Cut is now the first reported failure in 25 cases, and Join in 19. Both
need certified solid replacement and durable body lineage during replay.
Other feature kinds, sketch offset, command execution, and the remaining
wire namespaces are still open.

The PCB case still stops at profile discovery. A diagnostic retry exposed
`BuildPlanarRegions` at certification with `Joins: PairUnavailable` for
`RegionSourceId(5)` and `RegionSourceId(7)`. The sketch reported zero
degrees of freedom and its patterned hole moved from 3.6 cm to 5.6 cm
when the board width changed from 40 mm to 60 mm. The driver summarizes
the kernel refusal as “no board-with-holes profile.”

## Shared subtraction for Cut replay

The Oblikovati Martensite adapter now uses a detached subtraction path for
Hole. It takes the tool shape and its names, checks that the Boolean produced
one valid solid with lower analytic volume, applies the existing tolerance
limits, and carries generated, modified, and deleted entities into the
existing feature naming pass. Hole still builds its cylinder and supplies
its own names. Shell uses the same tolerance helper it used before.

The square-prism probe produced one geometrically valid 6 cm³ solid with
two through-openings and left the 8 cm³ source unchanged. The existing Hole
tolerance rule refused that result because the largest vertex tolerance grew
from 1.0 nm to 1.5 nm. The regression records both outcomes. The Cut API
still needs a two-operand tolerance contract, a persisted target body
reference, and ordered replay that replaces the target.
