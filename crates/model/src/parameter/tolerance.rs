// SPDX-License-Identifier: GPL-2.0-only

use super::*;
mod restore;
use crate::quantity::Quantity;
use oblikovati_api::types::{
    MODEL_VALUE_LOWER, MODEL_VALUE_MEDIAN, MODEL_VALUE_NOMINAL, MODEL_VALUE_UPPER, ModelValueType,
    TOLERANCE_BASIC, TOLERANCE_DEFAULT, TOLERANCE_DEVIATION, TOLERANCE_LIMITS_STACKED,
    TOLERANCE_MAX, TOLERANCE_MIN, TOLERANCE_REFERENCE, TOLERANCE_SYMMETRIC, Tolerance,
    ToleranceType,
};

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Read the selected engineering value; for example `parameters.model_value(depth_id)`.
    /// Expression references continue to read nominal values. Features use a tracked read scope.
    pub fn model_value(&self, id: Id) -> Result<Quantity, ParameterError<Id>> {
        let nominal = self.require_numeric_value(id)?;
        let state = &self.records[&id].state;
        selected_quantity(id, state, nominal)
    }

    /// Inspect the last-good engineering value without making a failed parameter usable by features.
    /// Example: `parameters.inspection_value(id)` retains a failed parameter's tolerance offset.
    pub fn inspection_value(&self, id: Id) -> Result<Option<Quantity>, ParameterError<Id>> {
        let state = &self.require_record(id)?.state;
        let nominal = match &state.last_good_value {
            Some(ParameterValue::Numeric(value)) => *value,
            Some(_) => return Ok(None),
            None => Quantity::new(0.0, crate::quantity::Dimension::Unitless),
        };
        selected_quantity(id, state, nominal).map(Some)
    }

    /// Select a tolerance-band value; for example `parameters.set_model_value_type(id, MODEL_VALUE_UPPER)`.
    pub fn set_model_value_type(
        &mut self,
        id: Id,
        selection: ModelValueType,
    ) -> Result<(), ParameterError<Id>> {
        self.require_numeric_value(id)?;
        require_selection(selection)?;
        self.store_model_selection(id, selection);
        Ok(())
    }

    pub(super) fn store_model_selection(&mut self, id: Id, selection: ModelValueType) {
        let state = &mut self
            .records
            .get_mut(&id)
            .expect("validated parameter")
            .state;
        if state.model_value_type != selection {
            state.model_value_type = selection;
            self.changed.insert(id);
        }
    }

    /// Store working-unit deviations; for example `parameters.set_deviation(id, 0.2, -0.1)`.
    pub fn set_deviation(
        &mut self,
        id: Id,
        upper: f64,
        lower: f64,
    ) -> Result<(), ParameterError<Id>> {
        self.store_tolerance(
            id,
            Tolerance {
                type_: TOLERANCE_DEVIATION,
                upper,
                lower,
                ..Tolerance::default()
            },
        )
    }

    /// Store a nonnegative symmetric band; for example `parameters.set_symmetric(id, 0.05)`.
    pub fn set_symmetric(&mut self, id: Id, band: f64) -> Result<(), ParameterError<Id>> {
        self.store_tolerance(
            id,
            Tolerance {
                type_: TOLERANCE_SYMMETRIC,
                upper: band,
                lower: -band,
                ..Tolerance::default()
            },
        )
    }

    /// Convert absolute limits to deviations from current nominal; for example `parameters.set_limits(id, 10.3, 9.8)`.
    pub fn set_limits(&mut self, id: Id, upper: f64, lower: f64) -> Result<(), ParameterError<Id>> {
        let nominal = self.require_numeric_value(id)?.value;
        self.store_tolerance(
            id,
            Tolerance {
                type_: TOLERANCE_LIMITS_STACKED,
                upper: upper - nominal,
                lower: lower - nominal,
                ..Tolerance::default()
            },
        )
    }

    /// Clear the explicit band; for example `parameters.set_tolerance_default(id)`.
    pub fn set_tolerance_default(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        self.store_tolerance(id, Tolerance::default())
    }

    /// Set a bandless annotation; for example `parameters.set_bandless(id, TOLERANCE_BASIC)`.
    pub fn set_bandless(&mut self, id: Id, kind: ToleranceType) -> Result<(), ParameterError<Id>> {
        if !matches!(
            kind,
            TOLERANCE_BASIC
                | TOLERANCE_REFERENCE
                | TOLERANCE_MIN
                | TOLERANCE_MAX
                | TOLERANCE_DEFAULT
        ) {
            return Err(numeric_error(
                id,
                format!("{kind:?}"),
                "Basic, Reference, Min, Max, or Default",
            ));
        }
        if kind == TOLERANCE_DEFAULT {
            return self.set_tolerance_default(id);
        }
        self.store_tolerance(
            id,
            Tolerance {
                type_: kind,
                ..Tolerance::default()
            },
        )
    }

    /// Restore saved deviations without resolving fit tables again.
    /// Example: `parameters.set_tolerance(id, saved_tolerance)?;`.
    pub fn set_tolerance(
        &mut self,
        id: Id,
        tolerance: Tolerance,
    ) -> Result<(), ParameterError<Id>> {
        validate_tolerance_kind(id, tolerance.type_)?;
        self.store_tolerance(id, tolerance)
    }

    pub(super) fn store_tolerance(
        &mut self,
        id: Id,
        tolerance: Tolerance,
    ) -> Result<(), ParameterError<Id>> {
        self.require_numeric_value(id)?;
        validate_band(id, &tolerance)?;
        let state = &mut self
            .records
            .get_mut(&id)
            .expect("validated parameter")
            .state;
        // Recipe comparison preserves signed zero even when numeric equality holds (#3614).
        if state.tolerance != tolerance
            || state.tolerance.upper.to_bits() != tolerance.upper.to_bits()
            || state.tolerance.lower.to_bits() != tolerance.lower.to_bits()
        {
            state.tolerance = tolerance;
            self.changed.insert(id);
        }
        Ok(())
    }

    pub(super) fn require_numeric_value(&self, id: Id) -> Result<Quantity, ParameterError<Id>> {
        let state = &self.require_record(id)?.state;
        let value = state
            .current_value()
            .filter(|quantity| quantity.dimension.is_arithmetic());
        value.ok_or_else(|| {
            numeric_error(
                id,
                format!("{:?}, {:?}", state.last_good_value, state.health),
                "a healthy numeric parameter",
            )
        })
    }
}

