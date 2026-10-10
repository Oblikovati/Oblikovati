// SPDX-License-Identifier: GPL-2.0-only

use super::{CustomPropertyFormat, UnitError, number, trim_decimal};
use oblikovati_api::types::*;

pub(super) fn format(degrees: f64, options: &CustomPropertyFormat) -> Result<String, UnitError> {
    let precision = options.precision;
    let decimals = (precision.0 - PRECISION_SECONDS_ANGLE.0).max(0) as u32;
    let subsecond = 10u64.pow(decimals);
    let per_degree = match precision {
        PRECISION_DEGREES_ANGLE => 1,
        PRECISION_MINUTES_ANGLE => 60,
        _ => 3600 * subsecond,
    };
    let ticks = number::ticks(degrees, per_degree)?;
    let body = components(ticks, per_degree, subsecond, decimals, options)?;
    Ok(if degrees.is_sign_negative() {
        format!("-{body}")
    } else {
        body
    })
}

fn components(
    ticks: u64,
    per_degree: u64,
    subsecond: u64,
    decimals: u32,
    options: &CustomPropertyFormat,
) -> Result<String, UnitError> {
    let whole = ticks / per_degree;
    let degree = if options.show_units_string { "°" } else { "" };
    if options.precision == PRECISION_DEGREES_ANGLE {
        return Ok(format!("{whole}{degree}"));
    }
    let remainder = ticks % per_degree;
    let minute = if options.precision == PRECISION_MINUTES_ANGLE {
        remainder
    } else {
        remainder / (60 * subsecond)
    };
    let minute_mark = if options.show_units_string { "'" } else { "" };
    let prefix = format!("{whole}{degree} {minute}{minute_mark}");
    if options.precision == PRECISION_MINUTES_ANGLE {
        return Ok(prefix);
    }
    let second = (remainder % (60 * subsecond)) as f64 / subsecond as f64;
    let text = trim_decimal(number::fixed(second, decimals.into())?, options);
    let second_mark = if options.show_units_string { "\"" } else { "" };
    Ok(format!("{prefix} {text}{second_mark}"))
}
