// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Restore a feature-owned model parameter's conversion guard after decoding.
    /// Example: `parameters.restore_builtin_model(id)?` after creating a model parameter.
    pub fn restore_builtin_model(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        let state = &self.require_record(id)?.state;
        if state.kind != MODEL_PARAM {
            return Err(ParameterError::Value {
                id,
                found: format!("kind={:?}", state.kind),
                expected: "a Model parameter for a built-in dimension",
            });
        }
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .built_in = true;
        Ok(())
    }
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
        let name = self.auto_model_name();
        self.insert_expression(id, &name, MODEL_PARAM, source)?;
        self.mark_auto_model(id);
        Ok(())
    }

    /// Store a built-in dimension with its captured unit context.
    /// Example: `parameters.insert_auto_model_scalar(id, "3", Dimension::Length)?`.
    pub fn insert_auto_model_scalar(
        &mut self,
        id: Id,
        source: &str,
        dimension: crate::quantity::Dimension,
    ) -> Result<(), ParameterError<Id>> {
        let name = self.auto_model_name();
        self.insert_scalar_expression(id, &name, source, dimension)?;
        self.mark_auto_model(id);
        Ok(())
    }

    fn auto_model_name(&self) -> String {
        self.first_available_name((0..).map(|index: usize| format!("d{index}")))
    }

    fn mark_auto_model(&mut self, id: Id) {
        self.records
            .get_mut(&id)
            .expect("inserted model parameter")
            .state
            .built_in = true;
    }

    pub(super) fn first_available_name(&self, candidates: impl Iterator<Item = String>) -> String {
        // With N occupied names, one of N+1 distinct candidates must be free.
        candidates
            .take(self.names.len() + 1)
            .find(|name| self.find(name).is_none())
            .expect("more distinct candidates than occupied names")
    }
}
