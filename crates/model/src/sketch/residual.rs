// SPDX-License-Identifier: GPL-2.0-only
use super::{
    CircleContact, DistanceMeasure, LineContactSide, LineRelation, arc_residual, circle_residual,
    line_residual, tangent_residual,
};
use oblikovati_solve::{
    ad::{DualNumber, DualVector},
    residual::{ResidualError, ResidualSource},
};

#[derive(Clone)]
pub(super) enum PlanarEquation {
    LineAngle {
        target: f64,
    },
    Angle {
        target: f64,
        turn: super::AngleTurn,
    },
    ArcLength {
        target: f64,
        counter_clockwise: bool,
    },
    ArcSweep {
        target: f64,
        counter_clockwise: bool,
    },
    AxisCollinear {
        layouts: [super::axis_operand::AxisLayout; 2],
    },
    AxisRelation {
        layouts: [super::axis_operand::AxisLayout; 2],
        perpendicular: bool,
    },
    AxisAlignment {
        quarter_turn: bool,
    },
    ArcCircularity,
    PatternPointLink([f64; 2]),
    ArcMidpoint(bool),
    Coincident,
    Coordinate(usize),
    Fix([f64; 2]),
    Ground(Vec<[f64; 2]>),
    Midpoint,
    PointOnLine,
    Offset {
        target: f64,
        side: LineContactSide,
    },
    Lines {
        relation: LineRelation,
        shared_endpoints: usize,
    },
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
    ScalarLength {
        target: f64,
        domain: super::scalar_length::LengthVariableDomain,
    },
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
            PlanarEquation::Ground(ref positions) => positions.len() * 2,
            PlanarEquation::AxisCollinear { .. }
            | PlanarEquation::ArcMidpoint(_)
            | PlanarEquation::Coincident
            | PlanarEquation::PatternPointLink(_)
            | PlanarEquation::Fix(_)
            | PlanarEquation::Midpoint => 2,
            PlanarEquation::Lines {
                relation: LineRelation::Collinear,
                shared_endpoints,
            } => 2 - shared_endpoints,
            _ => 1,
        }
    }
    fn variable_count(&self) -> usize {
        match self.equation {
            PlanarEquation::Ground(ref positions) => positions.len() * 2,
            PlanarEquation::AxisRelation { layouts, .. } => {
                layouts.iter().map(|layout| layout.width()).sum()
            }
            PlanarEquation::AxisCollinear { layouts } => {
                4 + layouts.iter().map(|layout| layout.width()).sum::<usize>()
            }
            _ => self.fixed_variable_count(),
        }
    }
    fn fixed_variable_count(&self) -> usize {
        match self.equation {
            PlanarEquation::Fix(_) | PlanarEquation::EqualRadius => 2,
            PlanarEquation::ScalarLength { .. } | PlanarEquation::AxisAlignment { .. } => 1,
            PlanarEquation::PointOnCircle => 5,
            PlanarEquation::LineTangent(_)
            | PlanarEquation::PinnedLineTangent { .. }
            | PlanarEquation::TangentDistance { .. }
            | PlanarEquation::PinnedTangentDistance { .. } => 7,
            PlanarEquation::PinnedCircularTangent(_) | PlanarEquation::ArcMidpoint(_) => 8,
            PlanarEquation::CircleTangent(_)
            | PlanarEquation::ArcCircularity
            | PlanarEquation::ArcLength { .. }
            | PlanarEquation::ArcSweep { .. } => 6,
            PlanarEquation::Midpoint
            | PlanarEquation::PointOnLine
            | PlanarEquation::Offset { .. } => 6,
            PlanarEquation::Lines { .. } | PlanarEquation::Angle { .. } => 8,
            _ => 4,
        }
    }
}
impl ResidualSource for PreparedConstraint {
    fn tolerance_kind(&self) -> oblikovati_solve::residual::ResidualTolerance {
        if matches!(
            self.equation,
            PlanarEquation::AxisAlignment { .. } | PlanarEquation::AxisRelation { .. }
        ) {
            return oblikovati_solve::residual::ResidualTolerance::Unitless;
        }
        oblikovati_solve::residual::ResidualTolerance::ModelRelative
    }

