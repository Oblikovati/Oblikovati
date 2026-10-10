// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::quantity::Quantity;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    pub(super) fn rebind(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        let result = self.bind_expression(id);
        self.recompute(id)?;
        result
    }

    pub(super) fn bind_expression(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        let binding = self.records[&id].expression.bind(|name| self.find(name));
        let drivers = binding.bound_ids();
        self.records
            .get_mut(&id)
            .expect("registered parameter")
            .binding = binding;
        if let Err(error) = self.graph.replace_drivers(id, &drivers) {
            self.graph.replace_drivers(id, &[])?;
            self.fail(id, ParameterFailure::Cycle(error.clone()));
            return Err(error.into());
        }
        self.records
            .get_mut(&id)
            .expect("registered parameter")
            .state
            .health = ParameterHealth::Healthy;
        Ok(())
    }

    pub(super) fn recompute(&mut self, changed: Id) -> Result<(), ParameterError<Id>> {
        for id in self.graph.recompute_order(changed)? {
            self.changed.insert(id);
            if matches!(
                self.records[&id].state.health,
                ParameterHealth::Failed(ParameterFailure::Cycle(_))
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

    fn evaluate_parameter(&self, id: Id) -> Result<Quantity, ParameterFailure<Id>> {
        let binding = &self.records[&id].binding;
        self.validate_inputs(binding)?;
        let value = binding
            .evaluate(|driver| self.records[&driver].state.current_value())
            .map_err(ParameterFailure::Evaluation)?;
        if !value.value.is_finite() {
            return Err(ParameterFailure::NonFinite(value));
        }
        Ok(value)
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
        let blocked: Vec<Id> = binding
            .bound_ids()
            .into_iter()
            .filter(|driver| self.records[driver].state.current_value().is_none())
            .collect();
        if !blocked.is_empty() {
            return Err(ParameterFailure::BlockedBy(blocked));
        }
        Ok(())
    }

    fn succeed(&mut self, id: Id, value: Quantity) {
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
