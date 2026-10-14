// SPDX-License-Identifier: GPL-2.0-only
use super::sampling::{require_sampling_options, sample_definition};
use super::{SketchLineDomain, SketchSamplingOptionsV1, shape};
use oblikovati_model::sketch::{
    SketchEntityId, SketchPlacementMember, SketchPlacementRecipe, SketchReadIndex,
};
mod local_model;
/// One sampled member in recipe order; these coordinates never drive model decisions.
#[derive(Clone, Debug, PartialEq)]
pub struct PlacementPreviewCurve {
    pub member_index: usize,
    pub construction: bool,
    pub points: Vec<[f64; 2]>,
}
/// Sample the same authoring graph used by commit, in a private temporary namespace.
/// Example: `sample_placement_preview(&recipe, options)?` for a transient outline.
pub fn sample_placement_preview(
    recipe: &SketchPlacementRecipe,
    options: SketchSamplingOptionsV1,
) -> Result<Vec<PlacementPreviewCurve>, String> {
    require_sampling_options(options)?;
    let (geometry, members) = local_model::build(recipe)?;
    let index = geometry.read_index();
    recipe
        .members
        .iter()
        .zip(members)
        .enumerate()
        .map(|(member_index, (member, id))| {
            sample_member(&index, id, *member, member_index, options)
                .map_err(|cause| format!("preview member {member_index}: {cause}"))
        })
        .collect()
}
fn sample_member(
    geometry: &SketchReadIndex<'_>,
    id: SketchEntityId,
    member: SketchPlacementMember,
    member_index: usize,
    options: SketchSamplingOptionsV1,
) -> Result<PlacementPreviewCurve, String> {
    let resolved = shape::resolve(geometry, id, SketchLineDomain::Segment, options.deflection)
        .map_err(|cause| cause.to_string())?;
    Ok(PlacementPreviewCurve {
        member_index,
        construction: member.construction,
        points: sample_definition(&resolved.definition, options)?,
    })
}
