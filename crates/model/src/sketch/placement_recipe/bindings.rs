// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{
    DistanceMeasure, MeasuredDimensionOperands, PlanarConstraint,
    SketchDimensionalConstraintCreation, SketchEntityId, SketchGeometricConstraintCreation,
    path_member::indexed_entity,
};
/// Geometric intent in the local member namespace; branches use the owned seed geometry.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchPlacementRelation {
    LineHorizontal(usize),
    LineVertical(usize),
    Perpendicular([usize; 2]),
    Midpoint { point: usize, line: usize },
    LineTangent { line: usize, curve: usize },
    CircularTangent([usize; 2]),
    EqualRadius([usize; 2]),
}
impl SketchPlacementRelation {
    pub(crate) fn resolve(
        self,
        points: &[SketchEntityId],
        members: &[SketchEntityId],
    ) -> Result<SketchGeometricConstraintCreation, SketchError> {
        let member = |index| indexed_entity(members, index);
        use SketchGeometricConstraintCreation::Relation;
        Ok(match self {
            Self::LineHorizontal(line) => Relation(PlanarConstraint::LineHorizontal {
                line: member(line)?,
            }),
            Self::LineVertical(line) => Relation(PlanarConstraint::LineVertical {
                line: member(line)?,
            }),
            Self::Perpendicular([a, b]) => Relation(PlanarConstraint::Lines {
                lines: [member(a)?, member(b)?],
                relation: crate::sketch::LineRelation::Perpendicular,
            }),
            Self::Midpoint { point, line } => Relation(PlanarConstraint::Midpoint {
                point: indexed_entity(points, point)?,
                line: member(line)?,
            }),
            Self::LineTangent { line, curve } => SketchGeometricConstraintCreation::LineTangent {
                line: member(line)?,
                curve: member(curve)?,
            },
            Self::CircularTangent([a, b]) => SketchGeometricConstraintCreation::CircularTangent {
                curves: [member(a)?, member(b)?],
            },
            Self::EqualRadius([a, b]) => {
                SketchGeometricConstraintCreation::Relation(PlanarConstraint::EqualRadius {
                    curves: [member(a)?, member(b)?],
                })
            }
        })
    }
}
/// Field operands use local points/members; a steering angle changes placement but has no reference edge.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchPlacementBinding {
    Distance([usize; 2]),
    HorizontalDistance([usize; 2]),
    VerticalDistance([usize; 2]),
    Diameter(usize),
    SteeringAngle,
}
impl SketchPlacementBinding {
    /// Read the field's unit category; example: `field.binding.dimension()` for an input box.
    pub fn dimension(self) -> crate::quantity::Dimension {
        if matches!(self, Self::SteeringAngle) {
            return crate::quantity::Dimension::Angle;
        }
        crate::quantity::Dimension::Length
    }
    pub(crate) fn resolve(
        self,
        points: &[SketchEntityId],
        members: &[SketchEntityId],
    ) -> Result<Option<SketchDimensionalConstraintCreation>, SketchError> {
        let operand = match self {
            Self::SteeringAngle => return Ok(None),
            Self::Distance(pair) => distance_operand(points, pair, DistanceMeasure::Aligned)?,
            Self::HorizontalDistance(pair) => {
                distance_operand(points, pair, DistanceMeasure::Horizontal)?
            }
            Self::VerticalDistance(pair) => {
                distance_operand(points, pair, DistanceMeasure::Vertical)?
            }
            Self::Diameter(index) => MeasuredDimensionOperands::Diameter {
                curve: indexed_entity(members, index)?,
            },
        };
        Ok(Some(SketchDimensionalConstraintCreation::Measured(operand)))
    }
}

fn distance_operand(
    points: &[SketchEntityId],
    [a, b]: [usize; 2],
    measure: DistanceMeasure,
) -> Result<MeasuredDimensionOperands, SketchError> {
    Ok(MeasuredDimensionOperands::Distance {
        points: [indexed_entity(points, a)?, indexed_entity(points, b)?],
        measure,
    })
}
