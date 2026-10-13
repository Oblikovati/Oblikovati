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
    LineTangent { line: usize, curve: usize },
    CircularTangent([usize; 2]),
    EqualRadius([usize; 2]),
}
impl SketchPlacementRelation {
    pub(crate) fn resolve(
        self,
        members: &[SketchEntityId],
    ) -> Result<SketchGeometricConstraintCreation, SketchError> {
        let member = |index| indexed_entity(members, index);
        Ok(match self {
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
            Self::Distance([a, b]) => MeasuredDimensionOperands::Distance {
                points: [indexed_entity(points, a)?, indexed_entity(points, b)?],
                measure: DistanceMeasure::Aligned,
            },
            Self::Diameter(index) => MeasuredDimensionOperands::Diameter {
                curve: indexed_entity(members, index)?,
            },
        };
        Ok(Some(SketchDimensionalConstraintCreation::Measured(operand)))
    }
}