    fn equation_tolerance(&self, row: usize) -> oblikovati_solve::residual::ResidualTolerance {
        if matches!(self.equation, PlanarEquation::AxisCollinear { .. }) {
            return if row == 0 {
                oblikovati_solve::residual::ResidualTolerance::Unitless
            } else {
                oblikovati_solve::residual::ResidualTolerance::ModelLength
            };
        }
        self.tolerance_kind()
    }

    fn equation_unit(&self, row: usize) -> oblikovati_solve::residual::ResidualValueUnit {
        use oblikovati_solve::residual::ResidualValueUnit::{
            Dimensionless, Length, Radians, Unknown,
        };
        if row >= self.equations() {
            return Unknown;
        }
        match &self.equation {
            PlanarEquation::Coincident
            | PlanarEquation::Coordinate(_)
            | PlanarEquation::Fix(_)
            | PlanarEquation::Ground(_)
            | PlanarEquation::PatternPointLink(_)
            | PlanarEquation::Midpoint
            | PlanarEquation::PointOnLine
            | PlanarEquation::PointOnCircle
            | PlanarEquation::EqualRadius
            | PlanarEquation::ScalarLength { .. }
            | PlanarEquation::Distance { .. }
            | PlanarEquation::ArcLength { .. } => Length,
            PlanarEquation::LineAngle { .. }
            | PlanarEquation::Angle { .. }
            | PlanarEquation::ArcSweep { .. }
            | PlanarEquation::AxisAlignment { .. }
            | PlanarEquation::AxisRelation { .. } => Radians,
            PlanarEquation::AxisCollinear { .. } => match row {
                0 => Radians,
                1 => Length,
                _ => Unknown,
            },
            PlanarEquation::Lines {
                relation: LineRelation::EqualLength,
                ..
            } => Length,
            PlanarEquation::Lines {
                relation: LineRelation::Parallel | LineRelation::Perpendicular,
                ..
            } => Dimensionless,
            PlanarEquation::Lines {
                relation: LineRelation::Collinear,
                ..
            }
            | PlanarEquation::ArcCircularity
            | PlanarEquation::ArcMidpoint(_)
            | PlanarEquation::Offset { .. }
            | PlanarEquation::LineTangent(_)
            | PlanarEquation::CircleTangent(_)
            | PlanarEquation::PinnedLineTangent { .. }
            | PlanarEquation::PinnedCircularTangent(_)
            | PlanarEquation::TangentDistance { .. }
            | PlanarEquation::PinnedTangentDistance { .. } => Unknown,
        }
    }
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
            PlanarEquation::AxisCollinear { layouts } => {
                super::axis_collinear::collinear_residual(variables, layouts)
            }
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
            PlanarEquation::LineAngle { target } => {
                super::angle::line_angle_residual(variables, target)
            }
            PlanarEquation::ArcLength {
                target,
                counter_clockwise,
            } => super::arc_length::length_residual(variables, target, counter_clockwise),
            PlanarEquation::ArcSweep {
                target,
                counter_clockwise,
            } => super::arc_sweep::sweep_residual(variables, target, counter_clockwise),
            PlanarEquation::AxisRelation {
                layouts,
                perpendicular,
            } => super::ellipse_axis::pair_residual(variables, layouts, perpendicular),
            PlanarEquation::AxisAlignment { quarter_turn } => {
                super::ellipse_axis::alignment_residual(&variables[0], quarter_turn)
            }
            PlanarEquation::ArcCircularity => arc_residual::circularity(variables),
            PlanarEquation::PatternPointLink(offset) => Ok(vec![
                variables[2].minus(&variables[0])?.shifted(-offset[0])?,
                variables[3].minus(&variables[1])?.shifted(-offset[1])?,
            ]),
            PlanarEquation::Coincident => Ok(point(variables, 0)?
                .minus(&point(variables, 2)?)?
                .components()
                .to_vec()),
            PlanarEquation::Coordinate(axis) => {
                Ok(vec![variables[axis].minus(&variables[axis + 2])?])
            }
            PlanarEquation::Fix(position) => {
                super::ground_constraint::coordinate_residuals(variables, &[position])
            }
            PlanarEquation::Ground(ref positions) => {
                super::ground_constraint::coordinate_residuals(variables, positions)
            }
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
            PlanarEquation::ScalarLength { target, domain } => {
                super::scalar_length::scalar_length_target(&variables[0], target, domain)
            }
            PlanarEquation::Lines {
                relation,
                shared_endpoints,
            } => line_residual::line_relation(variables, relation, shared_endpoints),
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
pub(super) fn distance(
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
