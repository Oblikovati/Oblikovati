// SPDX-License-Identifier: GPL-2.0-only
use super::sampling::require_sampling_options;
use super::{SketchDisplayPrimitive, SketchSamplingOptionsV1, presentation::display_primitive};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId, SketchPlacementRecipe, SketchReadIndex};
mod local_model;
/// One display member in recipe order; these coordinates never drive model decisions.
#[derive(Clone, Debug, PartialEq)]
pub struct PlacementPreviewMember {
    pub member_index: usize,
    pub construction: bool,
    pub primitive: SketchDisplayPrimitive,
}
/// Sample the same authoring graph used by commit, in a private temporary namespace.
/// Example: `sample_placement_preview(&recipe, options)?` for a transient outline.
pub fn sample_placement_preview(
    recipe: &SketchPlacementRecipe,
    options: SketchSamplingOptionsV1,
) -> Result<Vec<PlacementPreviewMember>, String> {
    sample_placement_preview_with_policy(recipe, options, Default::default())
}
/// Sample a solved placement using its document's numerical policy.
/// Example: `sample_placement_preview_with_policy(&recipe,sampling,solve_options)?`.
pub fn sample_placement_preview_with_policy(
    recipe: &SketchPlacementRecipe,
    options: SketchSamplingOptionsV1,
    solve_options: oblikovati_solve::nonlinear::SolveOptions,
) -> Result<Vec<PlacementPreviewMember>, String> {
    require_sampling_options(options)?;
    let (geometry, members) = local_model::build(recipe, solve_options)?;
    sample_sketch_preview_members(&geometry, &members, options)
}
/// Sample a verified private-copy result without solving it again or exposing entity IDs to picking.
/// Example: `sample_sketch_preview_members(&geometry, &members, sampling)?` after document preview.
pub fn sample_sketch_preview_members(
    geometry: &Sketch2D,
    members: &[SketchEntityId],
    options: SketchSamplingOptionsV1,
) -> Result<Vec<PlacementPreviewMember>, String> {
    require_sampling_options(options)?;
    let index = geometry.read_index();
    members
        .iter()
        .enumerate()
        .map(|(member_index, id)| {
            sample_member(&index, *id, member_index, options)
                .map_err(|cause| format!("preview member {member_index}: {cause}"))
        })
        .collect()
}

fn sample_member(
    geometry: &SketchReadIndex<'_>,
    id: SketchEntityId,
    member_index: usize,
    options: SketchSamplingOptionsV1,
) -> Result<PlacementPreviewMember, String> {
    Ok(PlacementPreviewMember {
        member_index,
        construction: geometry
            .is_construction(id)
            .map_err(|cause| cause.to_string())?,
        primitive: display_primitive(geometry, id, options)?,
    })
}
