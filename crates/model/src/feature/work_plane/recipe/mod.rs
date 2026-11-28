// SPDX-License-Identifier: GPL-2.0-only

mod restore;
mod versions;

use super::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneError, WorkPlaneReference};
use crate::sketch::{SketchPlane, SketchPlaneOptions, SketchPlaneRecipe};

/// Versioned authored construction data; no evaluated caches are stored.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPlaneRecipe {
    pub version: u32,
    pub definition: WorkPlaneDefinitionRecipe,
}

/// One serialized definition selected by variant; scalar closures are not saved.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub enum WorkPlaneDefinitionRecipe {
    DeleteBody(Vec<u8>),
    Chamfer(crate::feature::chamfer::ChamferRecipe),
    Fillet(crate::feature::fillet::FilletRecipe),
    Hole(crate::feature::hole::HoleRecipe),
    Slot(crate::feature::slot::SlotRecipe),
    Combine(crate::feature::combine::CombineRecipe),
    DirectEdit(crate::feature::direct_edit::DirectEditRecipe),
    Draft(crate::feature::draft::DraftRecipe),
    Thread(crate::feature::thread::ThreadRecipe),
    DeleteFace(crate::feature::delete_face::DeleteFaceRecipe),
    ThickenOffset(crate::feature::thicken_offset::ThickenOffsetRecipe),
    Split(crate::feature::split::SplitRecipe),
    Mark(crate::feature::mark::MarkRecipe),
    MoveBody(crate::feature::move_body::MoveBodyRecipe),
    Shell(crate::feature::shell::ShellRecipe),
    Mirror(crate::feature::mirror::MirrorRecipe),
    CircularPattern(crate::feature::circular_pattern::CircularPatternRecipe),
    RectangularPattern(crate::feature::rectangular_pattern::RectangularPatternRecipe),
    SketchPrism(crate::feature::sketch_prism::SketchPrismRecipe),
    SketchLoft(crate::feature::sketch_loft::SketchLoftRecipe),
    SketchRevolve(crate::feature::sketch_revolve::SketchRevolveRecipe),
    SketchCoil(crate::feature::sketch_coil::SketchCoilRecipe),
    SketchEmboss(crate::feature::sketch_emboss::SketchEmbossRecipe),
    SketchRib(crate::feature::sketch_rib::SketchRibRecipe),
    SketchDecal(crate::feature::sketch_decal::SketchDecalRecipe),
    DerivedPart(crate::feature::derived_part::DerivedPartRecipe),
    ImportedModel(crate::feature::imported_model::ImportedModelRecipe),
    Unwrap(crate::feature::unwrap::UnwrapRecipe),
    SketchSweep(crate::feature::sketch_sweep::SketchSweepRecipe),
    AtAngleAboutAxis(crate::feature::work_geometry::WorkPlaneLineAngle),
    ThroughTwoAxes([crate::feature::work_axis::WorkAxisReference; 2]),
    NormalToAxis(crate::feature::work_geometry::WorkPlaneLinePoint),
    LineThroughPoint(crate::feature::work_geometry::WorkPlaneLinePoint),
    ThroughThreePoints([crate::feature::work_geometry::WorkPointReference; 3]),
    ParallelThroughPoint(crate::feature::work_geometry::WorkPlaneThroughPoint),
    WorkPlaneExtended(Box<super::extended::WorkPlaneExtendedRecipe>),
    Axis(crate::feature::work_axis::WorkAxisLineRecipe),
    Point([crate::feature::work_geometry::DatumLength; 3]),
    AxisThroughPoints(crate::feature::work_geometry::WorkAxisPoints),
    PointAtReference(crate::feature::work_geometry::WorkPointReference),
    Fixed {
        frame: SketchPlaneRecipe,
        options: SketchPlaneOptions,
    },
    Offset {
        base: WorkPlaneReference,
        distance: WorkPlaneDistance,
    },
}

impl WorkPlaneDefinition {
    /// Capture authored source without evaluated output; example: `definition.recipe()`.
    pub fn recipe(&self) -> WorkPlaneRecipe {
        let definition = self.source_recipe();
        WorkPlaneRecipe {
            version: definition.source_version(),
            definition,
        }
    }

