# Measure midpoint index diagnostic

The Measure point pass uses the existing immutable sketch index for line and
arc defining-point lookup. The same-process paired release test compares it
with the former point-list scans. After 20 warmups and 120 samples, the
200-line median was 14.386 µs for scans and 8.777 µs for the index. For 1,000
lines it was 337.765 µs and 56.554 µs. These times include index construction;
they do not include curve picking, rendering, input delivery, or scanout.
The methods were always measured in the same order.

A separate native run on AMD Radeon 8060S / RADV at 1440 × 900 measured
53.296 ms median and 110.399 ms p95 per bounded frame. All 120 frames exceeded
16.67 ms. Other builds were active on the machine, so this run is not a
controlled comparison with the earlier baseline. The 60 fps target remains
open (#4032, #4035, Martensite #501).

The native Measure scenario is a diagnostic with an unversioned fixture; it
is not verified acceptance. The previous receipt describes its source and
coverage.
