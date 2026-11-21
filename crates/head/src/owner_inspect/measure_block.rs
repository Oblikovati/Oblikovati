// SPDX-License-Identifier: GPL-2.0-only
//! Resolve block members for Measure without modifying document geometry.
use super::*;
use oblikovati_model::{
    document_sketch::{PlacedSketch, SketchBlockInstance},
    history::DocumentSnapshot,
    identity::RecipeIdAllocator,
    sketch::{Sketch2D, SketchEntityId, SketchSelectionTransform},
};

pub(super) fn target(
    snapshot: &DocumentSnapshot,
    edit: crate::HeadSketchEdit,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<MeasurePick>, String> {
    let placed = snapshot
        .sketches
        .get(edit.sketch)
        .map_err(|cause| cause.to_string())?;
    if let Some(point) = measure_point(placed.geometry(), edit.plane, pixel, window)? {
        return Ok(Some(point));
    }
    if let Some((_, point)) =
        super::measure_block_points::nearest(snapshot, placed, edit.plane, pixel, window)?
    {
        return Ok(Some(point));
    }
    let loose = pick_witness(
        placed.geometry(),
        edit.plane,
        pixel,
        window,
        SketchPickFilter::Curves,
    )?;
    if placed.block_instances().is_empty() {
        return Ok(loose.map(|hit| MeasurePick::Curve(hit.entity)));
    }
    let loose = loose
        .map(|hit| {
            let projected = window.sketch_pixel(edit.plane, hit.position)?;
            Ok::<_, String>((
                super::super::sketch_picking::screen_distance(projected, pixel),
                MeasurePick::Curve(hit.entity),
            ))
        })
        .transpose()?;
    let block =
        super::super::sketch_block_pick::nearest_block_member(snapshot, edit, pixel, window, true)?
            .map(|(distance, instance, member)| {
                (distance, MeasurePick::BlockCurve { instance, member })
            });
    Ok(match (loose, block) {
        (Some(a), Some(b)) if a.0 <= b.0 => Some(a.1),
        (Some(_), Some(b)) | (None, Some(b)) => Some(b.1),
        (Some(a), None) => Some(a.1),
        (None, None) => None,
    })
}

pub(super) fn definition<'a>(
    snapshot: &'a DocumentSnapshot,
    instance: &SketchBlockInstance,
) -> Result<&'a Sketch2D, String> {
    snapshot
        .sketches
        .block_definitions()
        .iter()
        .find(|definition| definition.name() == instance.definition())
        .map(|definition| definition.geometry())
        .ok_or_else(|| {
            format!(
                "block instance {:?} definition {:?}: expected a current definition",
                instance.id(),
                instance.definition()
            )
        })
}

pub(super) fn rows(
    snapshot: &DocumentSnapshot,
    edit: crate::HeadSketchEdit,
    picks: &[MeasurePick],
) -> Result<Vec<String>, String> {
    let placed = snapshot
        .sketches
        .get(edit.sketch)
        .map_err(|cause| cause.to_string())?;
    if !picks
        .iter()
        .any(|pick| matches!(pick, MeasurePick::BlockCurve { .. }))
    {
        return measure_rows(
            &MeasureContext {
                sketch: placed.geometry(),
                plane: edit.plane,
                units: snapshot.parameters.units(),
            },
            picks,
        );
    }
    let mut geometry = if picks
        .iter()
        .any(|pick| matches!(pick, MeasurePick::Curve(_)))
    {
        placed.geometry().clone()
    } else {
        Sketch2D::default()
    };
    let maximum = geometry
        .owned_entity_ids()
        .iter()
        .map(|id| id.0)
        .max()
        .unwrap_or(0);
    let mut ids =
        RecipeIdAllocator::restore(Some(maximum), &[]).map_err(|cause| cause.to_string())?;
    let resolved = picks
        .iter()
        .map(|pick| resolve_pick(*pick, snapshot, placed, &mut geometry, &mut ids))
        .collect::<Result<Vec<_>, _>>()?;
    measure_rows(
        &MeasureContext {
            sketch: &geometry,
            plane: edit.plane,
            units: snapshot.parameters.units(),
        },
        &resolved,
    )
}
fn resolve_pick(
    pick: MeasurePick,
    snapshot: &DocumentSnapshot,
    placed: &PlacedSketch,
    geometry: &mut Sketch2D,
    ids: &mut RecipeIdAllocator,
) -> Result<MeasurePick, String> {
    let MeasurePick::BlockCurve { instance, member } = pick else {
        return Ok(pick);
    };
    let block = placed
        .block_instances()
        .iter()
        .find(|block| block.id() == instance)
        .ok_or_else(|| format!("measure block {instance:?}: expected a current instance"))?;
    let transform = SketchSelectionTransform::Similarity {
        position: block.position(),
        radians: block.rotation(),
        scale: block.scale(),
    };
    let report = geometry
        .copy_transformed_selection(
            definition(snapshot, block)?,
            &[member],
            transform,
            &mut || ids.allocate().map(SketchEntityId),
        )
        .map_err(|cause| cause.to_string())?;
    Ok(MeasurePick::Curve(report.copies[0]))
}

#[cfg(test)]
#[path = "../../tests/owner/measure_block.rs"]
mod tests;
