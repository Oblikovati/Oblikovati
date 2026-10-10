// SPDX-License-Identifier: GPL-2.0-only

//! Application-owned planar geometry and constraints, independent from kernel topology.
mod angle;
mod arc;
mod arc_length;
mod arc_midpoint;
mod arc_residual;
mod arc_sweep;
mod circle;
mod circle_residual;
mod circular_compile;
mod circular_frame;
mod circular_projection;
mod compile;
mod constraint;
mod error;
mod geometry;
mod line_residual;
mod residual;
mod shared_contact;
mod solve;
mod source;
mod tangent;
mod tangent_residual;
mod variables;
pub use angle::AngleTurn;
pub use arc::SketchArc2D;
pub use circle::SketchCircle2D;
pub use constraint::{
    DistanceMeasure, LineRelation, PlanarConstraint, SketchConstraint, SketchConstraintId,
};
pub use error::SketchError;
pub use geometry::{SketchEntityId, SketchLine2D, SketchPoint2D};
pub use solve::SketchSolveOutcome;
pub use source::SketchRelationId;
pub use tangent::{CircleContact, LineContactSide};

/// Ordered sketch ownership; shared endpoints use the same local point identity.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct Sketch2D {
    points: Vec<SketchPoint2D>,
    lines: Vec<SketchLine2D>,
    circles: Vec<SketchCircle2D>,
    arcs: Vec<SketchArc2D>,
    constraints: Vec<SketchConstraint>,
}
impl Sketch2D {
    /// Inspect all points in stable insertion order; example: `sketch.points()`.
    pub fn points(&self) -> &[SketchPoint2D] {
        &self.points
    }
    /// Inspect analytic line endpoints; example: `sketch.lines()`.
    pub fn lines(&self) -> &[SketchLine2D] {
        &self.lines
    }
    /// Inspect authored constraints without bypassing validation; example: `sketch.constraints()`.
    pub fn constraints(&self) -> &[SketchConstraint] {
        &self.constraints
    }
    /// Resolve a point by local identity; example: `sketch.point(SketchEntityId(7))`.
    pub fn point(&self, id: SketchEntityId) -> Result<&SketchPoint2D, SketchError> {
        self.points
            .iter()
            .find(|point| point.id == id)
            .ok_or_else(|| {
                SketchError::Definition(format!("sketch point {id:?}: expected an existing point"))
            })
    }
    /// Resolve a line by local identity; example: `sketch.line(SketchEntityId(8))`.
    pub fn line(&self, id: SketchEntityId) -> Result<&SketchLine2D, SketchError> {
        self.lines.iter().find(|line| line.id == id).ok_or_else(|| {
            SketchError::Definition(format!("sketch line {id:?}: expected an existing line"))
        })
    }
}
