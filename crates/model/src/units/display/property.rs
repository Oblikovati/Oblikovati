// SPDX-License-Identifier: GPL-2.0-only

use super::{UnitError, UnitsOfMeasure, engineering, number};
use crate::{
    parameter::CustomPropertyFormat,
    quantity::{Dimension, Quantity},
};
use oblikovati_api::types::*;
mod angular;

impl CustomPropertyFormat {
    /// Format a numeric property without changing its parameter or document preferences.
    /// Example: `format.format_numeric(quantity, &units)?` returns a canonical API value.
    pub fn format_numeric(
        &self,
        quantity: Quantity,
        units: &UnitsOfMeasure,
    ) -> Result<VariantValue, UnitError> {
        let (value, suffix) = selected_value(self, quantity, units)?;
        validate_options(self, quantity.dimension)?;
        if self.property_type == CUSTOM_PROPERTY_NUMBER {
            return Ok(VariantValue::Double(value));
        }
        let text = self.format_text(value, quantity, units)?;
        let inline_angle = self.precision.0 >= PRECISION_DEGREES_ANGLE.0;
        let suffix = if self.show_units_string && !inline_angle {
            suffix
        } else {
            ""
        };
        Ok(VariantValue::String(number::with_unit(text, suffix)))
    }

    fn format_text(
        &self,
        value: f64,
        quantity: Quantity,
        units: &UnitsOfMeasure,
    ) -> Result<String, UnitError> {
        let precision = self.precision.0;
        if precision <= PRECISION_EIGHT_DECIMAL_PLACES.0 {
            return Ok(trim_decimal(
                number::fixed(value, (precision - PRECISION_ZERO_DECIMAL_PLACE.0).into())?,
                self,
            ));
        }
        if precision <= PRECISION_ONE_TWENTY_EIGHTHS_FRACTIONAL.0 {
            return engineering::fraction(value, 1 << (precision - PRECISION_ZERO_FRACTIONAL.0));
        }
        angular::format(units.named_value(quantity, "deg")?, self)
    }
}

fn selected_value<'a>(
    format: &'a CustomPropertyFormat,
    quantity: Quantity,
    units: &'a UnitsOfMeasure,
) -> Result<(f64, &'a str), UnitError> {
    if format.units.is_empty() {
        return Ok((
            units.display_value(quantity)?,
            units.preferred_name(quantity.dimension),
        ));
    }
    Ok((units.named_value(quantity, &format.units)?, &format.units))
}

fn validate_options(format: &CustomPropertyFormat, dimension: Dimension) -> Result<(), UnitError> {
    let precision = format.precision.0;
    let valid_type = matches!(
        format.property_type,
        CUSTOM_PROPERTY_TEXT | CUSTOM_PROPERTY_NUMBER
    );
    let valid_precision = (PRECISION_ZERO_DECIMAL_PLACE.0..=PRECISION_SECONDS_FOUR_DECIMAL_ANGLE.0)
        .contains(&precision);
    let fractional = (PRECISION_ZERO_FRACTIONAL.0..=PRECISION_ONE_TWENTY_EIGHTHS_FRACTIONAL.0)
        .contains(&precision);
    let angular = precision >= PRECISION_DEGREES_ANGLE.0;
    if !valid_type
        || !valid_precision
        || (fractional && dimension != Dimension::Length)
        || (angular && dimension != Dimension::Angle)
    {
        return Err(UnitError::Display(format!(
            "custom property {format:?} for {dimension:?}: expected a defined type and precision compatible with the quantity"
        )));
    }
    Ok(())
}

fn trim_decimal(mut text: String, format: &CustomPropertyFormat) -> String {
    if !format.show_trailing_zeros && text.contains('.') {
        text = text.trim_end_matches('0').trim_end_matches('.').into();
    }
    if !format.show_leading_zeros {
        if text.starts_with("0.") {
            text.remove(0);
        } else if text.starts_with("-0.") {
            text.remove(1);
        }
    }
    text
}
