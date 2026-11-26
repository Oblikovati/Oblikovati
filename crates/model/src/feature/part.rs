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
mod blend_evaluation;
mod body_dependencies;
mod body_lineage;
mod body_resolution;
mod checkpoints;
mod cut_evaluation;
mod hole_evaluation;
mod move_body_evaluation;
pub use checkpoints::{CheckpointedPart, PartReplayCheckpoints};
mod delete;
mod error;
pub use delete::DeleteBodyDefinition;
mod evaluate;
mod evaluator;
pub use evaluate::{
    PartBlendKind, PartFeatureEvaluation, PartFeatureFailure, RecomputedPart,
    unavailable_planar_face,
};
pub use evaluator::PartFeatureEvaluator;
mod prefix_replay;
mod reads;
mod resolver;
mod transfer;
pub use prefix_replay::BorrowedPartReplay;
pub use transfer::{CapturedPartTransfer, CompletedPartTransfer};
mod checkpoint_suffix;
mod replay_boundary;
mod replay_observer;
mod shell_evaluation;
mod sketch_recompute;
pub use super::work_plane::{
    WorkPlaneProgramRecipe as PartFeatureProgramRecipe,
    WorkPlaneProgramRecipeError as PartFeatureProgramRecipeError,
};
pub use error::PartFeatureProgramError;
/// One owned datum or body input in the common ordered feature program.
#[derive(Clone, Debug, PartialEq)]
pub enum PartFeatureDefinition {
    DeleteBody(DeleteBodyDefinition),
    Chamfer(Box<super::chamfer::ChamferDefinition>),
    Fillet(Box<super::fillet::FilletDefinition>),
    Hole(Box<super::hole::HoleDefinition>),
    MoveBody(Box<super::move_body::MoveBodyDefinition>),
    Shell(Box<super::shell::ShellDefinition>),
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
impl PartFeatureDefinition {
    /// Return the stable feature vocabulary reported by diagnostics; example: `definition.kind() == "extrude"`.
    pub fn kind(&self) -> &'static str {
        match self {
            Self::SketchPrism(_) => "extrude",
            Self::DeleteBody(_) => "delete-body",
            Self::Chamfer(_) => "chamfer",
            Self::Fillet(_) => "fillet",
            Self::Hole(_) => "hole",
            Self::MoveBody(_) => "move-body",
            Self::Shell(_) => "shell",
            Self::Axis(_) | Self::AxisThroughPoints(_) => "work-axis",
            Self::Point(_) | Self::PointAtReference(_) => "work-point",
            _ => "work-plane",
        }
    }
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
