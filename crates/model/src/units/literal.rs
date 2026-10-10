// SPDX-License-Identifier: GPL-2.0-only

use super::{UnitError, UnitsOfMeasure, registry};
use crate::quantity::{Dimension, Quantity};

impl UnitsOfMeasure {
    /// Parse a literal with optional units; example: `units.parse("25 mm", Dimension::Length)`.
    /// A suffix overrides the default category. This does not parse expressions.
    pub fn parse(&self, input: &str, default: Dimension) -> Result<Quantity, UnitError> {
        let (number, suffix) = split_literal(input);
        let value = number
            .parse::<f64>()
            .ok()
            .filter(|value| value.is_finite())
            .ok_or_else(|| UnitError::InvalidNumber(input.to_owned()))?;
        if suffix.is_empty() {
            return Ok(self.from_preferred(value, default));
        }
        let unit = registry::lookup(suffix)?;
        Ok(self.to_working(value, unit.dimension, unit.factor))
    }
}

fn split_literal(input: &str) -> (&str, &str) {
    let trimmed = input.trim();
    let end = trimmed
        .bytes()
        .enumerate()
        .take_while(|&(index, byte)| number_byte(trimmed.as_bytes(), index, byte))
        .count();
    let (number, suffix) = trimmed.split_at(end);
    (number, suffix.trim())
}

fn number_byte(input: &[u8], index: usize, byte: u8) -> bool {
    match byte {
        b'0'..=b'9' | b'.' => true,
        b'e' | b'E' => index > 0,
        b'+' | b'-' => index == 0 || matches!(input[index - 1], b'e' | b'E'),
        _ => false,
    }
}
