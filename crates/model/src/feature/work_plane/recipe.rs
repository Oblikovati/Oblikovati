// SPDX-License-Identifier: GPL-2.0-only

use super::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneError, WorkPlaneReference};
use crate::sketch::{SketchPlaneOptions, SketchPlaneRecipe};
#[path = "recipe_helpers.rs"]
mod recipe_helpers;
use recipe_helpers::{
    fixed_frame_recipe, offset_source_recipe, restore_frame, restore_line_angle, restore_line_axis,
    restore_prism_inputs,
};

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
    ReplaceFace(crate::feature::replace_face::ReplaceFaceRecipe),
    ThickenOffset(crate::feature::thicken_offset::ThickenOffsetRecipe),
    Stitch(crate::feature::stitch::StitchRecipe),
    Split(crate::feature::split::SplitRecipe),
    SurfaceTrim(crate::feature::surface_trim::SurfaceTrimRecipe),
    Mark(crate::feature::mark::MarkRecipe),
    Finish(crate::feature::finish::FinishRecipe),
    RuledSurface(crate::feature::ruled_surface::RuledSurfaceRecipe),
    Ucs(crate::feature::ucs::UcsRecipe),
    ExtendSurface(crate::feature::extend_surface::ExtendSurfaceRecipe),
    BoundaryPatch(crate::feature::boundary_patch::BoundaryPatchRecipe),
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
    WorkPoint(crate::feature::work_point::WorkPointRecipe),
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
            Self::ReplaceFace(definition) => Source::ReplaceFace(definition.recipe()),
            Self::ThickenOffset(definition) => Source::ThickenOffset(definition.recipe()),
            Self::Stitch(definition) => Source::Stitch(definition.recipe()),
            Self::Split(definition) => Source::Split(definition.recipe()),
            Self::SurfaceTrim(definition) => Source::SurfaceTrim(definition.recipe()),
            Self::Mark(definition) => Source::Mark(definition.recipe()),
            Self::Finish(definition) => Source::Finish(definition.recipe()),
            Self::RuledSurface(definition) => Source::RuledSurface(definition.recipe()),
            Self::Ucs(definition) => Source::Ucs(definition.recipe()),
            Self::ExtendSurface(definition) => Source::ExtendSurface(definition.recipe()),
            Self::BoundaryPatch(definition) => Source::BoundaryPatch(definition.recipe()),
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
            Self::WorkPoint(definition) => Source::WorkPoint(definition.recipe()),
            Self::AxisThroughPoints(points) => Source::AxisThroughPoints(*points),
            Self::Fixed(plane) => fixed_frame_recipe(*plane),
            Self::Offset { base, distance } => offset_source_recipe(base.clone(), *distance),
        }
    }

    /// Restore and validate a supported recipe; example: `WorkPlaneDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: WorkPlaneRecipe) -> Result<Self, WorkPlaneError> {
        let expected = recipe.definition.source_version();
        if recipe.version != expected {
            return Err(WorkPlaneError::RecipeVersion(recipe.version));
        }
        restore_definition(recipe.definition)
    }
}

