// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchError,
    TangentDistanceOptions,
};
/// Constructor inputs separate direct scalar measurements from branches retained from current geometry.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchDimensionalConstraintCreation {
    Measured(MeasuredDimensionOperands),
    Seeded(SeededDimensionOperands),
}
/// Dimensions whose authored relation needs no geometric branch selection.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum MeasuredDimensionOperands {
    Distance {
        points: [SketchEntityId; 2],
        measure: DistanceMeasure,
    },
    EllipseRadius {
        ellipse: SketchEntityId,
    },
    EllipseMinorRadius {
        ellipse: SketchEntityId,
    },
    Radius {
        curve: SketchEntityId,
    },
    Diameter {
        curve: SketchEntityId,
    },
    ArcLength {
        arc: SketchEntityId,
    },
}
/// Branch-sensitive operands; arrays name their role and retain the API operand order.
#[derive(Clone, Debug, PartialEq)]
pub enum SeededDimensionOperands {
    Angle([SketchEntityId; 2]),
    /// Vertex, first ray point, second ray point.
    ThreePointAngle([SketchEntityId; 3]),
    Offset {
        point_line: [SketchEntityId; 2],
        linear_diameter: bool,
    },
    TangentDistance {
        line_curve: [SketchEntityId; 2],
        options: TangentDistanceOptions,
    },
}
mod validation;
impl SketchDimensionalConstraintCreation {
    /// Identify the scalar category before creating its owned source.
    /// Example: `parameters.insert_scalar_expression(id, name, source, input.dimension())?`.
    pub fn dimension(&self) -> crate::quantity::Dimension {
        if matches!(
            self,
            Self::Seeded(
                SeededDimensionOperands::Angle(_) | SeededDimensionOperands::ThreePointAngle(_)
            )
        ) {
            return crate::quantity::Dimension::Angle;
        }
        crate::quantity::Dimension::Length
    }

    /// Require current owner coordinates before retaining a branch.
    /// Example: `if input.requires_current_seed() { /* verify the owner geometry */ }`.
    pub fn requires_current_seed(&self) -> bool {
        matches!(self, Self::Seeded(_))
    }
    /// Append using existing native validation and branch selection; identities and parameter ownership belong to the document.
    /// Example: `input.append(&mut geometry, constraint_id, parameter_id)?;` on an isolated owner candidate.
    pub fn append(
        &self,
        geometry: &mut Sketch2D,
        id: SketchConstraintId,
        parameter: u64,
    ) -> Result<(), SketchError> {
        match self {
            Self::Measured(input) => geometry.add_constraint(id, input.relation(parameter)),
            Self::Seeded(input) => input.append(geometry, id, parameter),
        }
    }
}
impl MeasuredDimensionOperands {
    fn relation(&self, parameter: u64) -> PlanarConstraint {
        match *self {
            Self::Distance { points, measure } => PlanarConstraint::Distance {
                points,
                measure,
                parameter,
            },
            Self::EllipseRadius { ellipse } => {
                PlanarConstraint::EllipseRadius { ellipse, parameter }
            }
            Self::EllipseMinorRadius { ellipse } => {
                PlanarConstraint::EllipseMinorRadius { ellipse, parameter }
            }
            Self::Radius { curve } => PlanarConstraint::Radius { curve, parameter },
            Self::Diameter { curve } => PlanarConstraint::Diameter { curve, parameter },
            Self::ArcLength { arc } => PlanarConstraint::ArcLength { arc, parameter },
        }
    }
}
impl SeededDimensionOperands {
    fn append(
        &self,
        geometry: &mut Sketch2D,
        id: SketchConstraintId,
        parameter: u64,
    ) -> Result<(), SketchError> {
        match self {
            Self::Angle(lines) => geometry.add_angle(id, *lines, parameter),
            Self::ThreePointAngle([vertex, first, second]) => {
                geometry.add_three_point_angle(id, *vertex, [*first, *second], parameter)
            }
            Self::Offset {
                point_line: [point, line],
                linear_diameter,
            } => geometry.add_offset_dimension(id, *point, *line, parameter, *linear_diameter),
            Self::TangentDistance {
                line_curve: [line, curve],
                options,
            } => geometry.add_tangent_distance(id, *line, *curve, parameter, *options),
        }
    }
}
