// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate simultaneous parameter-expression changes on a private collection.
use super::{expression_context::compile_context_source, *};

/// A rejected batch expression and the parameter whose validation failed.
/// Example: `error.parameter` identifies the table cell to inspect.
#[derive(Debug)]
pub struct ParameterBatchError<Id> {
    pub parameter: Id,
    pub cause: ParameterError<Id>,
}
impl<Id> ParameterBatchError<Id> {
    fn at(parameter: Id, cause: ParameterError<Id>) -> Self {
        Self { parameter, cause }
    }
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Apply expressions together without transient dependency cycles.
    /// Example: `parameters.set_expressions(&[(width, "2 cm".into())])?`.
    pub fn set_expressions(&mut self, edits: &[(Id, String)]) -> Result<(), ParameterError<Id>> {
        self.set_expressions_detailed(edits)
            .map_err(|error| error.cause)
    }
    /// Apply a batch and identify its first invalid parameter without partial mutation.
    /// Example: `parameters.set_expressions_detailed(&[(width, "2 cm".into())])`.
    pub fn set_expressions_detailed(
        &mut self,
        edits: &[(Id, String)],
    ) -> Result<(), ParameterBatchError<Id>> {
        let mut candidate = self.clone();
        let compiled = candidate.compile_batch(edits)?;
        candidate.detach_batch(&compiled)?;
        candidate.install_batch(compiled)?;
        *self = candidate;
        Ok(())
    }
    fn compile_batch(
        &self,
        edits: &[(Id, String)],
    ) -> Result<Vec<(Id, Expression, Option<DimensionExpressionContext>)>, ParameterBatchError<Id>>
    {
        let mut seen = BTreeSet::new();
        let mut compiled = Vec::new();
        for (id, source) in edits {
            if !seen.insert(*id) {
                return Err(ParameterBatchError::at(
                    *id,
                    DependencyError::Duplicate(*id).into(),
                ));
            }
            self.require_editable(*id)
                .map_err(|error| ParameterBatchError::at(*id, error))?;
            self.require_formula(*id)
                .map_err(|error| ParameterBatchError::at(*id, error))?;
            let context = self
                .current_expression_context(*id)
                .map_err(|error| ParameterBatchError::at(*id, error))?;
            let expression = compile_context_source(source, &self.units, context.as_ref())
                .map_err(|error| ParameterBatchError::at(*id, error.into()))?;
            compiled.push((*id, expression, context));
        }
        Ok(compiled)
    }
    fn detach_batch(
        &mut self,
        edits: &[(Id, Expression, Option<DimensionExpressionContext>)],
    ) -> Result<(), ParameterBatchError<Id>> {
        for (id, _, _) in edits {
            self.graph
                .replace_drivers(*id, &[])
                .map_err(|error| ParameterBatchError::at(*id, error.into()))?;
        }
        Ok(())
    }
    fn install_batch(
        &mut self,
        edits: Vec<(Id, Expression, Option<DimensionExpressionContext>)>,
    ) -> Result<(), ParameterBatchError<Id>> {
        let ids: Vec<Id> = edits.iter().map(|(id, _, _)| *id).collect();
        for (id, expression, context) in edits {
            let record = self.records.get_mut(&id).expect("validated parameter");
            record.state.source = expression.source().into();
            record.state.expression_context = context;
            record.state.health = ParameterHealth::Healthy;
            record
                .formula
                .as_mut()
                .expect("validated formula")
                .expression = expression;
        }
        for id in &ids {
            self.bind_expression(*id)
                .map_err(|error| ParameterBatchError::at(*id, error))?;
        }
        for id in &ids {
            self.recompute(*id)
                .map_err(|error| ParameterBatchError::at(*id, error))?;
        }
        self.require_healthy_batch(&ids)
    }
    fn require_healthy_batch(&self, ids: &[Id]) -> Result<(), ParameterBatchError<Id>> {
        for id in ids {
            let state = &self.records[id].state;
            if !matches!(state.health, ParameterHealth::Healthy) {
                return Err(ParameterBatchError::at(
                    *id,
                    ParameterError::Value {
                        id: *id,
                        found: format!(
                            "expression {:?} with health {:?}",
                            state.source, state.health
                        ),
                        expected: "an evaluable expression",
                    },
                ));
            }
        }
        Ok(())
    }
}
