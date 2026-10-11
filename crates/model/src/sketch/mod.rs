// SPDX-License-Identifier: GPL-2.0-only

//! Application-owned planar geometry and constraints, independent from kernel topology.
mod angle;
mod available_targets;
pub use available_targets::SketchAvailableTargetAnalysis;
mod arc;
mod arc_length;
mod arc_midpoint;
mod arc_residual;
mod arc_sweep;
mod authoring;
mod centerline;
mod circle;
mod circle_residual;
mod circular_compile;
mod circular_frame;
mod circular_projection;
mod compile;
mod constraint;
mod construction;
mod dimensional_authoring;
mod geometric_authoring;
mod ground_constraint;
pub use dimensional_authoring::{
    MeasuredDimensionOperands, SeededDimensionOperands, SketchDimensionalConstraintCreation,
};
mod geometric_constraints;
pub use authoring::{SketchPointInput, SketchPrimitiveCreation};
pub use geometric_authoring::SketchGeometricConstraintCreation;
mod dependencies;
mod dimension_measurement;
mod dimension_placement;
mod dimension_settings;
pub use dimension_placement::SketchDimensionPlacement;
pub use dimension_settings::{SketchDimensionLimitRecipe, SketchDimensionLimits};
mod axis_collinear;
mod axis_operand;
mod ellipse;
mod ellipse_axis;
pub use axis_operand::SketchAxisOperand;
pub use ellipse::{SketchEllipse2D, SketchEllipseEvaluation};
mod entity_geometry;
mod entity_order;
mod error;
mod geometry;
pub(crate) use geometry::require_position;
mod inference;
mod inspection;
pub use inference::{
    SketchDirectionInference, SketchEndpointInference, SketchInferencePolicy,
    SketchLinePlacementInference,
};
mod translation;
pub use inspection::{SketchEntityInspection, SketchMovementOptions};
mod grounded;
mod line_residual;
mod offset_dimension;
mod plane;
pub(crate) use plane::{normalize_direction, point_difference, require_unit};
pub(crate) use recipe_compare::same_components;
mod recipe;
mod recipe_compare;
mod residual;
mod shared_contact;
mod solve;
mod source;
mod spline;
pub use spline::{SketchOffsetSpline2D, SketchSpline2D, SketchSplineKind};
mod tangent;
mod tangent_distance;
mod tangent_distance_compile;
mod tangent_distance_residual;
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
pub use plane::{SketchFrameAxis, SketchPlane, SketchPlaneOptions, SketchPlaneRecipe};
pub use recipe::{SKETCH_RECIPE_VERSION, SketchRecipe};
pub use solve::SketchSolveOutcome;
pub use source::SketchRelationId;
pub use tangent::{CircleContact, LineContactSide};
pub use tangent_distance::{TangentDistanceContact, TangentDistanceOptions, TangentLean};

/// Ordered sketch ownership; shared endpoints use the same local point identity.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct Sketch2D {
    entity_order: Vec<SketchEntityId>,
    construction_curves: std::collections::BTreeSet<SketchEntityId>,
    centerlines: std::collections::BTreeSet<SketchEntityId>,
    points: Vec<SketchPoint2D>,
    lines: Vec<SketchLine2D>,
    circles: Vec<SketchCircle2D>,
    arcs: Vec<SketchArc2D>,
    ellipses: Vec<SketchEllipse2D>,
    splines: Vec<SketchSpline2D>,
    offset_splines: Vec<SketchOffsetSpline2D>,
    constraints: Vec<SketchConstraint>,
    reference_dimensions: std::collections::BTreeSet<SketchConstraintId>,
    dimension_drive_limits: std::collections::BTreeMap<SketchConstraintId, SketchDimensionLimits>,
    dimension_text_points: std::collections::BTreeMap<SketchConstraintId, [f64; 2]>,
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
