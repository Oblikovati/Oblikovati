// SPDX-License-Identifier: GPL-2.0-only
//! Analytic picking in definition coordinates, projected back through the instance placement.
use crate::{HeadSketchEdit, HeadWindow};
use oblikovati_martensite::planar_curve::{PlanarPickOptionsV1, sketch::SketchPickQuery};
use oblikovati_model::{document_sketch::SketchBlockInstance, history::DocumentSnapshot};
use oblikovati_session::selection::SketchPickTarget;

pub(super) fn nearest_block(
    snapshot: &DocumentSnapshot,
    edit: HeadSketchEdit,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<(f64, SketchPickTarget)>, String> {
    let Some(cursor) = window.sketch_point(pixel, edit.plane)? else {
        return Ok(None);
    };
    let sketch = snapshot
        .sketches
        .get(edit.sketch)
        .map_err(|cause| cause.to_string())?;
    if !sketch.visible() {
        return Ok(None);
    }
    let mut candidates = Vec::new();
    for instance in sketch.block_instances() {
        if let Some(distance) = block_distance(snapshot, instance, cursor, edit, pixel, window)? {
            candidates.push((distance, instance.id()));
        }
    }
    Ok(candidates
        .into_iter()
        .min_by(|a, b| a.0.total_cmp(&b.0).then(a.1.cmp(&b.1)))
        .map(|(distance, id)| (distance, SketchPickTarget::BlockInstance(id))))
}

fn block_distance(
    snapshot: &DocumentSnapshot,
    instance: &SketchBlockInstance,
    cursor: [f64; 2],
    edit: HeadSketchEdit,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<f64>, String> {
    let definition = snapshot
        .sketches
        .block_definitions()
        .iter()
        .find(|item| item.name() == instance.definition())
        .ok_or_else(|| {
            format!(
                "block instance {:?} definition {:?}: expected a current definition",
                instance.id(),
                instance.definition()
            )
        })?;
    let local = instance
        .to_definition(cursor)
        .map_err(|cause| cause.to_string())?;
    let geometry = definition.geometry();
    let query = SketchPickQuery::new(geometry, local, PlanarPickOptionsV1::default())
        .map_err(|cause| cause.to_string())?;
    let mut distance: Option<f64> = None;
    for id in geometry.entity_ids() {
        let witness = query.candidate(*id).map_err(|cause| cause.to_string())?;
        let point = instance
            .to_sketch(witness.position)
            .map_err(|cause| cause.to_string())?;
        let projected = window.sketch_pixel(edit.plane, point)?;
        let candidate = super::sketch_picking::screen_distance(projected, pixel);
        if candidate <= 12. && distance.is_none_or(|current| candidate < current) {
            distance = Some(candidate);
        }
    }
    Ok(distance)
}
pub(super) fn nearest_target(
    loose: Option<(f64, SketchPickTarget)>,
    block: Option<(f64, SketchPickTarget)>,
) -> Option<SketchPickTarget> {
    match (loose, block) {
        (Some(a), Some(b)) if a.0 <= b.0 => Some(a.1),
        (Some(_), Some(b)) | (None, Some(b)) => Some(b.1),
        (Some(a), None) => Some(a.1),
        (None, None) => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::sketch::SketchEntityId;
    #[test]
    fn nearest_candidate_uses_distance_and_keeps_loose_geometry_on_a_tie() {
        let loose = SketchPickTarget::Entity(SketchEntityId(2));
        let block = SketchPickTarget::BlockInstance(SketchEntityId(3));
        assert_eq!(nearest_target(None, None), None);
        assert_eq!(
            nearest_target(Some((2., loose)), Some((3., block))),
            Some(loose)
        );
        assert_eq!(
            nearest_target(Some((3., loose)), Some((2., block))),
            Some(block)
        );
        assert_eq!(
            nearest_target(Some((2., loose)), Some((2., block))),
            Some(loose)
        );
    }
}
