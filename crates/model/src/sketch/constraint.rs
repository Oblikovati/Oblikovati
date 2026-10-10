// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, geometry::require_position};

/// Stable constraint identity, separate from the point/curve identity namespace.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchConstraintId(pub u64);
/// Application distance measure; this is not a wire DTO or API enum replacement.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum DistanceMeasure {
    Aligned,
    Horizontal,
    Vertical,
}
/// Application-owned relation between two analytic planar lines.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum LineRelation {
    Parallel,
    Perpendicular,
    Collinear,
    EqualLength,
}
/// Authored references remain intact when numerical values change.
#[derive(Clone, Debug, PartialEq)]
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
    Fix {
        point: SketchEntityId,
        position: [f64; 2],
    },
    Midpoint {
        point: SketchEntityId,
        line: SketchEntityId,
    },
    PointOnLine {
        point: SketchEntityId,
        line: SketchEntityId,
    },
    Lines {
        lines: [SketchEntityId; 2],
        relation: LineRelation,
    },
    PointOnCircle {
        point: SketchEntityId,
        circle: SketchEntityId,
    },
    Concentric {
        circles: [SketchEntityId; 2],
    },
    EqualRadius {
        circles: [SketchEntityId; 2],
    },
    Radius {
        circle: SketchEntityId,
        parameter: u64,
    },
    Distance {
        points: [SketchEntityId; 2],
        measure: DistanceMeasure,
        parameter: u64,
    },
}
/// One ordered constraint with identity for diagnostics and later persistence.
#[derive(Clone, Debug, PartialEq)]
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
        for point in self.constraint_points(&relation)? {
            self.point(point)?;
        }
        if let PlanarConstraint::Fix { position, .. } = relation {
            require_position(position)?;
        }
        self.constraints.push(SketchConstraint { id, relation });
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
            PlanarConstraint::Midpoint { point, line }
            | PlanarConstraint::PointOnLine { point, line } => {
                let endpoints = self.line(*line)?.endpoints;
                vec![*point, endpoints[0], endpoints[1]]
            }
            PlanarConstraint::PointOnCircle { point, circle } => {
                vec![*point, self.circle(*circle)?.center]
            }
            PlanarConstraint::Concentric { circles }
            | PlanarConstraint::EqualRadius { circles } => circles
                .iter()
                .map(|id| Ok(self.circle(*id)?.center))
                .collect::<Result<Vec<_>, SketchError>>()?,
            PlanarConstraint::Radius { circle, .. } => vec![self.circle(*circle)?.center],
            PlanarConstraint::Lines { lines, .. } => {
                let first = self.line(lines[0])?.endpoints;
                let second = self.line(lines[1])?.endpoints;
                [first, second].concat()
            }
        })
    }
}
