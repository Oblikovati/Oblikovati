// SPDX-License-Identifier: GPL-2.0-only
use super::PlacedSketch;
use crate::{
    parameter::{ParameterCollection, ParameterError},
    quantity::Dimension,
    sketch::{SketchError, SketchPoint2D, SketchRelationId, SketchSolveOutcome, same_components},
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
    ProjectionInputsRequired,
    ProjectionInputsChanged,
    Current(&'a SketchEvaluation),
}
#[derive(Clone, Debug, PartialEq)]
pub(super) struct SketchEvaluationStamp {
    inputs: Vec<(u64, SketchParameterInput)>,
    evaluation: SketchEvaluation,
    projections: Result<Vec<SketchPoint2D>, SketchError>,
}
#[derive(Clone, Debug, PartialEq)]
enum SketchParameterInput {
    Numeric { dimension: Dimension, bits: u64 },
    Failed(ParameterError<u64>),
}
impl PlacedSketch {
    /// Inspect dimension health; projections require the owning document's current source resolution.
    /// Example: `placed.health(&parameters)` for a sketch without associations.
    pub fn health<'a>(&'a self, parameters: &ParameterCollection<u64>) -> PlacedSketchHealth<'a> {
        if !self.point_projections.is_empty() {
            return PlacedSketchHealth::ProjectionInputsRequired;
        }
        self.parameter_health(parameters)
    }
    fn parameter_health<'a>(
        &'a self,
        parameters: &ParameterCollection<u64>,
    ) -> PlacedSketchHealth<'a> {
        let Some(stamp) = &self.evaluation else {
            return PlacedSketchHealth::Unevaluated;
        };
        let changed = changed_inputs(&stamp.inputs, parameters);
        if !changed.is_empty() {
            return PlacedSketchHealth::Stale(changed);
        }
        PlacedSketchHealth::Current(&stamp.evaluation)
    }
    pub(crate) fn health_with_projection_inputs<'a>(
        &'a self,
        parameters: &ParameterCollection<u64>,
        projections: &Result<Vec<SketchPoint2D>, SketchError>,
    ) -> PlacedSketchHealth<'a> {
        if self
            .evaluation
            .as_ref()
            .is_some_and(|stamp| !same_projection_inputs(&stamp.projections, projections))
        {
            return PlacedSketchHealth::ProjectionInputsChanged;
        }
        self.parameter_health(parameters)
    }
    pub(super) fn record_evaluation(
        &mut self,
        parameters: &ParameterCollection<u64>,
        projections: Result<Vec<SketchPoint2D>, SketchError>,
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
            projections,
        });
    }
}
fn same_projection_inputs(
    first: &Result<Vec<SketchPoint2D>, SketchError>,
    second: &Result<Vec<SketchPoint2D>, SketchError>,
) -> bool {
    match (first, second) {
        (Ok(first), Ok(second)) => {
            first.len() == second.len()
                && first
                    .iter()
                    .zip(second)
                    .all(|(a, b)| a.id == b.id && same_components(a.position, b.position))
        }
        (Err(first), Err(second)) => first == second,
        _ => false,
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
