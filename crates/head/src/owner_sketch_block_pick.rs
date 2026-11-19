// SPDX-License-Identifier: GPL-2.0-only
//! Analytic picking in definition coordinates, projected back through the instance placement.
use crate::{HeadSketchEdit, HeadWindow};
use oblikovati_martensite::planar_curve::{PlanarPickOptionsV1, sketch::SketchPickQuery};
use oblikovati_model::{
    document_sketch::SketchBlockInstance, history::DocumentSnapshot, sketch::SketchEntityId,
};
use oblikovati_session::selection::SketchPickTarget;

pub(super) fn nearest_block(
    snapshot: &DocumentSnapshot,
    edit: HeadSketchEdit,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<(f64, SketchPickTarget)>, String> {
    Ok(nearest_block_member(snapshot, edit, pixel, window, false)?
        .map(|(distance, instance, _)| (distance, SketchPickTarget::BlockInstance(instance))))
}

pub(super) fn nearest_block_member(
    snapshot: &DocumentSnapshot,
    edit: HeadSketchEdit,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
    curves_only: bool,
) -> Result<Option<(f64, SketchEntityId, SketchEntityId)>, String> {
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
        if let Some((distance, member)) =
            block_distance(snapshot, instance, cursor, edit, pixel, window, curves_only)?
        {
            candidates.push((distance, instance.id(), member));
        }
    }
    Ok(candidates
        .into_iter()
        .min_by(|a, b| a.0.total_cmp(&b.0).then(a.1.cmp(&b.1)).then(a.2.cmp(&b.2))))
}

fn block_distance(
    snapshot: &DocumentSnapshot,
    instance: &SketchBlockInstance,
    cursor: [f64; 2],
    edit: HeadSketchEdit,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
    curves_only: bool,
) -> Result<Option<(f64, SketchEntityId)>, String> {
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
    let points: std::collections::BTreeSet<_> =
        geometry.points().iter().map(|point| point.id).collect();
    let mut witnesses = Vec::new();
    for id in geometry.entity_ids() {
        if curves_only && points.contains(id) {
            continue;
        }
        let witness = query.candidate(*id).map_err(|cause| cause.to_string())?;
        let point = instance
            .to_sketch(witness.position)
            .map_err(|cause| cause.to_string())?;
        witnesses.push((*id, point));
    }
    let positions: Vec<_> = witnesses.iter().map(|(_, point)| *point).collect();
    let pixels = window.sketch_pixels(edit.plane, &positions)?;
    if pixels.len() != witnesses.len() {
        return Err(format!(
            "block projected {} positions: expected {}",
            pixels.len(),
            witnesses.len()
        ));
    }
    let mut distance: Option<(f64, SketchEntityId)> = None;
    for ((id, _), projected) in witnesses.into_iter().zip(pixels) {
        let candidate = super::sketch_picking::screen_distance(projected, pixel);
        if candidate <= super::sketch_picking::SKETCH_PICK_REACH
            && distance.is_none_or(|current| {
                candidate < current.0 || (candidate == current.0 && id < current.1)
            })
        {
            distance = Some((candidate, id));
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
