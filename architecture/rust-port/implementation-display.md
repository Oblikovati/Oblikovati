# Unit display implementation evidence

Tracking: [issue #3584](https://github.com/Oblikovati/Oblikovati/issues/3584).
This is part of R2 in the [Rust port](implementation.md), not milestone completion.

## Implemented behavior

Units now carry clone-owned display preferences. The length format uses the
canonical API enum; angle presentation selects decimal or degrees/minutes/seconds.
Defaults match Go: three length decimals, two angle decimals, and decimal formats.
Internal precision is an explicit `u8` (0–255), while document parameter settings
retain their narrower 0–8 range. Fraction subdivisions clamp to halves through
128ths. Unknown length-format enums are rejected before changing preferences.

Precise formatting emits shortest decimal values with Go-style general notation
and the preferred unit. Display formatting supports fixed decimals, reduced
fractions, architectural feet/inches, and DMS, including sign and carry behavior.
Display-rounded expressions emit parseable decimal unit literals even when the
visible format uses fractions or inline unit symbols. Other numeric categories
use precise formatting. Stored quantities are never changed by formatting.

Architectural conversion applies the document working scale before converting to
inches. This corrects the Go formatter's direct stored-value conversion, which
assumed centimetre working units. All conversion factors come from the unit
registry. Non-finite values and overflowing rounded intermediates return errors.
Integer-based display formats refuse tick counts above binary64's exact integer
range before conversion, rather than overflowing or saturating a platform-sized
integer. Text and Boolean quantities are refused; typed parameter literals own
those values.

## Validation and remaining work

Seven tests check 48 fixed-baseline Go cases for display, rounded-expression, and
precise strings; four working scales; fraction clamps/reduction; architectural
and DMS carries; signed zero; scientific notation; category fallback; invalid
options; and numeric-range failures. The fixture was captured from Go `aa8297d1`
with Go 1.27.0 on Linux/amd64, using the existing public formatter methods.

UI fields remain to be connected. API formatting, persisted unit preferences,
and [custom-property rendering](implementation-properties.md) now use the Rust
model and session paths. Display-rounded source intentionally
changes a value to display granularity; precise formatting instead preserves the
preferred-unit floating-point value's shortest representation. Neither is a
promise of exact physical round-trip through every floating-point unit conversion.
