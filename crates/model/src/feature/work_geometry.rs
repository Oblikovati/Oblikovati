// SPDX-License-Identifier: GPL-2.0-only
//! One ordered datum program; kind-specific views do not own a second graph or ID allocator.
mod error;
mod evaluate;
pub use super::work_plane::{
    WorkPlaneProgramRecipe as WorkGeometryProgramRecipe,
    WorkPlaneProgramRecipeError as WorkGeometryProgramRecipeError,
};
use super::{
    AuthoredFeature, FeatureId, FeatureReplayError, FeatureSequence,
    work_axis::WorkAxisLine,
    work_plane::{WorkPlaneDistance, WorkPlaneReference},
};
use crate::{parameter::ParameterCollection, sketch::SketchPlane};
pub use error::WorkGeometryError;
pub use evaluate::WorkGeometryEvaluation;

/// One authored datum selected by kind inside the common feature program.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum WorkGeometryDefinition {
    Fixed(SketchPlane),
    Offset {
        base: WorkPlaneReference,
        distance: WorkPlaneDistance,
    },
    Axis(WorkAxisLine),
}
impl AuthoredFeature<WorkGeometryDefinition> {
    /// Add a checked line axis to the same permanent namespace; example: `AuthoredFeature::work_axis(id, "Spindle", axis)`.
    pub fn work_axis(id: FeatureId, name: &str, axis: WorkAxisLine) -> Self {
        Self::work_plane(id, name, WorkGeometryDefinition::Axis(axis))
    }
}
impl FeatureSequence<WorkGeometryDefinition> {
    /// Store one mixed program with versioned datum kinds; example: `sequence.work_geometry_recipe()?`.
    pub fn work_geometry_recipe(
        &self,
    ) -> Result<WorkGeometryProgramRecipe, WorkGeometryProgramRecipeError> {
        self.work_plane_recipe()
    }
    /// Restore mixed datums before owner adoption; example: `FeatureSequence::from_work_geometry_recipe(&recipe)?`.
    pub fn from_work_geometry_recipe(
        recipe: &WorkGeometryProgramRecipe,
    ) -> Result<Self, WorkGeometryProgramRecipeError> {
        Self::from_work_plane_recipe(recipe)
    }
    /// Compare exact ordered datum sources and metadata; example: `baseline.same_work_geometry_recipe(&candidate)`.
    pub fn same_work_geometry_recipe(&self, other: &Self) -> bool {
        self.same_work_plane_recipe(other)
    }
    /// Query current planes through the single mixed evaluator; example: `sequence.evaluate_work_planes(&parameters)?`.
    pub fn evaluate_work_planes(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<WorkGeometryEvaluation, FeatureReplayError> {
        self.evaluate_work_geometry(parameters)
    }
}
impl crate::document_sketch::SketchAxisResolver for WorkGeometryEvaluation {
    fn axis_direction(&self, feature: FeatureId) -> Result<[f64; 3], crate::sketch::SketchError> {
        self.axis(feature)
            .map(WorkAxisLine::direction)
            .map_err(|cause| crate::sketch::SketchError::Definition(cause.to_string()))
    }
}
