// SPDX-License-Identifier: GPL-2.0-only

use super::{UnitError, number};

pub(super) fn fraction(value: f64, denominator: u64) -> Result<String, UnitError> {
    let ticks = number::ticks(value, denominator)?;
    Ok(signed(
        value,
        fraction_part(ticks / denominator, ticks % denominator, denominator),
    ))
}

pub(super) fn architectural(inches: f64, denominator: u64) -> Result<String, UnitError> {
    let ticks = number::ticks(inches, denominator)?;
    let feet = ticks / (12 * denominator);
    let remaining = ticks % (12 * denominator);
    let inches_text = fraction_part(
        remaining / denominator,
        remaining % denominator,
        denominator,
    );
    let text = if feet == 0 {
        format!("{inches_text}\"")
    } else {
        format!("{feet}' {inches_text}\"")
    };
    Ok(signed(inches, text))
}

pub(super) fn dms(degrees: f64) -> Result<String, UnitError> {
    number::ticks(degrees, 3600)?;
    // Keep Go's staged subtraction, rounding, and carry order for boundary parity.
    let mut whole = degrees.abs().trunc() as u64;
    let minutes = (degrees.abs() - whole as f64) * 60.0;
    let mut minute = minutes.trunc() as u64;
    let mut second = ((minutes - minute as f64) * 60.0).round() as u64;
    if second == 60 {
        second = 0;
        minute += 1;
    }
    if minute == 60 {
        minute = 0;
        whole += 1;
    }
    Ok(signed(degrees, format!("{whole}° {minute}' {second}\"")))
}

fn fraction_part(whole: u64, remainder: u64, denominator: u64) -> String {
    if remainder == 0 {
        return whole.to_string();
    }
    // Denominators are powers of two, so common factors are trailing zero bits.
    let shift = remainder.trailing_zeros().min(denominator.trailing_zeros());
    let fraction = format!("{}/{}", remainder >> shift, denominator >> shift);
    if whole == 0 {
        return fraction;
    }
    format!("{whole}-{fraction}")
}

fn signed(value: f64, text: String) -> String {
    if value < 0.0 {
        return format!("-{text}");
    }
    text
}
