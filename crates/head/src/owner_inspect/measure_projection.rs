// SPDX-License-Identifier: GPL-2.0-only
//! Preserve ordered Measure picks while batching pointer projection.
use crate::HeadWindow;
use oblikovati_model::sketch::SketchPlane;

pub(super) fn first_near(
    plane: SketchPlane,
    pixel: [i32; 2],
    reach: f64,
    positions: Vec<Result<[f64; 2], String>>,
    window: &dyn HeadWindow,
) -> Result<Option<[f64; 2]>, String> {
    if let Some(Err(cause)) = positions.first() {
        return Err(cause.clone());
    }
    let valid: Vec<[f64; 2]> = positions
        .iter()
        .filter_map(|position| position.as_ref().ok().copied())
        .collect();
    let projected = window.sketch_pixel_results(plane, &valid)?;
    if projected.len() != valid.len() {
        return Err(format!(
            "projected {} Measure positions: expected {} pixels",
            projected.len(),
            valid.len()
        ));
    }
    let mut projected = projected.into_iter();
    for position in positions {
        let position = position?;
        let Some(at) = projected.next() else {
            return Err("Measure projection None: expected one result per valid position".into());
        };
        if super::super::sketch_picking::screen_distance(at?, pixel) <= reach {
            return Ok(Some(position));
        }
    }
    Ok(None)
}