    fn source_recipe(&self) -> WorkPlaneDefinitionRecipe {
        use WorkPlaneDefinitionRecipe as Source;
        match self {
            Self::DeleteBody(definition) => Source::DeleteBody(definition.key().to_vec()),
            Self::Chamfer(definition) => Source::Chamfer(definition.recipe()),
            Self::Fillet(definition) => Source::Fillet(definition.recipe()),
            Self::Hole(definition) => Source::Hole(definition.recipe()),
            Self::Slot(definition) => Source::Slot(definition.recipe()),
            Self::Combine(definition) => Source::Combine(definition.recipe()),
            Self::DirectEdit(definition) => Source::DirectEdit(definition.recipe()),
            Self::Draft(definition) => Source::Draft(definition.recipe()),
            Self::Thread(definition) => Source::Thread(definition.recipe()),
            Self::DeleteFace(definition) => Source::DeleteFace(definition.recipe()),
            Self::ThickenOffset(definition) => Source::ThickenOffset(definition.recipe()),
            Self::Split(definition) => Source::Split(definition.recipe()),
            Self::Mark(definition) => Source::Mark(definition.recipe()),
            Self::MoveBody(definition) => Source::MoveBody(definition.recipe()),
            Self::Shell(definition) => Source::Shell(definition.recipe()),
            Self::Mirror(mirror) => Source::Mirror(mirror.recipe().clone()),
            Self::CircularPattern(pattern) => Source::CircularPattern(pattern.recipe().clone()),
            Self::RectangularPattern(pattern) => {
                Source::RectangularPattern(pattern.recipe().clone())
            }
            Self::SketchLoft(definition) => Source::SketchLoft(definition.recipe().clone()),
            Self::SketchRevolve(definition) => Source::SketchRevolve(definition.recipe()),
            Self::SketchCoil(definition) => Source::SketchCoil(definition.recipe()),
            Self::SketchEmboss(definition) => Source::SketchEmboss(definition.recipe()),
            Self::SketchRib(definition) => Source::SketchRib(definition.recipe()),
            Self::SketchDecal(definition) => Source::SketchDecal(definition.recipe()),
            Self::DerivedPart(definition) => Source::DerivedPart(definition.recipe()),
            Self::ImportedModel(definition) => Source::ImportedModel(definition.recipe()),
            Self::Unwrap(definition) => Source::Unwrap(definition.recipe()),
            Self::SketchSweep(definition) => Source::SketchSweep(definition.recipe().clone()),
            Self::SketchPrism(definition) => Source::SketchPrism(definition.recipe()),
            Self::AtAngleAboutAxis(source) => Source::AtAngleAboutAxis(source.clone()),
            Self::ThroughTwoAxes(axes) => Source::ThroughTwoAxes(*axes),
            Self::NormalToAxis(source) => Source::NormalToAxis(*source),
            Self::LineThroughPoint(source) => Source::LineThroughPoint(*source),
            Self::ThroughThreePoints(points) => Source::ThroughThreePoints(*points),
            Self::ParallelThroughPoint(source) => Source::ParallelThroughPoint(source.clone()),
            Self::WorkPlaneExtended(definition) => {
                Source::WorkPlaneExtended(Box::new(definition.recipe()))
            }
            Self::Axis(axis) => Source::Axis(axis.recipe()),
            Self::Point(coordinates) => Source::Point(*coordinates),
            Self::PointAtReference(reference) => Source::PointAtReference(*reference),
            Self::AxisThroughPoints(points) => Source::AxisThroughPoints(*points),
            Self::Fixed(plane) => fixed_frame_recipe(*plane),
            Self::Offset { base, distance } => Source::Offset {
                base: base.clone(),
                distance: *distance,
            },
        }
    }

    /// Restore and validate a supported recipe; example: `WorkPlaneDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: WorkPlaneRecipe) -> Result<Self, WorkPlaneError> {
        let expected = recipe.definition.source_version();
        if recipe.version != expected {
            return Err(WorkPlaneError::RecipeVersion(recipe.version));
        }
        restore::restore_definition(recipe.definition)
    }
}

fn fixed_frame_recipe(plane: SketchPlane) -> WorkPlaneDefinitionRecipe {
    WorkPlaneDefinitionRecipe::Fixed {
        frame: plane.recipe(),
        options: plane.options(),
    }
}
