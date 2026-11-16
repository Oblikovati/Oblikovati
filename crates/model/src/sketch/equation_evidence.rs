// SPDX-License-Identifier: GPL-2.0-only
//! Input attribution for prepared equations; this is not interval evidence.
use super::{
    DistanceMeasure, Sketch2D, SketchRelationId,
    residual::{PlanarEquation, PreparedConstraint},
};

/// The authored source of a prepared numeric target.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum SketchTargetSource {
    Parameter(u64),
    AuthoredPosition,
    Unknown,
}
/// A prepared binary64 target and its source, without an uncertainty bound.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchResidualTarget {
    pub value: f64,
    pub source: SketchTargetSource,
}
#[derive(Clone, Debug)]
pub(super) struct EquationEvidence {
    pub operands: Option<Vec<usize>>,
    pub targets: Option<Vec<SketchResidualTarget>>,
}

pub(super) fn rows(
    sketch: &Sketch2D,
    origin: SketchRelationId,
    source: &PreparedConstraint,
) -> Vec<EquationEvidence> {
    let parameter = authored_parameter(sketch, origin);
    (0..source.equations())
        .map(|row| EquationEvidence {
            operands: operand_indices(source, row),
            targets: targets(&source.equation, row, parameter),
        })
        .collect()
}
fn authored_parameter(sketch: &Sketch2D, origin: SketchRelationId) -> Option<u64> {
    let SketchRelationId::Authored(id) = origin else {
        return None;
    };
    sketch
        .constraints()
        .iter()
        .find(|constraint| constraint.id == id)
        .and_then(|constraint| super::dependencies::dimension_parameter(&constraint.relation))
}
fn operand_indices(source: &PreparedConstraint, row: usize) -> Option<Vec<usize>> {
    if source.projection.is_some() {
        return None;
    }
    let indices = match &source.equation {
        PlanarEquation::Distance { measure, .. } => distance_indices(*measure),
        other => linear_indices(other, row)?,
    };
    indices
        .iter()
        .all(|index| *index < source.variables.len())
        .then_some(indices)
}
fn linear_indices(equation: &PlanarEquation, row: usize) -> Option<Vec<usize>> {
    Some(match equation {
        PlanarEquation::Coincident | PlanarEquation::PatternPointLink(_) => vec![row, row + 2],
        PlanarEquation::Coordinate(axis) => vec![*axis, *axis + 2],
        PlanarEquation::Fix(_) | PlanarEquation::Ground(_) => vec![row],
        PlanarEquation::Midpoint => vec![row, row + 2, row + 4],
        PlanarEquation::ScalarLength { .. } => vec![0],
        _ => return None,
    })
}
fn distance_indices(measure: DistanceMeasure) -> Vec<usize> {
    match measure {
        DistanceMeasure::Horizontal => vec![0, 2],
        DistanceMeasure::Vertical => vec![1, 3],
        DistanceMeasure::Aligned => vec![0, 1, 2, 3],
    }
}

fn targets(
    equation: &PlanarEquation,
    row: usize,
    parameter: Option<u64>,
) -> Option<Vec<SketchResidualTarget>> {
    let (value, source) = match equation {
        PlanarEquation::Coincident | PlanarEquation::Coordinate(_) | PlanarEquation::Midpoint => {
            return Some(vec![]);
        }
        PlanarEquation::Fix(position) => {
            (*position.get(row)?, SketchTargetSource::AuthoredPosition)
        }
        PlanarEquation::Ground(positions) => (
            *positions.get(row / 2)?.get(row % 2)?,
            SketchTargetSource::AuthoredPosition,
        ),
        PlanarEquation::Distance { target, .. } | PlanarEquation::ScalarLength { target, .. } => (
            *target,
            parameter.map_or(SketchTargetSource::Unknown, SketchTargetSource::Parameter),
        ),
        _ => return None,
    };
    Some(vec![SketchResidualTarget { value, source }])
}
