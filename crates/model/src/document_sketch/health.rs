// SPDX-License-Identifier: GPL-2.0-only
use super::PlacedSketch;
use crate::{
    parameter::{ParameterCollection, ParameterError},
    quantity::Dimension,
    sketch::{SketchError, SketchRelationId, SketchSolveOutcome},
};
use oblikovati_solve::{diagnostics::ConstraintStructure, nonlinear::SolveTermination};

/// Retained numerical evidence; convergence does not imply zero degrees of freedom.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchSolveDiagnostics {
    pub termination: SolveTermination,
    pub structure: ConstraintStructure,
    pub conflicts: Vec<SketchRelationId>,
    pub relative_residual: f64,
}
/// Last attempted verification, retained independently from the authored recipe.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchEvaluation {
    Numerical(SketchSolveDiagnostics),
    Failed(SketchError),
}
/// Freshness against current model-value reads; stale geometry is not verified geometry.
#[derive(Clone, Debug, PartialEq)]
pub enum PlacedSketchHealth<'a> {
    Unevaluated,
    Stale(Vec<u64>),
    Current(&'a SketchEvaluation),
}
#[derive(Clone, Debug)]
pub(super) struct SketchEvaluationStamp {
    inputs: Vec<(u64, SketchParameterInput)>,
    evaluation: SketchEvaluation,
}
#[derive(Clone, Debug, PartialEq)]
enum SketchParameterInput {
    Numeric { dimension: Dimension, bits: u64 },
    Failed(ParameterError<u64>),
}
impl PlacedSketch {
    /// Inspect health against current dimension targets; example: `placed.health(&parameters)`.
    pub fn health<'a>(&'a self, parameters: &ParameterCollection<u64>) -> PlacedSketchHealth<'a> {
        let Some(stamp) = &self.evaluation else {
            return PlacedSketchHealth::Unevaluated;
        };
        let changed = changed_inputs(&stamp.inputs, parameters);
        if !changed.is_empty() {
            return PlacedSketchHealth::Stale(changed);
        }
        PlacedSketchHealth::Current(&stamp.evaluation)
    }
    pub(super) fn record_evaluation(
        &mut self,
        parameters: &ParameterCollection<u64>,
        result: &Result<SketchSolveOutcome, SketchError>,
    ) {
        let inputs = self
            .geometry
            .parameter_dependencies()
            .into_iter()
            .map(|id| (id, parameter_input(parameters, id)))
            .collect();
        self.evaluation = Some(SketchEvaluationStamp {
            inputs,
            evaluation: evaluation(result),
        });
    }
}
fn parameter_input(parameters: &ParameterCollection<u64>, id: u64) -> SketchParameterInput {
    match parameters.model_value(id) {
        Ok(quantity) => SketchParameterInput::Numeric {
            dimension: quantity.dimension,
            bits: quantity.value.to_bits(),
        },
        Err(cause) => SketchParameterInput::Failed(cause),
    }
}
fn changed_inputs(
    inputs: &[(u64, SketchParameterInput)],
    parameters: &ParameterCollection<u64>,
) -> Vec<u64> {
    inputs
        .iter()
        .filter(|(id, old)| *old != parameter_input(parameters, *id))
        .map(|(id, _)| *id)
        .collect()
}
fn evaluation(result: &Result<SketchSolveOutcome, SketchError>) -> SketchEvaluation {
    match result {
        Ok(outcome) => numerical_evaluation(outcome),
        Err(cause) => SketchEvaluation::Failed(cause.clone()),
    }
}
fn numerical_evaluation(outcome: &SketchSolveOutcome) -> SketchEvaluation {
    let numerical = outcome.numerical();
    SketchEvaluation::Numerical(SketchSolveDiagnostics {
        termination: numerical.termination,
        structure: numerical.structure.clone(),
        conflicts: outcome.conflicting_relations(),
        relative_residual: numerical.relative_residual,
    })
}
