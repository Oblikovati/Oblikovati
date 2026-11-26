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

## Cut with a single target body

The typed `features.add` Extrude request now accepts `operation: "cut"`
for a part with one current solid. The saved extrusion stores that solid's
exact reference and adds its producer to the feature's dependency footprint.
Replay builds a certified tool from the selected profile, resolves the saved
body through earlier replacements, and publishes the surviving solid only
after validating geometry, tolerance, and topology names.

The Cut result must meet the tighter of the target and tool tolerance caps.
Each operand is checked against its own cap. Cut refuses an empty result,
a split result, or a tool that removes no volume. Join and the Through All
extent still require separate integration.

The checked run passed all 32 desktop-owner extrusion tests, 14 authored
recipe tests, and 252 executed adapter tests. The Cut regression checked
analytic volume and the original body key after resize, undo, redo, and
file reopen. A disposable native Wayland host also passed the v0.166.0 MCP
Bridge Cut workflow with a circular tool and a subsequent depth edit.

The 11 live cases that previously stopped at a distance Cut all reached a
result or a later blocker. Corner bracket passed on a fresh host. Carriage
end advanced to Join. Four cases exposed split or merged topology naming
gaps, two returned multiple solids, and two reached region certification
refusals. The fan frame reached its third mounting-hole request before the
owner dispatch timed out. The 14 prior Through All cases remain open.

## Through All Cut

Through All stores the target reference and the positive, negative, or
symmetric direction. Replay projects the current body's analytic bounds
onto the sketch normal and extends the tool beyond the selected end using
the admitted tolerance and an outward rounding allowance. One-sided cuts
start at the sketch plane. The model stores neither a fixed distance nor
the measured body extents. The browser displays the feature as Cut,
Through All.

The desktop-owner regression covers all three directions, grows the
original solid from 10 mm to 20 mm, and verifies analytic volume and the
same target body key after save and reopen.

The focused verification passed 33 desktop-owner extrusion tests, 15
recipe tests, and 253 executed adapter tests. A native Wayland MCP
test confirmed the analytic volume of a circular Through All Cut before
and after the target depth grew from 50 mm to 90 mm.

All 14 previously blocked Through All workflows were rerun. Mains socket
holes passed. Fixing block, bolt flange, veroboard, and jack advanced to
mirror, circular pattern, rectangular pattern, and Join, respectively.
Three cases exposed unresolved split or merged topology names and three
hit profile certification limits. Two Through All requests timed out.
Ziptie timed out during its initial New Solid request on the fresh host.

## Join with a single target body

The typed Extrude request accepts Join for a part with one current solid.
Join stores the exact target body and reuses the ordered replacement path
used by Cut. It fuses the certified prism and validates one solid with
increased analytic volume. The resulting topology must meet both operand
tolerance caps and retain the target body key.

The desktop-owner regression joins a boss supported by a parametric work
plane and checks analytic volume and body identity after resize, undo,
redo, and reopen. A separate regression checks rejection of a disconnected
tool and preservation of the prior body and visible sketch.

The focused run passed 35 desktop-owner extrusion tests, 16 recipe tests,
and 253 executed adapter tests. The Wayland MCP smoke test joined a
circular boss and matched the analytic volume before and after the target
depth changed from 50 mm to 60 mm.

In the 21 live Join workflows, the first implementation exposed a common
case where the kernel attributes one result entity to both the saved
target and the temporary tool. The adapter keeps the saved name when it
is the only persistent contributor and binds to exactly one result entity.
The surface-mount resistor then passed. After that change, 12 workflows
still reached a split-name refusal, five produced two solids, two added
no volume, and one required a parameter formula for extrusion depth.

## Split topology names

Boolean replay records each split input and its immediate descendants.
Each descendant's identity combines the original name with exact history
from the result entity and its incident boundaries. The naming check
requires distinct identities across the result. The old key remains in
the split history and does not bind to an arbitrary descendant.

Split records survive retained replay checkpoints and carry forward into
later body features. The ordering test reverses the candidate topology
and checks identical bindings. The desktop regression uses an overlapping
Join, checks both descendants of the original cap, and verifies their
names and analytic volume after resize and reopen.

When the full witness set exceeds the reference format's limits, the
adapter selects exact witnesses until every pair of descendants is
distinguished. It keeps the original parent in each encoded name. Tests
cover order independence, over-limit signatures, and refusal when two
descendants have identical evidence.

The focused run passed 36 desktop-owner extrusion tests and 255 executed
adapter tests. The Wayland MCP smoke test also passed. Of the 12 Join
workflows previously stopped by split names, seven now pass. PCB mount
and faceplate reach rectangular pattern. Extrusion center section
advances to a Cut that produces four solids, and square button advances
to a later Join that adds no volume. Belt B still has indistinguishable
descendants. In a separate rerun of seven Cut naming failures, adjust and
light strip clip passed and five cases still had indistinguishable names.

## Rectangular patterns of extrusion features

The typed `patternRectangular` request stores Join and Cut source feature
IDs and replays their tools on one current solid. It supports numeric
counts, spacing or fitted extents, mid-plane placement, and occurrence
suppression. Occurrence names use the source ID and signed offsets from
the seed cell. The feature recipe and body references survive save and
reopen.

The adapter translates certified seed tools and batches consecutive
tools with the same Boolean operation. Each batch checks solid geometry,
analytic volume, and the target and tool tolerance caps. Batching keeps
the 6×4 veroboard Cut inside the desktop owner's response window.

Three Wayland MCP workflows passed on API 0.166.0. PCB mount matched
4.504779 cm³ and 4.844071 cm³ at post lengths of 11 mm and 14 mm.
Faceplate matched 5.817876 cm³ and 6.873451 cm³ at boss lengths of 8 mm
and 12 mm. Veroboard matched 0.485969 cm³ with 24 holes. The desktop-owner
regression checks the original body key and analytic volume through
undo, redo, resize, and reopen.

Count expressions, boundary clipping, other placement modes, and source
features outside Join and Cut extrusions remain open. The report's
unsupported-method totals still reflect its original sweep.

## Circular patterns of extrusion features

The typed `patternCircular` request rotates certified Join and Cut tools
about an authored axis and uses the same batched replay as rectangular
patterns. It stores a numeric count and an angle in radians or a
reference to an angular parameter. It supports full-sweep spacing,
explicit increments, fitted angles, mid-plane placement, and occurrence
suppression. The desktop regression checks an off-origin axis and
preserves the original body key and remaining holes after resize and
reopen.

The bolt flange Wayland MCP workflow passed. Its six-hole flange matched
4.410796 cm³ at 40 mm diameter and 7.238229 cm³ after the diameter grew
to 50 mm. Count expressions, clipping, and source features outside
Join and Cut extrusions remain open for circular patterns.

PCB mount, faceplate, and veroboard passed again on the shared pattern
implementation. The wheel's cut still stops at the source sketch on a
face-offset work plane. The typed extrusion returns `expected an
available current plane` before it creates the seed for the wheel's
circular pattern.
