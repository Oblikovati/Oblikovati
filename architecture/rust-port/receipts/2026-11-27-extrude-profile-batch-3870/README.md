# Certify a batch of Extrude profiles

Oblikovati's Martensite adapter builds one planar arrangement for the
selected sketch. Each directed boundary must resolve to a distinct
region before the adapter extrudes any of them. The result retains one
body per selection, with cap and source-curve roles and measured
tolerances. An error returns a failure for the batch.

The analytic fixture contains disjoint squares of side lengths 2 and 3
in working units. At depth 1.5, the test checks volumes of 6 and 13.5,
the areas of all twelve faces, loop counts, topology, source roles, and
the tolerance limit. Empty selections, missing boundaries, and a
duplicate with a different cyclic start are refused.

The existing single-profile and taper tests also use the same
arrangement and retention helpers. Multi-body feature authoring,
persistent body slots, and desktop selection remain open.

[Certified adapter tests](certified-tests.txt),
[single-profile replay tests](sketch-prism-tests.txt).

The existing single-profile clear-and-repick desktop regression passed
on both backends after the adapter change.
[Wayland output](wayland-native.txt), [X11 output](x11-native.txt).
