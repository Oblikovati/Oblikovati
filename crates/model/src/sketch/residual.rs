// SPDX-License-Identifier: GPL-2.0-only
use super::{
    CircleContact, DistanceMeasure, LineContactSide, LineRelation, arc_residual, circle_residual,
    line_residual, tangent_residual,
};
use oblikovati_solve::{
    ad::{DualNumber, DualVector},
    residual::{ResidualError, ResidualSource},
};

#[derive(Clone, Copy)]
pub(super) enum PlanarEquation {
    Angle {
        target: f64,
        turn: super::AngleTurn,
    },
    ArcLength {
        target: f64,
        counter_clockwise: bool,
    },
    ArcCircularity,
    ArcMidpoint(bool),
    Coincident,
    Coordinate(usize),
    Fix([f64; 2]),
    Midpoint,
    PointOnLine,
    Offset {
        target: f64,
        side: LineContactSide,
    },
    Lines(LineRelation),
    PointOnCircle,
    PinnedLineTangent {
        at_b: bool,
        side: LineContactSide,
    },
    PinnedCircularTangent(CircleContact),
    TangentDistance {
        target: f64,
        far_side: bool,
        contact: super::TangentDistanceContact,
        at_b: Option<bool>,
    },
    PinnedTangentDistance {
        target: f64,
        at_b: bool,
        side: LineContactSide,
    },
    EqualRadius,
    Radius(f64),
    LineTangent(LineContactSide),
    CircleTangent(CircleContact),
    Distance {
        measure: DistanceMeasure,
        target: f64,
    },
}
pub(super) struct PreparedConstraint {
    pub projection: Option<super::circular_projection::CircularProjection>,
    pub equation: PlanarEquation,
    pub variables: Vec<usize>,
}
impl PreparedConstraint {
    pub fn equations(&self) -> usize {
        match self.equation {
            PlanarEquation::ArcMidpoint(_)
            | PlanarEquation::Coincident
            | PlanarEquation::Fix(_)
            | PlanarEquation::Midpoint
            | PlanarEquation::Lines(LineRelation::Collinear) => 2,
            _ => 1,
        }
    }
    fn variable_count(&self) -> usize {
        match self.equation {
            PlanarEquation::Fix(_) | PlanarEquation::EqualRadius => 2,
            PlanarEquation::Radius(_) => 1,
            PlanarEquation::PointOnCircle => 5,
            PlanarEquation::LineTangent(_)
            | PlanarEquation::PinnedLineTangent { .. }
            | PlanarEquation::TangentDistance { .. }
            | PlanarEquation::PinnedTangentDistance { .. } => 7,
            PlanarEquation::PinnedCircularTangent(_) | PlanarEquation::ArcMidpoint(_) => 8,
            PlanarEquation::CircleTangent(_)
            | PlanarEquation::ArcCircularity
            | PlanarEquation::ArcLength { .. } => 6,
            PlanarEquation::Midpoint
            | PlanarEquation::PointOnLine
            | PlanarEquation::Offset { .. } => 6,
            PlanarEquation::Lines(_) | PlanarEquation::Angle { .. } => 8,
            _ => 4,
        }
    }
}
impl ResidualSource for PreparedConstraint {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        let expected = self
            .projection
            .as_ref()
            .map_or(self.variable_count(), |layout| layout.input_count());
        if variables.len() != expected {
            return Err(ResidualError::Domain(format!(
                "planar residual received {} variables: expected {}",
                variables.len(),
                expected
            )));
        }
        let projected = self
            .projection
            .as_ref()
            .map(|layout| layout.project(variables))
            .transpose()?;
        let variables = projected.as_deref().unwrap_or(variables);
        match self.equation {
            PlanarEquation::PinnedLineTangent { at_b, side } => {
                tangent_residual::pinned_line_tangent(variables, at_b, side)
            }
            PlanarEquation::TangentDistance {
                target,
                far_side,
                contact,
                at_b,
            } => super::tangent_distance_residual::distance_residual(
                variables, target, far_side, contact, at_b,
            ),
            PlanarEquation::PinnedTangentDistance { target, at_b, side } => {
                super::tangent_distance_residual::angular_residual(variables, target, at_b, side)
            }
            PlanarEquation::PinnedCircularTangent(contact) => {
                tangent_residual::pinned_circular_tangent(variables, contact)
            }
            PlanarEquation::ArcMidpoint(counter_clockwise) => {
                super::arc_midpoint::midpoint_residual(variables, counter_clockwise)
            }
            PlanarEquation::Angle { target, turn } => {
                super::angle::angle_residual(variables, target, turn)
            }
            PlanarEquation::ArcLength {
                target,
                counter_clockwise,
            } => super::arc_length::length_residual(variables, target, counter_clockwise),
            PlanarEquation::ArcCircularity => arc_residual::circularity(variables),
            PlanarEquation::Coincident => Ok(point(variables, 0)?
                .minus(&point(variables, 2)?)?
                .components()
                .to_vec()),
            PlanarEquation::Coordinate(axis) => {
                Ok(vec![variables[axis].minus(&variables[axis + 2])?])
            }
            PlanarEquation::Fix(position) => Ok(vec![
                variables[0].shifted(-position[0])?,
                variables[1].shifted(-position[1])?,
            ]),
            PlanarEquation::Midpoint => midpoint(variables),
            PlanarEquation::PointOnLine => line_residual::point_on_line(variables),
            PlanarEquation::Offset { target, side } => {
                super::offset_dimension::offset_residual(variables, target, side)
            }
            PlanarEquation::PointOnCircle => circle_residual::point_on_circle(variables),
            PlanarEquation::LineTangent(side) => tangent_residual::line_tangent(variables, side),
            PlanarEquation::CircleTangent(contact) => {
                tangent_residual::circle_tangent(variables, contact)
            }
            PlanarEquation::EqualRadius => circle_residual::equal_radius(variables),
            PlanarEquation::Radius(target) => circle_residual::radius_target(&variables[0], target),
            PlanarEquation::Lines(relation) => line_residual::line_relation(variables, relation),
            PlanarEquation::Distance { measure, target } => {
                Ok(vec![distance(variables, measure)?.shifted(-target)?])
            }
        }
    }
}
pub(super) fn point(
    variables: &[DualNumber],
    offset: usize,
) -> Result<DualVector<2>, ResidualError> {
    Ok(DualVector::new([
        variables[offset].clone(),
        variables[offset + 1].clone(),
    ])?)
}
fn midpoint(variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
    // Scale before summing so a representable midpoint does not overflow (#3704).
    let middle = point(variables, 2)?
        .scaled(0.5)?
        .plus(&point(variables, 4)?.scaled(0.5)?)?;
    Ok(point(variables, 0)?.minus(&middle)?.components().to_vec())
}
fn distance(
    variables: &[DualNumber],
    measure: DistanceMeasure,
) -> Result<DualNumber, ResidualError> {
    match measure {
        DistanceMeasure::Aligned => Ok(point(variables, 0)?
            .minus(&point(variables, 2)?)?
            .length()?),
        DistanceMeasure::Horizontal => Ok(variables[0].minus(&variables[2])?.abs()?),
        DistanceMeasure::Vertical => Ok(variables[1].minus(&variables[3])?.abs()?),
    }
}

#[cfg(test)]
#[path = "../../tests/sketch/residuals.rs"]
mod tests;
