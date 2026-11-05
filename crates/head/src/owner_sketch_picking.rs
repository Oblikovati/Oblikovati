// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadSketchEdit, HeadWindow};
use oblikovati_martensite::planar_curve::{PlanarPickOptionsV1, sketch::SketchPickQuery};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId, SketchPlane};
use oblikovati_session::selection::{SelectionClickMode, SketchPickTarget};
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn process_sketch_pick(
        &mut self,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        match self.apply_sketch_pick(event, window) {
            Ok(changed) => changed,
            Err(cause) => {
                self.command_notice = Some(DesktopCommandNotice::SelectionFailed { cause });
                true
            }
        }
    }
    fn apply_sketch_pick(
        &mut self,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        if self.placement_active() || window.navigation_active() {
            return Ok(false);
        }
        let Some(edit) = self.active_sketch_edit() else {
            return Ok(false);
        };
        let Some((position, mode)) = ordinary_click(event) else {
            return Ok(false);
        };
        let picked = self.resolve_sketch_pick(edit, position, window)?;
        self.store_sketch_click(edit, picked, mode)?;
        self.publish_selection_count(edit)?;
        Ok(true)
    }
    fn publish_selection_count(&mut self, edit: HeadSketchEdit) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let count = session.selected_sketch_picks(edit.sketch).len();
        self.command_notice = Some(DesktopCommandNotice::SelectionChanged { count });
        Ok(())
    }
    fn store_sketch_click(
        &mut self,
        edit: HeadSketchEdit,
        picked: Option<SketchEntityId>,
        mode: SelectionClickMode,
    ) -> Result<(), String> {
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_selection(
                edit.sketch,
                picked.map(SketchPickTarget::Entity),
                mode,
                false,
            )
            .map(|_| ())
            .map_err(|cause| cause.to_string())
    }
    fn resolve_sketch_pick(
        &self,
        edit: HeadSketchEdit,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<Option<SketchEntityId>, String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let sketch = session
            .snapshot()
            .sketches
            .get(edit.sketch)
            .map_err(|cause| cause.to_string())?;
        pick_entity(sketch.geometry(), edit.plane, pixel, window)
    }
}
fn ordinary_click(event: ViewportEvent) -> Option<([i32; 2], SelectionClickMode)> {
    let ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Left,
        pressed: true,
        position,
        modifiers,
    }) = event
    else {
        return None;
    };
    if modifiers.alt || modifiers.super_key {
        return None;
    }
    Some((
        position,
        SelectionClickMode::from_modifiers(modifiers.shift, modifiers.control),
    ))
}

fn pick_entity(
    sketch: &Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<SketchEntityId>, String> {
    let Some(cursor) = window.sketch_point(pixel, plane)? else {
        return Ok(None);
    };
    let query = SketchPickQuery::new(sketch, cursor, PlanarPickOptionsV1::default())
        .map_err(|cause| cause.to_string())?;
    let point_ids = sketch
        .points()
        .iter()
        .map(|point| point.id)
        .collect::<std::collections::BTreeSet<_>>();
    // Go app/sketch_select.go gives every standalone or defining point priority over curves.
    let points = nearest_screen_candidate(point_ids.iter().copied(), &query, plane, pixel, window)?;
    if points.is_some() {
        return Ok(points);
    }
    let curves = sketch
        .entity_ids()
        .iter()
        .copied()
        .filter(|id| !point_ids.contains(id));
    nearest_screen_candidate(curves, &query, plane, pixel, window)
}
fn nearest_screen_candidate(
    ids: impl Iterator<Item = SketchEntityId>,
    query: &SketchPickQuery<'_>,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<SketchEntityId>, String> {
    let mut candidates: Vec<(f64, SketchEntityId)> = Vec::new();
    for entity in ids {
        let witness = query.candidate(entity).map_err(|cause| cause.to_string())?;
        let projected = window.sketch_pixel(plane, witness.position)?;
        let distance = (f64::from(projected[0]) - f64::from(pixel[0]))
            .hypot(f64::from(projected[1]) - f64::from(pixel[1]));
        // Match Go's snapPixels=12; this is a UI hit radius, not a modelling tolerance.
        if distance <= 12. {
            candidates.push((distance, entity));
        }
    }
    Ok(candidates
        .into_iter()
        .min_by(|first, second| first.0.total_cmp(&second.0).then(first.1.cmp(&second.1)))
        .map(|(_, entity)| entity))
}
