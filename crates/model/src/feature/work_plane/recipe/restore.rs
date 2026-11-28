// SPDX-License-Identifier: GPL-2.0-only

use super::WorkPlaneDefinitionRecipe;
use crate::feature::work_plane::{
    WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneError, WorkPlaneReference,
    extended::WorkPlaneExtendedDefinition,
};
use crate::sketch::{SketchPlane, SketchPlaneOptions, SketchPlaneRecipe};

pub(super) fn restore_definition(
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
        ThickenOffset(recipe) => {
            crate::feature::thicken_offset::ThickenOffsetDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::ThickenOffset(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Split(recipe) => crate::feature::split::SplitDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Split(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Mark(recipe) => crate::feature::mark::MarkDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Mark(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
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
        WorkPlaneExtended(recipe) => WorkPlaneExtendedDefinition::from_recipe(*recipe)
            .map(|definition| WorkPlaneDefinition::WorkPlaneExtended(Box::new(definition))),
        Axis(axis) => restore_line_axis(axis),
        Fixed { frame, options } => restore_frame(frame, options),
        Offset { base, distance } => validate_offset(base, distance),
        Point(coordinates) => restore_point(coordinates),
        PointAtReference(reference) => Ok(WorkPlaneDefinition::PointAtReference(reference)),
        AxisThroughPoints(points) => Ok(WorkPlaneDefinition::AxisThroughPoints(points)),
    }
}

fn validate_offset(
    base: WorkPlaneReference,
    distance: WorkPlaneDistance,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    distance
        .validate_source()
        .map_err(crate::feature::work_plane::plane_length_error)?;
    Ok(WorkPlaneDefinition::Offset { base, distance })
}

fn restore_point(
    coordinates: [crate::feature::work_geometry::DatumLength; 3],
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    crate::feature::work_geometry::validate_position(coordinates)?;
    Ok(WorkPlaneDefinition::Point(coordinates))
}

fn restore_line_axis(
    recipe: crate::feature::work_axis::WorkAxisLineRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    crate::feature::work_axis::WorkAxisLine::from_recipe(recipe)
        .map(WorkPlaneDefinition::Axis)
        .map_err(WorkPlaneError::Axis)
}

fn restore_frame(
    frame: SketchPlaneRecipe,
    options: SketchPlaneOptions,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    SketchPlane::from_recipe(frame, options)
        .map(WorkPlaneDefinition::Fixed)
        .map_err(WorkPlaneError::Frame)
}

fn restore_line_angle(
    source: crate::feature::work_geometry::WorkPlaneLineAngle,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    source.angle.validate_source()?;
    Ok(WorkPlaneDefinition::AtAngleAboutAxis(source))
}

fn restore_prism_inputs(
    recipe: crate::feature::sketch_prism::SketchPrismRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    crate::feature::sketch_prism::SketchPrismDefinition::from_recipe(&recipe)
        .map(|definition| WorkPlaneDefinition::SketchPrism(Box::new(definition)))
        .map_err(WorkPlaneError::Prism)
}
