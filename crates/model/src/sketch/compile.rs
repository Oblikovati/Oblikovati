// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarConstraint, Sketch2D, SketchConstraint, SketchError, SketchRelationId,
    residual::{PlanarEquation, PreparedConstraint},
};
use crate::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
};
use oblikovati_solve::residual::ResidualBlock;

pub(super) struct PreparedSketch {
    pub values: Vec<f64>,
    pub origins: Vec<SketchRelationId>,
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
    let (sources, origins) = authored_sources(sketch, parameters)?;
    let mut prepared = PreparedSketch {
        origins,
        sources,
        values: sketch.variable_values(),
        parameter_dependencies: parameter_dependencies(sketch),
    };
    append_arc_sources(sketch, &mut prepared)?;
    Ok(prepared)
}
fn authored_sources(
    sketch: &Sketch2D,
    parameters: &ParameterCollection<u64>,
) -> Result<(Vec<PreparedConstraint>, Vec<SketchRelationId>), SketchError> {
    sketch
        .constraints
        .iter()
        .map(|constraint| {
            Ok((
                prepared_source(sketch, constraint, parameters)?,
                SketchRelationId::Authored(constraint.id),
            ))
        })
        .collect::<Result<Vec<_>, SketchError>>()
        .map(|sources| sources.into_iter().unzip())
}
fn append_arc_sources(sketch: &Sketch2D, prepared: &mut PreparedSketch) -> Result<(), SketchError> {
    for arc in &sketch.arcs {
        prepared.sources.push(PreparedConstraint {
            equation: PlanarEquation::ArcCircularity,
            projection: None,
            variables: sketch.arc_columns(arc)?,
        });
        prepared
            .origins
            .push(SketchRelationId::ArcCircularity(arc.id));
    }
    Ok(())
}
fn parameter_dependencies(sketch: &Sketch2D) -> Vec<u64> {
    let mut dependencies = Vec::new();
    for constraint in &sketch.constraints {
        if let PlanarConstraint::Distance { parameter, .. }
        | PlanarConstraint::Radius { parameter, .. }
        | PlanarConstraint::Diameter { parameter, .. }
        | PlanarConstraint::Offset { parameter, .. }
        | PlanarConstraint::TangentDistance { parameter, .. }
        | PlanarConstraint::ArcLength { parameter, .. }
        | PlanarConstraint::Angle { parameter, .. }
        | PlanarConstraint::ThreePointAngle { parameter, .. } = constraint.relation
            && !dependencies.contains(&parameter)
        {
            dependencies.push(parameter);
        }
    }
    dependencies
}

fn prepared_source(
    sketch: &Sketch2D,
    constraint: &SketchConstraint,
    parameters: &ParameterCollection<u64>,
) -> Result<PreparedConstraint, SketchError> {
    let equation = equation(sketch, constraint, parameters)?;
    super::circular_compile::prepare_relation(sketch, &constraint.relation, equation)
}