fn selected_offset(tolerance: &Tolerance, selection: ModelValueType) -> f64 {
    match selection {
        MODEL_VALUE_UPPER => tolerance.upper,
        MODEL_VALUE_LOWER => tolerance.lower,
        // Keep the Go arithmetic order. A non-finite result is refused by model_value.
        MODEL_VALUE_MEDIAN => (tolerance.upper + tolerance.lower) / 2.0,
        _ => 0.0,
    }
}

fn validate_band<Id>(id: Id, tolerance: &Tolerance) -> Result<(), ParameterError<Id>> {
    if !tolerance.upper.is_finite()
        || !tolerance.lower.is_finite()
        || tolerance.upper < tolerance.lower
    {
        return Err(numeric_error(
            id,
            format!("upper={}, lower={}", tolerance.upper, tolerance.lower),
            "finite working-unit deviations with upper >= lower",
        ));
    }
    Ok(())
}

fn numeric_error<Id>(id: Id, found: impl ToString, expected: &'static str) -> ParameterError<Id> {
    ParameterError::Value {
        id,
        found: found.to_string(),
        expected,
    }
}

pub(super) fn require_selection<Id>(selection: ModelValueType) -> Result<(), ParameterError<Id>> {
    if !matches!(
        selection,
        MODEL_VALUE_NOMINAL | MODEL_VALUE_LOWER | MODEL_VALUE_UPPER | MODEL_VALUE_MEDIAN
    ) {
        return Err(ParameterError::Selection(selection));
    }
    Ok(())
}

fn selected_quantity<Id: Copy>(
    id: Id,
    state: &ParameterState<Id>,
    nominal: Quantity,
) -> Result<Quantity, ParameterError<Id>> {
    if state.model_value_type == MODEL_VALUE_NOMINAL {
        return Ok(nominal);
    }
    let value = nominal.value + selected_offset(&state.tolerance, state.model_value_type);
    if !value.is_finite() {
        return Err(numeric_error(id, value, "a finite selected model value"));
    }
    Ok(Quantity::new(value, nominal.dimension))
}

fn validate_tolerance_kind<Id>(id: Id, kind: ToleranceType) -> Result<(), ParameterError<Id>> {
    if !(TOLERANCE_DEFAULT.0..=TOLERANCE_REFERENCE.0).contains(&kind.0) {
        return Err(numeric_error(
            id,
            format!("{:?}", kind),
            "a known API tolerance type",
        ));
    }
    Ok(())
}
