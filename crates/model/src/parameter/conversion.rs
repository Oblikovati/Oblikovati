// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Change kind without changing identity or dependencies; for example `parameters.convert_kind(id, REFERENCE_PARAM)`.
    pub fn convert_kind(
        &mut self,
        id: Id,
        target: ParameterKind,
    ) -> Result<(), ParameterError<Id>> {
        let state = &self.require_record(id)?.state;
        if state.built_in || state.kind == DERIVED_PARAM {
            return Err(ParameterError::Value {
                id,
                found: format!("kind={:?}, built_in={}", state.kind, state.built_in),
                expected: "a non-built-in, non-Derived source parameter",
            });
        }
        if !matches!(target, USER_PARAM | MODEL_PARAM | REFERENCE_PARAM) {
            return Err(ParameterError::Kind {
                kind: target,
                expected: "a User, Model, or Reference conversion target",
            });
        }
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .kind = target;
        Ok(())
    }

    /// Add a feature-owned dimension under the first unused dN name; for example `parameters.insert_auto_model(id, "3 cm")`.
    pub fn insert_auto_model(&mut self, id: Id, source: &str) -> Result<(), ParameterError<Id>> {
        let name = self.first_available_name((0..).map(|index: usize| format!("d{index}")));
        self.insert_expression(id, &name, MODEL_PARAM, source)?;
        self.records
            .get_mut(&id)
            .expect("inserted model parameter")
            .state
            .built_in = true;
        Ok(())
    }

    pub(super) fn first_available_name(&self, candidates: impl Iterator<Item = String>) -> String {
        // With N occupied names, one of N+1 distinct candidates must be free.
        candidates
            .take(self.names.len() + 1)
            .find(|name| self.find(name).is_none())
            .expect("more distinct candidates than occupied names")
    }
}
