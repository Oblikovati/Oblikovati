// SPDX-License-Identifier: GPL-2.0-only
use super::{
    AngleTurn, CircleContact, LineContactSide, Sketch2D, SketchEntityId, SketchError,
    geometry::require_position,
};

/// Stable constraint identity, separate from the point/curve identity namespace.
#[derive(
    Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord, serde::Serialize, serde::Deserialize,
)]
pub struct SketchConstraintId(pub u64);
/// Application distance measure; this is not a wire DTO or API enum replacement.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case", deny_unknown_fields)]
pub enum DistanceMeasure {
    Aligned,
    Horizontal,
    Vertical,
}
/// Application-owned relation between two analytic planar lines.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case", deny_unknown_fields)]
pub enum LineRelation {
    Parallel,
    Perpendicular,
    Collinear,
    EqualLength,
}
/// Authored references remain intact when numerical values change.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case", deny_unknown_fields)]
pub enum PlanarConstraint {
    Coincident {
        points: [SketchEntityId; 2],
    },
    Horizontal {
        points: [SketchEntityId; 2],
    },
    Vertical {
        points: [SketchEntityId; 2],
    },
    EllipseAxisAlignment {
        ellipse: SketchEntityId,
        major_axis: bool,
        vertical: bool,
    },
    LineHorizontal {
        line: SketchEntityId,
    },
    LineVertical {
        line: SketchEntityId,
    },
    Fix {
        point: SketchEntityId,
        position: [f64; 2],
    },
    Ground {
        points: Vec<super::SketchPoint2D>,
    },
    ArcMidpoint {
        point: SketchEntityId,
        arc: SketchEntityId,
    },
    Midpoint {
        point: SketchEntityId,
        line: SketchEntityId,
    },
    PointOnLine {
        point: SketchEntityId,
        line: SketchEntityId,
    },
    Offset {
        point: SketchEntityId,
        line: SketchEntityId,
        parameter: u64,
        side: LineContactSide,
        linear_diameter: bool,
    },
    Angle {
        lines: [SketchEntityId; 2],
        parameter: u64,
        turn: AngleTurn,
    },
    ThreePointAngle {
        vertex: SketchEntityId,
        points: [SketchEntityId; 2],
        parameter: u64,
        turn: AngleTurn,
    },
    AxisCollinear {
        axes: [super::SketchAxisOperand; 2],
    },
    AxisRelation {
        axes: [super::SketchAxisOperand; 2],
        perpendicular: bool,
    },
    Lines {
        lines: [SketchEntityId; 2],
        relation: LineRelation,
    },
    PointOnCircularCurve {
        point: SketchEntityId,
        curve: SketchEntityId,
    },
    Concentric {
        curves: [SketchEntityId; 2],
    },
    EqualRadius {
        curves: [SketchEntityId; 2],
    },
    Diameter {
        curve: SketchEntityId,
        parameter: u64,
    },
    ArcLength {
        arc: SketchEntityId,
        parameter: u64,
    },
    EllipseRadius {
        ellipse: SketchEntityId,
        parameter: u64,
    },
    OffsetSplineDistance {
        offset: SketchEntityId,
        parameter: u64,
        side: super::LineContactSide,
    },
    EllipseMinorRadius {
        ellipse: SketchEntityId,
        parameter: u64,
    },
    Radius {
        curve: SketchEntityId,
        parameter: u64,
    },
    LineTangent {
        line: SketchEntityId,
        curve: SketchEntityId,
        side: LineContactSide,
    },
    TangentDistance {
        line: SketchEntityId,
        curve: SketchEntityId,
        parameter: u64,
        options: super::TangentDistanceOptions,
        contact: super::TangentDistanceContact,
    },
    CircularTangent {
        curves: [SketchEntityId; 2],
        contact: CircleContact,
    },
    Distance {
        points: [SketchEntityId; 2],
        measure: DistanceMeasure,
        parameter: u64,
    },
}
/// One ordered constraint with identity for diagnostics and later persistence.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchConstraint {
    pub id: SketchConstraintId,
    pub relation: PlanarConstraint,
}
impl Sketch2D {
    /// Add a validated relation; example: `sketch.add_constraint(id, PlanarConstraint::Horizontal { points: [a, b] })?`.
    pub fn add_constraint(
        &mut self,
        id: SketchConstraintId,
        relation: PlanarConstraint,
    ) -> Result<(), SketchError> {
        if id.0 == 0
            || self
                .constraints
                .iter()
                .any(|constraint| constraint.id == id)
        {
            return Err(SketchError::Definition(format!(
                "sketch constraint {id:?}: expected a unique nonzero identity"
            )));
        }
        self.validate_constraint_relation(&relation)?;
        self.constraints.push(SketchConstraint { id, relation });
        Ok(())
    }
    pub(crate) fn validate_constraint_relation(
        &self,
        relation: &PlanarConstraint,
    ) -> Result<(), SketchError> {
        for point in self.constraint_points(relation)? {
            self.point(point)?;
        }
        self.validate_contact_ownership(relation)?;
        if let PlanarConstraint::Ground { points } = relation {
            super::ground_constraint::validate_anchors(points)?;
        }
        if let PlanarConstraint::Fix { position, .. } = relation {
            require_position(*position)?;
        }
        Ok(())
    }
    pub(super) fn constraint_points(
        &self,
        relation: &PlanarConstraint,
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        Ok(match relation {
            PlanarConstraint::Coincident { points }
            | PlanarConstraint::Horizontal { points }
            | PlanarConstraint::Vertical { points }
            | PlanarConstraint::Distance { points, .. } => points.to_vec(),
            PlanarConstraint::Fix { point, .. } => vec![*point],
            PlanarConstraint::Ground { points } => points.iter().map(|point| point.id).collect(),
            PlanarConstraint::LineHorizontal { line } | PlanarConstraint::LineVertical { line } => {
                self.line(*line)?.endpoints.to_vec()
            }
            PlanarConstraint::ArcMidpoint { point, arc } => {
                let arc = self.arc(*arc)?;
                vec![*point, arc.center, arc.start, arc.end]
            }
            PlanarConstraint::Midpoint { point, line }
            | PlanarConstraint::PointOnLine { point, line }
            | PlanarConstraint::Offset { point, line, .. } => {
                let endpoints = self.line(*line)?.endpoints;
                vec![*point, endpoints[0], endpoints[1]]
            }
            PlanarConstraint::PointOnCircularCurve { point, curve } => {
                self.circular_frame(*curve)?;
                vec![*point]
            }
            PlanarConstraint::Concentric { curves }
            | PlanarConstraint::EqualRadius { curves }
            | PlanarConstraint::CircularTangent { curves, .. } => {
                for curve in curves {
                    self.circular_frame(*curve)?;
                }
                Vec::new()
            }
            PlanarConstraint::LineTangent { line, curve, .. }
            | PlanarConstraint::TangentDistance { line, curve, .. } => {
                self.circular_frame(*curve)?;
                self.line(*line)?.endpoints.to_vec()
            }
            PlanarConstraint::AxisRelation { axes, .. }
            | PlanarConstraint::AxisCollinear { axes } => {
                for axis in axes {
                    axis.columns(self)?;
                }
                Vec::new()
            }
            PlanarConstraint::OffsetSplineDistance { offset, .. } => {
                self.offset_spline(*offset)?;
                Vec::new()
            }
            PlanarConstraint::EllipseRadius { ellipse, .. }
            | PlanarConstraint::EllipseMinorRadius { ellipse, .. }
            | PlanarConstraint::EllipseAxisAlignment { ellipse, .. } => {
                self.ellipse(*ellipse)?;
                Vec::new()
            }
            PlanarConstraint::ArcLength { arc, .. } => {
                let arc = self.arc(*arc)?;
                vec![arc.center, arc.start, arc.end]
            }
            PlanarConstraint::Radius { curve, .. } | PlanarConstraint::Diameter { curve, .. } => {
                self.circular_frame(*curve)?;
                Vec::new()
            }
            PlanarConstraint::ThreePointAngle { vertex, points, .. } => {
                vec![*vertex, points[0], *vertex, points[1]]
            }
            PlanarConstraint::Lines { lines, .. } | PlanarConstraint::Angle { lines, .. } => {
                let first = self.line(lines[0])?.endpoints;
                let second = self.line(lines[1])?.endpoints;
                [first, second].concat()
            }
        })
    }
}
