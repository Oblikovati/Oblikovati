// SPDX-License-Identifier: GPL-2.0-only

use super::UnitError;

pub(super) fn finite(value: f64) -> Result<f64, UnitError> {
    if !value.is_finite() {
        return Err(UnitError::Display(format!(
            "nonfinite display value {value}"
        )));
    }
    Ok(value)
}

pub(super) fn shortest(value: f64) -> String {
    let scientific = format!("{value:e}");
    let (mantissa, exponent) = scientific
        .split_once('e')
        .expect("finite scientific number");
    let power: i32 = exponent.parse().expect("decimal exponent");
    if !(-4..6).contains(&power) {
        return format!("{mantissa}e{power:+03}");
    }
    value.to_string()
}

pub(super) fn fixed(value: f64, precision: i64) -> Result<String, UnitError> {
    // Keep output work bounded independently of the persisted preference (#3654).
    const MAX_FORMAT_DIGITS: i64 = 1_000_000;
    if !(0..=MAX_FORMAT_DIGITS).contains(&precision) {
        return Err(UnitError::Display(format!(
            "display precision {precision}: expected at most {MAX_FORMAT_DIGITS} digits per formatted value"
        )));
    }
    Ok(format!(
        "{value:.precision$}",
        precision = precision as usize
    ))
}

pub(super) fn with_unit(value: String, unit: &str) -> String {
    if unit.is_empty() {
        return value;
    }
    format!("{value} {unit}")
}

pub(super) fn round_decimal(value: f64, precision: i64) -> Result<f64, UnitError> {
    if precision > 308 {
        // Formatting avoids overflowing the power-of-ten intermediate for tiny values.
        return fixed(value, precision)?.parse().map_err(|_| {
            UnitError::Display(format!(
                "rounded value {value} at precision {precision}: expected a finite decimal"
            ))
        });
    }
    round_scaled(value, 10_f64.powi(precision as i32))
}

pub(super) fn round_scaled(value: f64, scale: f64) -> Result<f64, UnitError> {
    finite(value * scale).map(|scaled| scaled.round() / scale)
}

pub(super) fn ticks(value: f64, scale: u64) -> Result<u64, UnitError> {
    let rounded = finite(value.abs() * scale as f64)?.round();
    // Exact integer range of binary64; refuse lossy/saturating integer display casts.
    if rounded > 9_007_199_254_740_991.0 {
        return Err(UnitError::Display(format!(
            "display tick count {rounded} exceeds the exact binary64 integer range"
        )));
    }
    Ok(rounded as u64)
}
