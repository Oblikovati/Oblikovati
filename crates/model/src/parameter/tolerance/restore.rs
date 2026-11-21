// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Restore diagnostic numeric state without clearing failure health or evaluating readers.
    /// Example: `parameters.restore_numeric_diagnostics(id, last_good, tolerance, selection)?;`.
    pub fn restore_numeric_diagnostics(
        &mut self,
        id: Id,
        last_good: Option<Quantity>,
        tolerance: Tolerance,
        selection: ModelValueType,
    ) -> Result<(), ParameterError<Id>> {
        self.require_failed_numeric_recipe(id)?;
        validate_diagnostic(id, last_good)?;
        if tolerance != Tolerance::default() {
            validate_tolerance_kind(id, tolerance.type_)?;
        }
        validate_band(id, &tolerance)?;
        require_selection(selection)?;
        let mut candidate = self.records[&id].state.clone();
        candidate.last_good_value = last_good.map(ParameterValue::Numeric);
        candidate.tolerance = tolerance;
        candidate.model_value_type = selection;
        if let Some(nominal) = last_good {
            selected_quantity(id, &candidate, nominal)?;
        }
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state = candidate;
        self.changed.insert(id);
        Ok(())
    }

    fn require_failed_numeric_recipe(&self, id: Id) -> Result<(), ParameterError<Id>> {
        let record = self.require_record(id)?;
        let numeric = record.formula.is_some()
            || matches!(
                record.state.last_good_value,
                Some(ParameterValue::Numeric(_))
            );
        if !numeric || matches!(record.state.health, ParameterHealth::Healthy) {
            return Err(numeric_error(
                id,
                format!("{:?}", record.state.health),
                "a failed numeric recipe",
            ));
        }
        Ok(())
    }
}

fn validate_diagnostic<Id>(id: Id, quantity: Option<Quantity>) -> Result<(), ParameterError<Id>> {
    if let Some(quantity) = quantity
        && (!quantity.value.is_finite() || !quantity.dimension.is_arithmetic())
    {
        return Err(numeric_error(
            id,
            format!("{quantity:?}"),
            "a finite numeric diagnostic quantity",
        ));
    }
    Ok(())
}
