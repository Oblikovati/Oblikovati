// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    pub(super) fn rebind(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        let result = self.bind_expression(id);
        self.recompute(id)?;
        result
    }

    pub(super) fn bind_expression(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        let Some(formula) = &self.records[&id].formula else {
            return Ok(());
        };
        let binding = formula.expression.bind(|name| self.find(name));
        let drivers = binding.bound_ids();
        self.records
            .get_mut(&id)
            .expect("registered parameter")
            .formula
            .as_mut()
            .expect("numeric parameter")
            .binding = binding;
        if let Err(error) = self.graph.replace_drivers(id, &drivers) {
            self.graph.replace_drivers(id, &[])?;
            self.fail(id, ParameterFailure::Cycle(error.clone()));
            return Err(error.into());
        }
        let state = &mut self
            .records
            .get_mut(&id)
            .expect("registered parameter")
            .state;
        // Rebinding a renamed reader cannot repair an unavailable external source (#3603).
        if !matches!(
            state.health,
            ParameterHealth::Failed(ParameterFailure::ExternalLink { .. })
        ) {
            state.health = ParameterHealth::Healthy;
        }
        Ok(())
    }

    pub(super) fn recompute(&mut self, changed: Id) -> Result<(), ParameterError<Id>> {
        for id in self.graph.recompute_order(changed)? {
            self.changed.insert(id);
            if matches!(
                self.records[&id].state.health,
                ParameterHealth::Failed(
                    ParameterFailure::Cycle(_) | ParameterFailure::ExternalLink { .. }
                )
            ) {
                continue;
            }
            match self.evaluate_parameter(id) {
                Ok(value) => self.succeed(id, value),
                Err(failure) => self.fail(id, failure),
            }
        }
        Ok(())
    }

    fn evaluate_parameter(&self, id: Id) -> Result<ParameterValue, ParameterFailure<Id>> {
        let record = &self.records[&id];
        let Some(formula) = &record.formula else {
            return Ok(record.state.last_good_value.clone().expect("literal value"));
        };
        let binding = &formula.binding;
        self.validate_inputs(binding)?;
        let value = binding
            .evaluate(|driver| self.records[&driver].state.current_value())
            .map_err(ParameterFailure::Evaluation)?;
        if let Some(context) = &record.state.expression_context {
            context
                .validate_quantity(value)
                .map_err(ParameterFailure::Evaluation)?;
        }
        if !value.value.is_finite() {
            return Err(ParameterFailure::NonFinite(value));
        }
        Ok(ParameterValue::Numeric(value))
    }

    fn validate_inputs(&self, binding: &BoundExpression<Id>) -> Result<(), ParameterFailure<Id>> {
        let unresolved: Vec<String> = binding
            .unresolved()
            .into_iter()
            .map(str::to_owned)
            .collect();
        if !unresolved.is_empty() {
            return Err(ParameterFailure::Unresolved(unresolved));
        }
        self.validate_bound_values(binding)
    }

    fn validate_bound_values(
        &self,
        binding: &BoundExpression<Id>,
    ) -> Result<(), ParameterFailure<Id>> {
        let blocked: Vec<Id> = binding
            .bound_ids()
            .into_iter()
            .filter(|driver| self.records[driver].state.current().is_none())
            .collect();
        if !blocked.is_empty() {
            return Err(ParameterFailure::BlockedBy(blocked));
        }
        let nonnumeric: Vec<Id> = binding
            .bound_ids()
            .into_iter()
            .filter(|driver| self.records[driver].state.current_value().is_none())
            .collect();
        if !nonnumeric.is_empty() {
            return Err(ParameterFailure::NonNumeric(nonnumeric));
        }
        Ok(())
    }

    fn succeed(&mut self, id: Id, value: ParameterValue) {
        let state = &mut self
            .records
            .get_mut(&id)
            .expect("registered parameter")
            .state;
        state.last_good_value = Some(value);
        state.health = ParameterHealth::Healthy;
    }

    fn fail(&mut self, id: Id, failure: ParameterFailure<Id>) {
        self.records
            .get_mut(&id)
            .expect("registered parameter")
            .state
            .health = ParameterHealth::Failed(failure);
    }
}
