// SPDX-License-Identifier: GPL-2.0-only

use super::{UnitError, UnitsOfMeasure};
use crate::quantity::Quantity;

impl UnitsOfMeasure {
    /// Transfer a current quantity into another document's working units.
    /// Example: `source_units.transfer_quantity(length, &target_units)?` before deriving a value.
    pub fn transfer_quantity(
        &self,
        quantity: Quantity,
        target: &Self,
    ) -> Result<Quantity, UnitError> {
        if !quantity.dimension.is_arithmetic() || !quantity.value.is_finite() {
            return Err(transfer_error(quantity, self, target));
        }
        let source_factor = self.scale_factor(quantity.dimension);
        let target_factor = target.scale_factor(quantity.dimension);
        if source_factor == target_factor || quantity.value == 0.0 {
            return Ok(quantity);
        }
        let value = scale_value(quantity.value, source_factor, target_factor);
        if !value.is_finite() || value == 0.0 {
            return Err(transfer_error(quantity, self, target));
        }
        Ok(Quantity::new(value, quantity.dimension))
    }
}

fn scale_value(value: f64, source: f64, target: f64) -> f64 {
    let ratio = source / target;
    if ratio.is_normal() {
        return value * ratio;
    }
    // #3657: an extreme ratio can overflow/underflow while the transferred value is representable.
    let product = value * source;
    if product == 0.0 {
        return (value / target) * source;
    }
    product / target
}

fn transfer_error(
    quantity: Quantity,
    source: &UnitsOfMeasure,
    target: &UnitsOfMeasure,
) -> UnitError {
    UnitError::Display(format!(
        "source quantity {quantity:?}, working scale {} -> {}: expected a finite arithmetic quantity representable in target working units",
        source.working_scale(),
        target.working_scale()
    ))
}
