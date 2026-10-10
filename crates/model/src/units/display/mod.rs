// SPDX-License-Identifier: GPL-2.0-only

mod engineering;
mod number;
mod property;
use super::{UnitError, UnitsOfMeasure};
use crate::quantity::{Dimension, Quantity};
use oblikovati_api::types::{
    DISPLAY_FORMAT_ARCHITECTURAL, DISPLAY_FORMAT_DECIMAL, DISPLAY_FORMAT_FRACTIONAL,
    ParameterDisplayFormat,
};

/// Internal angle presentation, independent of the quantity's stored radians.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum AngleFormat {
    Decimal,
    DegreesMinutesSeconds,
}

/// Document display preferences; precision retains the signed API range.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DisplayOptions {
    pub length_precision: i64,
    pub angle_precision: i64,
    pub length_format: ParameterDisplayFormat,
    pub angle_format: AngleFormat,
}

impl Default for DisplayOptions {
    fn default() -> Self {
        Self {
            length_precision: 3,
            angle_precision: 2,
            length_format: DISPLAY_FORMAT_DECIMAL,
            angle_format: AngleFormat::Decimal,
        }
    }
}

impl UnitsOfMeasure {
    /// Read display preferences; for example `units.display_options().length_precision`.
    pub fn display_options(&self) -> DisplayOptions {
        self.display
    }

    /// Replace display preferences; for example `units.set_display_options(DisplayOptions::default())`.
    pub fn set_display_options(&mut self, options: DisplayOptions) -> Result<(), UnitError> {
        for precision in [options.length_precision, options.angle_precision] {
            if precision < 0 {
                return Err(UnitError::Display(format!(
                    "negative display precision {precision}"
                )));
            }
        }
        if !matches!(
            options.length_format,
            DISPLAY_FORMAT_DECIMAL | DISPLAY_FORMAT_FRACTIONAL | DISPLAY_FORMAT_ARCHITECTURAL
        ) {
            return Err(UnitError::Display(format!(
                "unsupported length format {:?}",
                options.length_format
            )));
        }
        self.display = options;
        Ok(())
    }

    /// Format a preferred-unit value at full precision; for example `units.format_value(length)`.
    pub fn format_value(&self, quantity: Quantity) -> Result<String, UnitError> {
        Ok(number::shortest(self.display_value(quantity)?))
    }

    /// Append the preferred unit to full-precision output; for example `units.format_precise(length)`.
    pub fn format_precise(&self, quantity: Quantity) -> Result<String, UnitError> {
        Ok(number::with_unit(
            self.format_value(quantity)?,
            self.preferred_name(quantity.dimension),
        ))
    }

    /// Render the display value without an external unit label; for example `units.format_value_display(length)`.
    pub fn format_value_display(&self, quantity: Quantity) -> Result<String, UnitError> {
        let value = self.display_value(quantity)?;
        match quantity.dimension {
            Dimension::Length => self.length_display(quantity, value),
            Dimension::Angle if self.display.angle_format == AngleFormat::DegreesMinutesSeconds => {
                engineering::dms(self.named_value(quantity, "deg")?)
            }
            Dimension::Angle => number::fixed(value, self.display.angle_precision),
            _ => Ok(number::shortest(value)),
        }
    }

    /// Render a quantity with its display units; for example `units.format_display(length)`.
    pub fn format_display(&self, quantity: Quantity) -> Result<String, UnitError> {
        let value = self.format_value_display(quantity)?;
        let inline = (quantity.dimension == Dimension::Length
            && self.display.length_format == DISPLAY_FORMAT_ARCHITECTURAL)
            || (quantity.dimension == Dimension::Angle
                && self.display.angle_format == AngleFormat::DegreesMinutesSeconds);
        Ok(number::with_unit(
            value,
            if inline {
                ""
            } else {
                self.preferred_name(quantity.dimension)
            },
        ))
    }

    fn display_value(&self, quantity: Quantity) -> Result<f64, UnitError> {
        if !quantity.dimension.is_arithmetic() {
            return Err(UnitError::Display(format!(
                "nonnumeric quantity {quantity:?}"
            )));
        }
        number::finite(self.to_preferred(quantity))
    }

    fn named_value(&self, quantity: Quantity, name: &str) -> Result<f64, UnitError> {
        let unit = super::registry::require_category(name, quantity.dimension)?;
        number::finite(quantity.value * self.scale_factor(quantity.dimension) / unit.factor)
    }

    fn denominator(&self) -> u64 {
        1 << self.display.length_precision.clamp(1, 7)
    }

    fn length_display(&self, quantity: Quantity, preferred: f64) -> Result<String, UnitError> {
        match self.display.length_format {
            DISPLAY_FORMAT_FRACTIONAL => engineering::fraction(preferred, self.denominator()),
            DISPLAY_FORMAT_ARCHITECTURAL => {
                engineering::architectural(self.named_value(quantity, "in")?, self.denominator())
            }
            _ => number::fixed(preferred, self.display.length_precision),
        }
    }
}

impl UnitsOfMeasure {
    /// Produce parseable source at display granularity; for example `units.display_rounded_expression(length)`.
    pub fn display_rounded_expression(&self, quantity: Quantity) -> Result<String, UnitError> {
        let preferred = self.display_value(quantity)?;
        let (rounded, unit) = match quantity.dimension {
            Dimension::Length => self.rounded_length(quantity, preferred)?,
            Dimension::Angle if self.display.angle_format == AngleFormat::DegreesMinutesSeconds => {
                (
                    number::round_scaled(self.named_value(quantity, "deg")?, 3600.0)?,
                    "deg",
                )
            }
            Dimension::Angle => (
                number::round_decimal(preferred, self.display.angle_precision)?,
                self.preferred_name(Dimension::Angle),
            ),
            _ => (preferred, self.preferred_name(quantity.dimension)),
        };
        Ok(number::with_unit(number::shortest(rounded), unit))
    }

    fn rounded_length(&self, quantity: Quantity, preferred: f64) -> Result<(f64, &str), UnitError> {
        let unit = self.preferred_name(Dimension::Length);
        match self.display.length_format {
            DISPLAY_FORMAT_FRACTIONAL => Ok((
                number::round_scaled(preferred, self.denominator() as f64)?,
                unit,
            )),
            DISPLAY_FORMAT_ARCHITECTURAL => Ok((
                number::round_scaled(self.named_value(quantity, "in")?, self.denominator() as f64)?,
                "in",
            )),
            _ => Ok((
                number::round_decimal(preferred, self.display.length_precision)?,
                unit,
            )),
        }
    }
}
