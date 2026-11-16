// SPDX-License-Identifier: GPL-2.0-only
//! Analytic candidates for a sketch context menu.
use super::*;

/// A right press acts on a point within this reach, in pixels.
const CONTEXT_POINT_REACH: f64 = 6.;

pub(super) fn context_target(
    geometry: &Sketch2D,
    edit: crate::HeadSketchEdit,
    generation: u64,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<SketchContextTarget, String> {
    let points = context_points(geometry, edit.plane, pixel, window)?;
    let curves =
        super::sketch_picking::curve_context_candidates(geometry, edit.plane, pixel, window)?;
    let curve = points
        .is_empty()
        .then(|| curves.first().map(|(hit, _)| hit.entity))
        .flatten();
    Ok(SketchContextTarget {
        token: 0,
        point: points.first().map(|(_, id)| *id),
        curve,
        generation,
        document: edit.document,
        sketch: edit.sketch,
        candidates: context_items(&points, &curves),
    })
}

fn context_points(
    geometry: &Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Vec<(f64, SketchEntityId)>, String> {
    let mut within = Vec::new();
    for point in geometry.points() {
        let at = window.sketch_pixel(plane, point.position)?;
        let distance = super::sketch_picking::screen_distance(at, pixel);
        if distance <= CONTEXT_POINT_REACH {
            within.push((distance, point.id));
        }
    }
    within.sort_by(|a, b| a.0.total_cmp(&b.0).then(a.1.cmp(&b.1)));
    Ok(within)
}

fn context_items(
    points: &[(f64, SketchEntityId)],
    curves: &[super::super::sketch_picking::ContextCurve],
) -> Vec<oblikovati_gui::GuiSketchCandidate> {
    let mut candidates: Vec<_> = points
        .iter()
        .map(|(_, id)| oblikovati_gui::GuiSketchCandidate {
            entity: *id,
            label: "Point".to_owned(),
        })
        .collect();
    candidates.extend(
        curves
            .iter()
            .map(|(hit, kind)| oblikovati_gui::GuiSketchCandidate {
                entity: hit.entity,
                label: context_kind_label(kind),
            }),
    );
    candidates
}

fn context_kind_label(kind: &oblikovati_api::types::SketchEntityKind) -> String {
    let mut chars = kind.0.chars();
    let Some(first) = chars.next() else {
        return "Geometry".to_owned();
    };
    first.to_uppercase().chain(chars).collect()
}