fn restore_definition(
    definition: WorkPlaneDefinitionRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    use WorkPlaneDefinitionRecipe::*;
    match definition {
        Mirror(recipe) => crate::feature::mirror::MirrorDefinition::new(recipe)
            .map(|definition| WorkPlaneDefinition::Mirror(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        CircularPattern(recipe) => {
            crate::feature::circular_pattern::CircularPatternDefinition::new(recipe)
                .map(|definition| WorkPlaneDefinition::CircularPattern(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        RectangularPattern(recipe) => {
            crate::feature::rectangular_pattern::RectangularPatternDefinition::new(recipe)
                .map(|definition| WorkPlaneDefinition::RectangularPattern(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Shell(recipe) => crate::feature::shell::ShellDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::Shell(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        MoveBody(recipe) => crate::feature::move_body::MoveBodyDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::MoveBody(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Hole(recipe) => crate::feature::hole::HoleDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Hole(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Slot(recipe) => crate::feature::slot::SlotDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Slot(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Combine(recipe) => crate::feature::combine::CombineDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Combine(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        DirectEdit(recipe) => {
            crate::feature::direct_edit::DirectEditDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::DirectEdit(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Draft(recipe) => crate::feature::draft::DraftDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Draft(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Thread(recipe) => crate::feature::thread::ThreadDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Thread(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        DeleteFace(recipe) => {
            crate::feature::delete_face::DeleteFaceDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::DeleteFace(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        ReplaceFace(recipe) => {
            crate::feature::replace_face::ReplaceFaceDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::ReplaceFace(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        ThickenOffset(recipe) => {
            crate::feature::thicken_offset::ThickenOffsetDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::ThickenOffset(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Stitch(recipe) => crate::feature::stitch::StitchDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Stitch(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Split(recipe) => crate::feature::split::SplitDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Split(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        SurfaceTrim(recipe) => {
            crate::feature::surface_trim::SurfaceTrimDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::SurfaceTrim(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Mark(recipe) => crate::feature::mark::MarkDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Mark(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Finish(recipe) => crate::feature::finish::FinishDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Finish(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        RuledSurface(recipe) => {
            crate::feature::ruled_surface::RuledSurfaceDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::RuledSurface(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Ucs(recipe) => crate::feature::ucs::UcsDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Ucs(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        ExtendSurface(recipe) => crate::feature::extend_surface::restore_definition(recipe),
        BoundaryPatch(recipe) => {
            crate::feature::boundary_patch::BoundaryPatchDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::BoundaryPatch(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Chamfer(recipe) => crate::feature::chamfer::ChamferDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::Chamfer(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Fillet(recipe) => crate::feature::fillet::FilletDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::Fillet(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        DeleteBody(key) => crate::feature::part::DeleteBodyDefinition::new(key)
            .map(WorkPlaneDefinition::DeleteBody)
            .map_err(WorkPlaneError::BodyReference),
        SketchLoft(recipe) => {
            crate::feature::sketch_loft::SketchLoftDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::SketchLoft(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        SketchRevolve(recipe) => {
            crate::feature::sketch_revolve::SketchRevolveDefinition::from_recipe(&recipe)
                .map(|definition| WorkPlaneDefinition::SketchRevolve(Box::new(definition)))
                .map_err(|cause| WorkPlaneError::BodyReference(cause.to_string()))
        }
        SketchCoil(recipe) => {
            crate::feature::sketch_coil::SketchCoilDefinition::from_recipe(&recipe)
                .map(|definition| WorkPlaneDefinition::SketchCoil(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        SketchEmboss(recipe) => {
            crate::feature::sketch_emboss::SketchEmbossDefinition::from_recipe(&recipe)
                .map(|definition| WorkPlaneDefinition::SketchEmboss(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        SketchRib(recipe) => crate::feature::sketch_rib::SketchRibDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::SketchRib(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        SketchDecal(recipe) => {
            crate::feature::sketch_decal::SketchDecalDefinition::from_recipe(&recipe)
                .map(|definition| WorkPlaneDefinition::SketchDecal(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        DerivedPart(recipe) => {
            crate::feature::derived_part::DerivedPartDefinition::from_recipe(&recipe)
                .map(|definition| WorkPlaneDefinition::DerivedPart(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        ImportedModel(recipe) => {
            crate::feature::imported_model::ImportedModelDefinition::from_recipe(&recipe)
                .map(|definition| WorkPlaneDefinition::ImportedModel(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Unwrap(recipe) => crate::feature::unwrap::UnwrapDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::Unwrap(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        SketchSweep(recipe) => {
            crate::feature::sketch_sweep::SketchSweepDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::SketchSweep(Box::new(definition)))
                .map_err(|cause| WorkPlaneError::Sweep(Box::new(cause)))
        }
        SketchPrism(recipe) => restore_prism_inputs(recipe),
        AtAngleAboutAxis(source) => restore_line_angle(source),
        ThroughTwoAxes(axes) => Ok(WorkPlaneDefinition::ThroughTwoAxes(axes)),
        NormalToAxis(source) => Ok(WorkPlaneDefinition::NormalToAxis(source)),
        LineThroughPoint(source) => Ok(WorkPlaneDefinition::LineThroughPoint(source)),
        ThroughThreePoints(points) => Ok(WorkPlaneDefinition::ThroughThreePoints(points)),
        ParallelThroughPoint(source) => Ok(WorkPlaneDefinition::ParallelThroughPoint(source)),
        WorkPlaneExtended(recipe) => {
            super::extended::WorkPlaneExtendedDefinition::from_recipe(*recipe)
                .map(|definition| WorkPlaneDefinition::WorkPlaneExtended(Box::new(definition)))
        }
        Axis(axis) => restore_line_axis(axis),
        Fixed { frame, options } => restore_frame(frame, options),
        Offset { base, distance } => validate_offset(base, distance),
        Point(coordinates) => restore_point(coordinates),
        PointAtReference(reference) => Ok(WorkPlaneDefinition::PointAtReference(reference)),
        WorkPoint(recipe) => crate::feature::work_point::WorkPointDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::WorkPoint(Box::new(definition))),
        AxisThroughPoints(points) => Ok(WorkPlaneDefinition::AxisThroughPoints(points)),
    }
}

fn validate_offset(
    base: WorkPlaneReference,
    distance: WorkPlaneDistance,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    distance
        .validate_source()
        .map_err(super::plane_length_error)?;
    Ok(WorkPlaneDefinition::Offset { base, distance })
}
fn restore_point(
    coordinates: [crate::feature::work_geometry::DatumLength; 3],
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    crate::feature::work_geometry::validate_position(coordinates)?;
    Ok(WorkPlaneDefinition::Point(coordinates))
}
impl WorkPlaneDefinitionRecipe {
    fn base_reference(&self) -> Option<&WorkPlaneReference> {
        match self {
            Self::Offset { base, .. } => Some(base),
            Self::ParallelThroughPoint(source) => Some(&source.base),
            Self::AtAngleAboutAxis(source) => Some(&source.base),
            _ => None,
        }
    }
    fn has_go_face_reference(&self) -> bool {
        matches!(
            self.base_reference(),
            Some(WorkPlaneReference::GoPrismFace(_))
        )
    }
    fn has_face_reference(&self) -> bool {
        matches!(self.base_reference(), Some(WorkPlaneReference::Face(_)))
    }
}

#[path = "recipe_versions.rs"]
mod recipe_versions;
