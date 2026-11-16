# Projection cache evidence

Base: `b1ed49be0`. Martensite: `bef25f47548431c8f128f08f66a1f719bcee4982`.
The final change extracts the projection loop into a helper after these measurements.

The native projection benchmark used Xvfb and llvmpipe at 1440 × 900. For 201
unchanged points, the median was 1397 µs before and 28 µs after. For alternating
fixed and changing 201-point batches, the median was 2902 µs before and 1174 µs
after. These are single exploratory runs, not a performance guarantee.

The full Ready-state hover benchmark used 200 segments. Its final run on AMD
Radeon 8060S (RADV STRIX_HALO), at 1440 × 900, measured a median of 45.93 ms
and a 95th percentile of 54.94 ms; all 120 frames exceeded 16.67 ms. The test
uses scripted motion and bounded frame runs; it does not measure display scanout.
The Xvfb frame runs included concurrent builds and are not a controlled comparison.
The 60 fps target remains open.

The viewport suite first had four failures with the inherited `@im=ibus` setting
and no input method provider. With `XMODIFIERS=@im=none`, all 173 tests passed.
The native projection and XTEST UI tests passed. The screenshot shows the point
hover cue; its status bar records the unavailable inherited input method.
The focused tests were repeated after the helper extraction.
