// SPDX-License-Identifier: GPL-2.0-only
use super::{
    work_axis::WorkAxisLine,
    work_geometry::{
        DatumLength, WorkAxisPoints, WorkPlaneLineAngle, WorkPlaneLinePoint, WorkPlaneThroughPoint,
        WorkPointReference,
    },
    work_plane::{WorkPlaneDistance, WorkPlaneReference},
};
use crate::sketch::SketchPlane;
mod delete;
mod error;
pub use delete::DeleteBodyDefinition;
mod evaluate;
pub use evaluate::{PartFeatureEvaluation, PartFeatureFailure, SketchPrismEvaluator};
mod reads;
pub use super::work_plane::{
    WorkPlaneProgramRecipe as PartFeatureProgramRecipe,
    WorkPlaneProgramRecipeError as PartFeatureProgramRecipeError,
};
pub use error::PartFeatureProgramError;
/// One owned datum or body input in the common ordered feature program.
#[derive(Clone, Debug, PartialEq)]
pub enum PartFeatureDefinition {
    DeleteBody(DeleteBodyDefinition),
    SketchPrism(Box<super::sketch_prism::SketchPrismDefinition>),
    Fixed(SketchPlane),
    Offset {
        base: WorkPlaneReference,
        distance: WorkPlaneDistance,
    },
    ParallelThroughPoint(WorkPlaneThroughPoint),
    ThroughThreePoints([WorkPointReference; 3]),
    LineThroughPoint(WorkPlaneLinePoint),
    NormalToAxis(WorkPlaneLinePoint),
    AtAngleAboutAxis(WorkPlaneLineAngle),
    ThroughTwoAxes([super::work_axis::WorkAxisReference; 2]),
    Axis(WorkAxisLine),
    Point([DatumLength; 3]),
    PointAtReference(WorkPointReference),
    AxisThroughPoints(WorkAxisPoints),
}
impl super::AuthoredFeature<PartFeatureDefinition> {
    /// Attach prism sources to the common program with current document dependencies.
    /// Example: `AuthoredFeature::sketch_prism(id, "Extrude", definition, &snapshot)?`.
    pub fn sketch_prism(
        id: super::FeatureId,
        name: &str,
        definition: super::sketch_prism::SketchPrismDefinition,
        snapshot: &crate::history::DocumentSnapshot,
    ) -> Result<Self, super::sketch_prism::SketchPrismError> {
        let reads = definition.reads(snapshot)?;
        Ok(Self::new(
            id,
            name,
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        ))
    }
}
impl super::FeatureSequence<PartFeatureDefinition> {
    /// Store all authored kinds through the existing versioned program serializer; example: `program.part_recipe()?`.
    pub fn part_recipe(&self) -> Result<PartFeatureProgramRecipe, PartFeatureProgramRecipeError> {
        self.work_plane_recipe()
    }
    /// Restore one common graph and ordered namespace; example: `FeatureSequence::from_part_recipe(&recipe)?`.
    pub fn from_part_recipe(
        recipe: &PartFeatureProgramRecipe,
    ) -> Result<Self, PartFeatureProgramRecipeError> {
        Self::from_work_plane_recipe(recipe)
    }
}
