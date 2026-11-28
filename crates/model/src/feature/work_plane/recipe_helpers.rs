// SPDX-License-Identifier: GPL-2.0-only
//! Helper conversions extracted from work_plane/recipe.rs to keep file lengths concise.
use super::{
    WorkPlaneDefinition, WorkPlaneDefinitionRecipe, WorkPlaneDistance, WorkPlaneError,
    WorkPlaneReference,
};
use crate::sketch::{SketchPlane, SketchPlaneOptions, SketchPlaneRecipe};

pub(super) fn restore_line_axis(
    recipe: crate::feature::work_axis::WorkAxisLineRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    crate::feature::work_axis::WorkAxisLine::from_recipe(recipe)
        .map(WorkPlaneDefinition::Axis)
        .map_err(WorkPlaneError::Axis)
}

pub(super) fn fixed_frame_recipe(plane: SketchPlane) -> WorkPlaneDefinitionRecipe {
    WorkPlaneDefinitionRecipe::Fixed {
        frame: plane.recipe(),
        options: plane.options(),
    }
}

pub(super) fn restore_frame(
    frame: SketchPlaneRecipe,
    options: SketchPlaneOptions,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    SketchPlane::from_recipe(frame, options)
        .map(WorkPlaneDefinition::Fixed)
        .map_err(WorkPlaneError::Frame)
}

pub(super) fn restore_line_angle(
    source: crate::feature::work_geometry::WorkPlaneLineAngle,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    source.angle.validate_source()?;
    Ok(WorkPlaneDefinition::AtAngleAboutAxis(source))
}

pub(super) fn restore_prism_inputs(
    recipe: crate::feature::sketch_prism::SketchPrismRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    crate::feature::sketch_prism::SketchPrismDefinition::from_recipe(&recipe)
        .map(|definition| WorkPlaneDefinition::SketchPrism(Box::new(definition)))
        .map_err(WorkPlaneError::Prism)
}

pub(super) fn offset_source_recipe(
    base: WorkPlaneReference,
    distance: WorkPlaneDistance,
) -> WorkPlaneDefinitionRecipe {
    WorkPlaneDefinitionRecipe::Offset { base, distance }
}
