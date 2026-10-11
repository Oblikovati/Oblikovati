// SPDX-License-Identifier: GPL-2.0-only
//! Associative construction frames from Go work_plane.go/work_plane_defs.go, aa8297d1.

mod collection;
mod construction;
mod recipe;
pub use super::work_geometry::WorkGeometryError as WorkPlaneError;
pub use super::work_geometry::WorkGeometryEvaluation as WorkPlaneEvaluation;
use super::{AuthoredFeature, FeatureId};
use crate::{parameter::ParameterCollection, sketch::SketchPlane};
pub use collection::{WorkPlaneFeatureRecipe, WorkPlaneProgramRecipe, WorkPlaneProgramRecipeError};
pub use recipe::{WorkPlaneDefinitionRecipe, WorkPlaneRecipe};

/// Origin construction frames, independent of collection positions or wire strings.
#[derive(Clone, Copy, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum WorkPlaneOrigin {
    XY,
    XZ,
    YZ,
}

/// References resolve by stable feature identity, never by nearest geometry.
#[derive(Clone, Copy, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum WorkPlaneReference {
    Origin(WorkPlaneOrigin),
    Feature(FeatureId),
}

/// Source-compatible length used by plane offsets and datum point coordinates.
pub use super::work_geometry::DatumLength as WorkPlaneDistance;

/// Source-compatible name for the single ordered datum definition.
pub use super::work_geometry::WorkGeometryDefinition as WorkPlaneDefinition;

/// Project-owned seam for current plane, axis, and point inputs to construction frames.
pub trait WorkPlaneResolver {
    /// Resolve one available current line; example: `resolver.axis(FeatureId(1))?`.
    fn axis(&self, feature: FeatureId) -> Result<super::work_axis::WorkAxisLine, WorkPlaneError>;
    /// Resolve one available current frame; example: `resolver.plane(FeatureId(1))?`.
    fn plane(&self, feature: FeatureId) -> Result<SketchPlane, WorkPlaneError>;
    /// Resolve one available current position; example: `resolver.point(FeatureId(2))?`.
    fn point(&self, feature: FeatureId) -> Result<[f64; 3], WorkPlaneError>;
}

impl WorkPlaneOrigin {
    /// Use the validated origin frame; example: `WorkPlaneOrigin::XZ.plane()`.
    pub fn plane(self) -> SketchPlane {
        match self {
            Self::XY => SketchPlane::xy(),
            Self::XZ => SketchPlane::xz(),
            Self::YZ => SketchPlane::yz(),
        }
    }
}

impl WorkPlaneDefinition {
    /// Evaluate exact construction coordinates without consulting display geometry; example: `definition.evaluate(&parameters, &resolver)?`.
    pub fn evaluate(
        self,
        parameters: &ParameterCollection<u64>,
        resolver: &impl WorkPlaneResolver,
    ) -> Result<SketchPlane, WorkPlaneError> {
        match self {
            Self::Fixed(plane) => Ok(plane),
            Self::NormalToAxis(source) => normal_axis_frame(source, resolver),
            Self::LineThroughPoint(source) => line_point_frame(source, resolver),
            Self::ThroughThreePoints(points) => three_point_frame(points, resolver),
            Self::Offset { base, distance } => shifted_frame(
                base.frame(resolver)?,
                plane_offset_value(distance, parameters)?,
            ),
            Self::ParallelThroughPoint(source) => repositioned_frame(
                source.base.frame(resolver)?,
                source.point.position(resolver)?,
            ),
            _ => Err(WorkPlaneError::NonPlane(Box::new(self))),
        }
    }
}

impl AuthoredFeature<WorkPlaneDefinition> {
    /// Build an entry with definition-derived footprints; example: `AuthoredFeature::work_plane(id, "Offset", definition)`.
    pub fn work_plane(id: FeatureId, name: &str, definition: WorkPlaneDefinition) -> Self {
        Self::datum(id, name, definition)
    }
}

fn shifted_frame(plane: SketchPlane, distance: f64) -> Result<SketchPlane, WorkPlaneError> {
    let mut recipe = plane.recipe();
    let normal = plane.normal();
    recipe.origin = std::array::from_fn(|axis| recipe.origin[axis] + normal[axis] * distance);
    SketchPlane::from_recipe(recipe, plane.options()).map_err(WorkPlaneError::Frame)
}

fn plane_offset_value(
    distance: WorkPlaneDistance,
    parameters: &ParameterCollection<u64>,
) -> Result<f64, WorkPlaneError> {
    distance.resolve(parameters).map_err(plane_length_error)
}
fn plane_length_error(error: super::work_geometry::DatumLengthError) -> WorkPlaneError {
    match error {
        super::work_geometry::DatumLengthError::Parameter(cause) => {
            WorkPlaneError::Parameter(cause)
        }
        super::work_geometry::DatumLengthError::Quantity {
            parameter,
            quantity,
        } => WorkPlaneError::Offset {
            parameter,
            quantity,
        },
    }
}

impl WorkPlaneReference {
    /// Resolve the current base frame; example: `reference.frame(&evaluation)?`.
    pub fn frame(self, resolver: &impl WorkPlaneResolver) -> Result<SketchPlane, WorkPlaneError> {
        match self {
            Self::Origin(origin) => Ok(origin.plane()),
            Self::Feature(id) => resolver.plane(id),
        }
    }
}
fn repositioned_frame(
    plane: SketchPlane,
    position: [f64; 3],
) -> Result<SketchPlane, WorkPlaneError> {
    let mut recipe = plane.recipe();
    recipe.origin = position;
    SketchPlane::from_recipe(recipe, plane.options()).map_err(WorkPlaneError::Frame)
}

fn three_point_frame(
    points: [super::work_geometry::WorkPointReference; 3],
    resolver: &impl WorkPlaneResolver,
) -> Result<SketchPlane, WorkPlaneError> {
    SketchPlane::through_points(
        points[0].position(resolver)?,
        points[1].position(resolver)?,
        points[2].position(resolver)?,
        Default::default(),
    )
    .map_err(WorkPlaneError::Frame)
}

fn line_point_frame(
    source: super::work_geometry::WorkPlaneLinePoint,
    resolver: &impl WorkPlaneResolver,
) -> Result<SketchPlane, WorkPlaneError> {
    let line = source.line.line(resolver)?;
    SketchPlane::through_line_and_point(
        line.origin(),
        line.direction(),
        source.point.position(resolver)?,
        Default::default(),
    )
    .map_err(WorkPlaneError::Frame)
}

fn normal_axis_frame(
    source: super::work_geometry::WorkPlaneLinePoint,
    resolver: &impl WorkPlaneResolver,
) -> Result<SketchPlane, WorkPlaneError> {
    let normal = source.line.line(resolver)?.direction();
    let origin = source.point.position(resolver)?;
    SketchPlane::from_origin_normal(origin, normal, Default::default())
        .map_err(WorkPlaneError::Frame)
}