fn equation(
    sketch: &Sketch2D,
    constraint: &SketchConstraint,
    parameters: &ParameterCollection<u64>,
) -> Result<PlanarEquation, SketchError> {
    Ok(match constraint.relation {
        PlanarConstraint::Coincident { .. } | PlanarConstraint::Concentric { .. } => {
            PlanarEquation::Coincident
        }
        PlanarConstraint::PointOnCircularCurve { .. } => PlanarEquation::PointOnCircle,
        PlanarConstraint::LineTangent { side, .. } => PlanarEquation::LineTangent(side),
        PlanarConstraint::CircularTangent { contact, .. } => PlanarEquation::CircleTangent(contact),
        PlanarConstraint::TangentDistance { .. } => {
            super::tangent_distance_compile::prepare_distance(sketch, constraint, parameters)?
        }
        PlanarConstraint::Offset {
            parameter,
            side,
            linear_diameter,
            ..
        } => PlanarEquation::Offset {
            target: offset_target(constraint, parameter, parameters, linear_diameter)?,
            side,
        },
        PlanarConstraint::EqualRadius { .. } => PlanarEquation::EqualRadius,
        PlanarConstraint::Diameter { parameter, .. } => {
            PlanarEquation::Radius(diameter_radius(constraint, parameter, parameters)?)
        }
        PlanarConstraint::ArcLength { arc, parameter } => PlanarEquation::ArcLength {
            target: positive_length_target(constraint, parameter, parameters, "arc length")?,
            counter_clockwise: sketch.arc(arc)?.counter_clockwise,
        },
        PlanarConstraint::Radius { parameter, .. } => PlanarEquation::Radius(
            positive_length_target(constraint, parameter, parameters, "radius")?,
        ),
        PlanarConstraint::Horizontal { .. } => PlanarEquation::Coordinate(1),
        PlanarConstraint::Vertical { .. } => PlanarEquation::Coordinate(0),
        PlanarConstraint::Fix { position, .. } => PlanarEquation::Fix(position),
        PlanarConstraint::ArcMidpoint { arc, .. } => {
            PlanarEquation::ArcMidpoint(sketch.arc(arc)?.counter_clockwise)
        }
        PlanarConstraint::Midpoint { .. } => PlanarEquation::Midpoint,
        PlanarConstraint::PointOnLine { .. } => PlanarEquation::PointOnLine,
        PlanarConstraint::Angle {
            parameter, turn, ..
        }
        | PlanarConstraint::ThreePointAngle {
            parameter, turn, ..
        } => PlanarEquation::Angle {
            target: angle_target(constraint, parameter, parameters)?,
            turn,
        },
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
    dimensional_target(
        constraint,
        parameter,
        parameters,
        DimensionDomain {
            kind: Dimension::Length,
            minimum: 0.0,
            maximum: f64::INFINITY,
            label: "length",
            expected: "a healthy finite nonnegative working-unit length",
        },
    )
}

pub(super) fn positive_length_target(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
    measure: &str,
) -> Result<f64, SketchError> {
    let value = length_target(constraint, parameter, parameters)?;
    if value == 0.0 {
        return Err(SketchError::Definition(format!(
            "sketch constraint {:?} {measure} parameter {parameter} value {value}: expected a strictly positive working-unit length",
            constraint.id
        )));
    }
    Ok(value)
}

fn parameter_quantity(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
) -> Result<Quantity, SketchError> {
    parameters
        .model_value(parameter)
        .map_err(|cause| SketchError::Parameter {
            constraint: constraint.id,
            parameter,
            cause,
        })
}

fn angle_target(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
) -> Result<f64, SketchError> {
    dimensional_target(
        constraint,
        parameter,
        parameters,
        DimensionDomain {
            kind: Dimension::Angle,
            minimum: 0.0,
            maximum: std::f64::consts::PI,
            label: "angle",
            expected: "a healthy finite angular quantity in [0, pi] radians",
        },
    )
}
pub(super) struct DimensionDomain {
    pub kind: Dimension,
    pub minimum: f64,
    pub maximum: f64,
    pub label: &'static str,
    pub expected: &'static str,
}
pub(super) fn dimensional_target(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
    domain: DimensionDomain,
) -> Result<f64, SketchError> {
    let quantity = parameter_quantity(constraint, parameter, parameters)?;
    if quantity.dimension != domain.kind
        || !quantity.value.is_finite()
        || !(domain.minimum..=domain.maximum).contains(&quantity.value)
    {
        return Err(SketchError::Definition(format!(
            "sketch constraint {:?} {} parameter {parameter} value {quantity:?}: expected {}",
            constraint.id, domain.label, domain.expected
        )));
    }
    Ok(quantity.value)
}

pub(super) fn diameter_radius(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
) -> Result<f64, SketchError> {
    let diameter = positive_length_target(constraint, parameter, parameters, "diameter")?;
    half_dimension(
        constraint,
        parameter,
        diameter,
        "a representable positive half-diameter",
    )
}
fn offset_target(
    constraint: &SketchConstraint,
    parameter: u64,
    parameters: &ParameterCollection<u64>,
    linear_diameter: bool,
) -> Result<f64, SketchError> {
    let value = length_target(constraint, parameter, parameters)?;
    if !linear_diameter {
        return Ok(value);
    }
    half_dimension(
        constraint,
        parameter,
        value,
        "a representable half-dimension",
    )
}
pub(super) fn half_dimension(
    constraint: &SketchConstraint,
    parameter: u64,
    value: f64,
    expected: &str,
) -> Result<f64, SketchError> {
    let half = value / 2.0;
    if value != 0.0 && half == 0.0 {
        return Err(SketchError::Definition(format!(
            "sketch constraint {:?} dimension parameter {parameter} value {value:?}: expected {expected}",
            constraint.id
        )));
    }
    Ok(half)
}
