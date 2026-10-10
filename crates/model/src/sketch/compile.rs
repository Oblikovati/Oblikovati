// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarConstraint, Sketch2D, SketchConstraint, SketchError,
    residual::{PlanarEquation, PreparedConstraint},
};
use crate::{parameter::ParameterCollection, quantity::Dimension};
use oblikovati_solve::residual::ResidualBlock;

pub(super) struct PreparedSketch {
    pub values: Vec<f64>,
    pub sources: Vec<PreparedConstraint>,
    pub parameter_dependencies: Vec<u64>,
}
impl PreparedSketch {
    pub fn blocks(&self) -> Vec<ResidualBlock<'_>> {
        self.sources
            .iter()
            .map(|source| ResidualBlock {
                source,
                variables: &source.variables,
                equations: source.equations(),
            })
            .collect()
    }
}
pub(super) fn prepare(
    sketch: &Sketch2D,
    parameters: &ParameterCollection<u64>,
) -> Result<PreparedSketch, SketchError> {
    let values = sketch.variable_values();
    let sources = sketch
        .constraints
        .iter()
        .map(|constraint| {
            Ok(PreparedConstraint {
                equation: equation(constraint, parameters)?,
                variables: map_variables(sketch, &constraint.relation)?,
            })
        })
        .collect::<Result<Vec<_>, SketchError>>()?;
    Ok(PreparedSketch {
        values,
        sources,
        parameter_dependencies: parameter_dependencies(sketch),
    })
}
fn parameter_dependencies(sketch: &Sketch2D) -> Vec<u64> {
    let mut dependencies = Vec::new();
    for constraint in &sketch.constraints {
        if let PlanarConstraint::Distance { parameter, .. }
        | PlanarConstraint::Radius { parameter, .. } = constraint.relation
            && !dependencies.contains(&parameter)
        {
            dependencies.push(parameter);
        }
    }
    dependencies
}

fn map_variables(
    sketch: &Sketch2D,
    relation: &PlanarConstraint,
) -> Result<Vec<usize>, SketchError> {
    sketch
        .constraint_variables(relation)?
        .iter()
        .map(|variable| sketch.variable_column(*variable))
        .collect()
}

fn equation(
    constraint: &SketchConstraint,
    parameters: &ParameterCollection<u64>,
) -> Result<PlanarEquation, SketchError> {
    Ok(match constraint.relation {
        PlanarConstraint::Coincident { .. } | PlanarConstraint::Concentric { .. } => {
            PlanarEquation::Coincident
        }
        PlanarConstraint::PointOnCircle { .. } => PlanarEquation::PointOnCircle,
        PlanarConstraint::LineTangent { side, .. } => PlanarEquation::LineTangent(side),
        PlanarConstraint::CircleTangent { contact, .. } => PlanarEquation::CircleTangent(contact),
        PlanarConstraint::EqualRadius { .. } => PlanarEquation::EqualRadius,
        PlanarConstraint::Radius { parameter, .. } => {
            PlanarEquation::Radius(radius_target(constraint, parameter, parameters)?)
        }
        PlanarConstraint::Horizontal { .. } => PlanarEquation::Coordinate(1),
        PlanarConstraint::Vertical { .. } => PlanarEquation::Coordinate(0),
        PlanarConstraint::Fix { position, .. } => PlanarEquation::Fix(position),
        PlanarConstraint::Midpoint { .. } => PlanarEquation::Midpoint,
        PlanarConstraint::PointOnLine { .. } => PlanarEquation::PointOnLine,
        PlanarConstraint::Lines { relation, .. } => PlanarEquation::Lines(relation),
        PlanarConstraint::Distance {
            measure, parameter, ..
        } => PlanarEquation::Distance {
            measure,
            target: length_target(constraint, parameter, parameters)?,
        },
    })
}
fn length_target(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
) -> Result<f64, SketchError> {
    let quantity = parameters
        .model_value(parameter)
        .map_err(|cause| SketchError::Parameter {
            constraint: constraint.id,
            parameter,
            cause,
        })?;
    if quantity.dimension != Dimension::Length
        || !quantity.value.is_finite()
        || quantity.value < 0.0
    {
        return Err(SketchError::Definition(format!(
            "sketch constraint {:?} parameter {parameter} value {quantity:?}: expected a healthy finite nonnegative working-unit length",
            constraint.id
        )));
    }
    Ok(quantity.value)
}

fn radius_target(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
) -> Result<f64, SketchError> {
    let value = length_target(constraint, parameter, parameters)?;
    if value == 0.0 {
        return Err(SketchError::Definition(format!(
            "sketch constraint {:?} radius parameter {parameter} value {value}: expected a strictly positive working-unit length",
            constraint.id
        )));
    }
    Ok(value)
}
