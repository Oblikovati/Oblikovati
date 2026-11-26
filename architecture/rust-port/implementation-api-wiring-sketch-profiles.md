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

The live report's cut and join extrusion, other feature operations, sketch
offset, command execution, and remaining wire namespaces are tracked
separately from this change.
