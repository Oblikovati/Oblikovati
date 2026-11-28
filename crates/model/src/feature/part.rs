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
mod combine_evaluation;
mod delete_face_evaluation;
mod finish_evaluation;

mod direct_edit_evaluation;
mod hole_evaluation;
mod mark_evaluation;
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
mod extrusion_pattern;
mod prefix_replay;
mod prefix_replay_backend;
mod prism_replacement;
mod reads;
mod resolver;
mod split_evaluation;
mod transfer;
pub use prefix_replay::BorrowedPartReplay;
pub use transfer::{CapturedPartTransfer, CompletedPartTransfer};
mod checkpoint_suffix;
mod replay_boundary;
mod replay_observer;
mod shell_evaluation;
mod sketch_body_evaluation;
mod sketch_recompute;
mod thicken_offset_evaluation;
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
    Slot(Box<super::slot::SlotDefinition>),
    Combine(Box<super::combine::CombineDefinition>),
    DirectEdit(Box<super::direct_edit::DirectEditDefinition>),
    Draft(Box<super::draft::DraftDefinition>),
    Thread(Box<super::thread::ThreadDefinition>),
    DeleteFace(Box<super::delete_face::DeleteFaceDefinition>),
    ThickenOffset(Box<super::thicken_offset::ThickenOffsetDefinition>),
    Split(Box<super::split::SplitDefinition>),
    Mark(Box<super::mark::MarkDefinition>),
    Finish(Box<super::finish::FinishDefinition>),
    MoveBody(Box<super::move_body::MoveBodyDefinition>),
    Shell(Box<super::shell::ShellDefinition>),
    Mirror(Box<super::mirror::MirrorDefinition>),
    CircularPattern(Box<super::circular_pattern::CircularPatternDefinition>),
    RectangularPattern(Box<super::rectangular_pattern::RectangularPatternDefinition>),
    SketchPrism(Box<super::sketch_prism::SketchPrismDefinition>),
    SketchCoil(Box<super::sketch_coil::SketchCoilDefinition>),
    SketchEmboss(Box<super::sketch_emboss::SketchEmbossDefinition>),
    SketchRib(Box<super::sketch_rib::SketchRibDefinition>),
    SketchDecal(Box<super::sketch_decal::SketchDecalDefinition>),
    DerivedPart(Box<super::derived_part::DerivedPartDefinition>),
    ImportedModel(Box<super::imported_model::ImportedModelDefinition>),
    Unwrap(Box<super::unwrap::UnwrapDefinition>),
    SketchLoft(Box<super::sketch_loft::SketchLoftDefinition>),
    SketchRevolve(Box<super::sketch_revolve::SketchRevolveDefinition>),
    SketchSweep(Box<super::sketch_sweep::SketchSweepDefinition>),
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
    Ucs(Box<super::ucs::UcsDefinition>),
}
impl PartFeatureDefinition {
    /// Return the stable feature vocabulary reported by diagnostics; example: `definition.kind() == "extrude"`.
    pub fn kind(&self) -> &'static str {
        match self {
            Self::SketchPrism(_) => "extrude",
            Self::SketchCoil(_) => "coil",
            Self::SketchEmboss(_) => "emboss",
            Self::SketchRib(_) => "rib",
            Self::SketchDecal(_) => "decal",
            Self::DerivedPart(_) => "derived",
            Self::ImportedModel(_) => "imported",
            Self::Unwrap(_) => "unwrap",
            Self::SketchSweep(_) => "sweep",
            Self::SketchLoft(_) => "loft",
            Self::SketchRevolve(_) => "revolve",
            Self::DeleteBody(_) => "delete-body",
            Self::Chamfer(_) => "chamfer",
            Self::Fillet(_) => "fillet",
            Self::Hole(_) => "hole",
            Self::Slot(_) => "slot",
            Self::Combine(_) => "combine",
            Self::DirectEdit(_) => "direct-edit",
            Self::Draft(_) => "draft",
            Self::Thread(_) => "thread",
            Self::DeleteFace(_) => "delete-face",
            Self::ThickenOffset(_) => "thicken-offset",
            Self::Split(_) => "split",
            Self::Mark(_) => "mark",
            Self::Finish(_) => "finish",
            Self::MoveBody(_) => "move-body",
            Self::Shell(_) => "shell",
            Self::Mirror(_) => "mirror",
            Self::CircularPattern(_) => "circular-pattern",
            Self::RectangularPattern(_) => "rectangular-pattern",
            Self::Axis(_) | Self::AxisThroughPoints(_) => "work-axis",
            Self::Point(_) | Self::PointAtReference(_) => "work-point",
            Self::Ucs(_) => "ucs",
            _ => "work-plane",
        }
    }

    /// True when this feature must carry plain plane metadata instead of datum-plane flags.
    pub fn requires_plain_plane_metadata(&self) -> bool {
        matches!(
            self,
            Self::SketchPrism(_)
                | Self::SketchSweep(_)
                | Self::SketchLoft(_)
                | Self::SketchRevolve(_)
                | Self::SketchCoil(_)
                | Self::SketchEmboss(_)
                | Self::SketchRib(_)
                | Self::SketchDecal(_)
                | Self::DerivedPart(_)
                | Self::ImportedModel(_)
                | Self::Unwrap(_)
                | Self::DeleteBody(_)
                | Self::Fillet(_)
                | Self::Chamfer(_)
                | Self::Hole(_)
                | Self::Slot(_)
                | Self::Combine(_)
                | Self::DirectEdit(_)
                | Self::Draft(_)
                | Self::Thread(_)
                | Self::DeleteFace(_)
                | Self::ThickenOffset(_)
                | Self::Split(_)
                | Self::Mark(_)
                | Self::Finish(_)
                | Self::Shell(_)
                | Self::MoveBody(_)
                | Self::Mirror(_)
                | Self::CircularPattern(_)
                | Self::RectangularPattern(_)
        )
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
