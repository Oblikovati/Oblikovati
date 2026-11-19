// SPDX-License-Identifier: GPL-2.0-only
//! Choose a stable nearest point across block instances in one projection batch.
use super::*;
use oblikovati_model::{
    document_sketch::{PlacedSketch, SketchBlockInstance},
    history::DocumentSnapshot,
    sketch::{Sketch2D, SketchEntityId},
};

#[derive(Clone, Copy)]
struct BlockPoint {
    instance: SketchEntityId,
    entity: SketchEntityId,
    position: [f64; 2],
    middle: bool,
}

pub(super) fn nearest(
    snapshot: &DocumentSnapshot,
    placed: &PlacedSketch,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<(SketchEntityId, MeasurePick)>, String> {
    let mut candidates = Vec::new();
    for instance in placed.block_instances() {
        append_points(
            &mut candidates,
            super::measure_block::definition(snapshot, instance)?,
            instance,
        )?;
    }
    if candidates.is_empty() {
        return Ok(None);
    }
    let positions: Vec<_> = candidates.iter().map(|item| item.position).collect();
    let projected = window.sketch_pixels(plane, &positions)?;
    if projected.len() != candidates.len() {
        return Err(format!(
            "projected {} block points: expected {}",
            projected.len(),
            candidates.len()
        ));
    }
    Ok(closest(candidates, projected, pixel))
}

fn append_points(
    candidates: &mut Vec<BlockPoint>,
    geometry: &Sketch2D,
    instance: &SketchBlockInstance,
) -> Result<(), String> {
    for point in geometry.points() {
        append(candidates, instance, point.id, point.position, false)?;
    }
    let index = geometry.read_index();
    for line in geometry.lines() {
        append(
            candidates,
            instance,
            line.id,
            line_middle(&index, line)?,
            true,
        )?;
    }
    for arc in geometry.arcs() {
        append(candidates, instance, arc.id, arc_middle(&index, arc)?, true)?;
    }
    Ok(())
}

fn append(
    candidates: &mut Vec<BlockPoint>,
    instance: &SketchBlockInstance,
    entity: SketchEntityId,
    local: [f64; 2],
    middle: bool,
) -> Result<(), String> {
    let position = instance
        .to_sketch(local)
        .map_err(|cause| cause.to_string())?;
    candidates.push(BlockPoint {
        instance: instance.id(),
        entity,
        position,
        middle,
    });
    Ok(())
}

fn closest(
    candidates: Vec<BlockPoint>,
    projected: Vec<[i32; 2]>,
    pixel: [i32; 2],
) -> Option<(SketchEntityId, MeasurePick)> {
    candidates
        .into_iter()
        .zip(projected)
        .filter_map(|(candidate, at)| {
            let distance = super::super::sketch_picking::screen_distance(at, pixel);
            let reach = if candidate.middle {
                MIDPOINT_REACH
            } else {
                VERTEX_REACH
            };
            (distance <= reach).then_some((candidate, distance))
        })
        .min_by(|(a, x), (b, y)| {
            a.middle
                .cmp(&b.middle)
                .then(x.total_cmp(y))
                .then(a.instance.cmp(&b.instance))
                .then(a.entity.cmp(&b.entity))
        })
        .map(|(item, _)| {
            (
                item.instance,
                if item.middle {
                    MeasurePick::Middle(item.position)
                } else {
                    MeasurePick::Point(item.position)
                },
            )
        })
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn nearer_point_wins_and_equal_distance_uses_stable_identity() {
        let point = |instance, entity, x| BlockPoint {
            instance: SketchEntityId(instance),
            entity: SketchEntityId(entity),
            position: [x, 0.],
            middle: false,
        };
        let first = point(8, 1, 2.);
        let second = point(9, 1, 1.);
        let third = point(7, 2, 1.);
        let result = closest(
            vec![first, second, third],
            vec![[2, 0], [1, 0], [1, 0]],
            [0, 0],
        )
        .unwrap();
        assert_eq!(result.0, SketchEntityId(7));
        assert_eq!(result.1, MeasurePick::Point([1., 0.]));
    }
    #[test]
    fn endpoint_priority_matches_existing_sketch_measure() {
        let endpoint = BlockPoint {
            instance: SketchEntityId(2),
            entity: SketchEntityId(1),
            position: [3., 0.],
            middle: false,
        };
        let middle = BlockPoint {
            instance: SketchEntityId(1),
            entity: SketchEntityId(2),
            position: [0., 0.],
            middle: true,
        };
        let result = closest(vec![middle, endpoint], vec![[0, 0], [3, 0]], [0, 0]).unwrap();
        assert_eq!(result.1, MeasurePick::Point([3., 0.]));
    }
}
