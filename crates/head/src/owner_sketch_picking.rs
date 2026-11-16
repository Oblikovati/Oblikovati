// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadSketchEdit, HeadWindow};
use oblikovati_gui::GuiSketchConstraint;
use oblikovati_martensite::planar_curve::{
    PlanarPickOptionsV1,
    sketch::{SketchEntityPickWitness, SketchPickQuery},
};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId, SketchPlane};
#[derive(Clone, Copy)]
pub(super) enum SketchPickFilter {
    Any,
    /// Everything but a point; example: Measure once its point reach has been checked.
    Curves,
    Geometric(GuiSketchConstraint),
    Dimension,
}
impl SketchPickFilter {
    fn accepts(self, kind: &oblikovati_api::types::SketchEntityKind) -> bool {
        match self {
            Self::Any => true,
            Self::Curves => *kind != oblikovati_api::types::SKETCH_ENTITY_POINT,
            Self::Dimension => crate::sketch_operands::category(kind).is_some(),
            Self::Geometric(choice) => crate::sketch_geometric::accepts(choice, kind),
        }
    }
}
use oblikovati_session::selection::{SelectionClickMode, SketchPickTarget};
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput};
// Match Go's snapPixels; this is a UI hit radius, not a modelling tolerance.
const SKETCH_PICK_REACH: f64 = 12.;
pub(super) type ContextCurve = (
    SketchEntityPickWitness,
    oblikovati_api::types::SketchEntityKind,
);

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn resolve_tool_candidate(
        &self,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<Option<oblikovati_viewport::ViewportSketchFeedback>, String> {
        if self.sketch_tool.is_none() {
            return Ok(self.ready_hover_feedback());
        }
        if let Some(crate::sketch_tool::SketchToolState::Measure(tool)) = self.sketch_tool.as_ref()
        {
            if self.active_sketch_edit() != Some(tool.edit) || window.navigation_active() {
                return Ok(None);
            }
            return self.measure_candidate(tool.edit, pixel, window);
        }
        let Some((edit, filter)) = self.tool_candidate_filter() else {
            return Ok(None);
        };
        if self.active_sketch_edit() != Some(edit) || window.navigation_active() {
            return Ok(None);
        }
        let Some(hit) = self.resolve_sketch_hit(edit, pixel, window, filter)? else {
            return Ok(None);
        };
        if !self.tool_accepts_candidate(edit, hit.entity)? {
            return Ok(None);
        }
        Ok(Some(oblikovati_viewport::ViewportSketchFeedback::Entities(
            oblikovati_viewport::ViewportSketchHover {
                sketch: edit.sketch,
                entities: vec![hit.entity],
            },
        )))
    }
    fn tool_candidate_filter(&self) -> Option<(HeadSketchEdit, SketchPickFilter)> {
        use crate::sketch_tool::SketchToolState;
        match self.sketch_tool.as_ref() {
            Some(SketchToolState::Geometric(tool)) => {
                Some((tool.edit, SketchPickFilter::Geometric(tool.choice)))
            }
            Some(SketchToolState::Dimension(tool)) if tool.placed.is_none() => {
                Some((tool.edit, SketchPickFilter::Dimension))
            }
            Some(SketchToolState::Move(tool)) if tool.accepts_geometry() => {
                Some((tool.edit, SketchPickFilter::Any))
            }
            _ => None,
        }
    }
    fn tool_accepts_candidate(
        &self,
        edit: HeadSketchEdit,
        entity: SketchEntityId,
    ) -> Result<bool, String> {
        use crate::sketch_tool::SketchToolState;
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let picks = session.selected_sketch_picks(edit.sketch);
        match self.sketch_tool.as_ref() {
            Some(SketchToolState::Move(_)) => Ok(true),
            Some(SketchToolState::Geometric(_)) => Ok(!picks
                .iter()
                .any(|pick| pick.target() == SketchPickTarget::Entity(entity))),
            Some(SketchToolState::Dimension(tool)) => {
                let sketch = session
                    .snapshot()
                    .sketches
                    .get(edit.sketch)
                    .map_err(|cause| cause.to_string())?;
                tool.wants(sketch.geometry(), &picks, entity)
            }
            _ => Ok(false),
        }
    }
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
        let mode = if self.block_selection == Some(edit) {
            SelectionClickMode::Toggle
        } else {
            mode
        };
        let picked = self.resolve_sketch_target(edit, position, window)?;
        self.store_sketch_target(edit, picked, mode)?;
        self.publish_selection_count(edit)?;
        Ok(true)
    }
    pub(super) fn publish_selection_count(&mut self, edit: HeadSketchEdit) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let count = session.selected_sketch_picks(edit.sketch).len();
        // Nothing selected is the sketch at rest, and the status line says so.
        self.command_notice = Some(if count == 0 {
            DesktopCommandNotice::Ready
        } else {
            DesktopCommandNotice::SelectionChanged { count }
        });
        Ok(())
    }
    pub(super) fn store_sketch_click(
        &mut self,
        edit: HeadSketchEdit,
        picked: Option<SketchEntityId>,
        mode: SelectionClickMode,
    ) -> Result<(), String> {
        self.store_sketch_target(edit, picked.map(SketchPickTarget::Entity), mode)
    }
    fn store_sketch_target(
        &mut self,
        edit: HeadSketchEdit,
        picked: Option<SketchPickTarget>,
        mode: SelectionClickMode,
    ) -> Result<(), String> {
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_selection(edit.sketch, picked, mode, false)
            .map(|_| ())
            .map_err(|cause| cause.to_string())
    }
    fn resolve_sketch_target(
        &self,
        edit: HeadSketchEdit,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<Option<SketchPickTarget>, String> {
        let loose = self.resolve_sketch_hit(edit, pixel, window, SketchPickFilter::Any)?;
        if self.block_selection == Some(edit) {
            return Ok(loose.map(|hit| SketchPickTarget::Entity(hit.entity)));
        }
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let sketch = snapshot
            .sketches
            .get(edit.sketch)
            .map_err(|cause| cause.to_string())?;
        if loose.is_some_and(|hit| sketch.geometry().point(hit.entity).is_ok()) {
            return Ok(loose.map(|hit| SketchPickTarget::Entity(hit.entity)));
        }
        let block = super::sketch_block_pick::nearest_block(snapshot, edit, pixel, window)?;
        let loose = loose
            .map(|hit| -> Result<_, String> {
                let point = window.sketch_pixel(edit.plane, hit.position)?;
                Ok((
                    screen_distance(point, pixel),
                    SketchPickTarget::Entity(hit.entity),
                ))
            })
            .transpose()?;
        Ok(super::sketch_block_pick::nearest_target(loose, block))
    }
    pub(super) fn resolve_sketch_hit(
        &self,
        edit: HeadSketchEdit,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
        filter: SketchPickFilter,
    ) -> Result<Option<SketchEntityPickWitness>, String> {
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
        if !sketch.visible() {
            return Ok(None);
        }
        pick_witness(sketch.geometry(), edit.plane, pixel, window, filter)
    }
}

pub(super) fn ordinary_click(event: ViewportEvent) -> Option<([i32; 2], SelectionClickMode)> {
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

pub(super) fn pick_witness(
    sketch: &Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
    filter: SketchPickFilter,
) -> Result<Option<SketchEntityPickWitness>, String> {
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
    // Go app/sketch_select.go gives eligible standalone and defining points priority over curves.
    if filter.accepts(&oblikovati_api::types::SKETCH_ENTITY_POINT) {
        let points =
            nearest_screen_candidate(point_ids.iter().copied(), &query, plane, pixel, window)?;
        if points.is_some() {
            return Ok(points);
        }
    }
    let curves = eligible_curves(sketch, &query, &point_ids, filter)?;
    nearest_screen_candidate(curves.into_iter(), &query, plane, pixel, window)
}
fn eligible_curves(
    sketch: &Sketch2D,
    query: &SketchPickQuery<'_>,
    points: &std::collections::BTreeSet<SketchEntityId>,
    filter: SketchPickFilter,
) -> Result<Vec<SketchEntityId>, String> {
    let mut curves = Vec::new();
    for entity in sketch
        .entity_ids()
        .iter()
        .copied()
        .filter(|id| !points.contains(id))
    {
        let kind = query
            .entity_kind(entity)
            .map_err(|cause| cause.to_string())?;
        if filter.accepts(&kind) {
            curves.push(entity);
        }
    }
    Ok(curves)
}

fn nearest_screen_candidate(
    ids: impl Iterator<Item = SketchEntityId>,
    query: &SketchPickQuery<'_>,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<SketchEntityPickWitness>, String> {
    let (witnesses, projected) = project_witnesses(ids, query, plane, window)?;
    Ok(closest_projected_witness(witnesses, projected, pixel))
}

fn project_witnesses(
    ids: impl Iterator<Item = SketchEntityId>,
    query: &SketchPickQuery<'_>,
    plane: SketchPlane,
    window: &dyn HeadWindow,
) -> Result<(Vec<SketchEntityPickWitness>, Vec<[i32; 2]>), String> {
    let witnesses = ids
        .map(|entity| query.candidate(entity).map_err(|cause| cause.to_string()))
        .collect::<Result<Vec<_>, _>>()?;
    let positions: Vec<_> = witnesses.iter().map(|witness| witness.position).collect();
    let projected = window.sketch_pixels(plane, &positions)?;
    if projected.len() != witnesses.len() {
        return Err(format!(
            "projected {} sketch witnesses: expected {} pixels",
            projected.len(),
            witnesses.len()
        ));
    }
    Ok((witnesses, projected))
}

pub(super) fn curve_context_candidates(
    sketch: &Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Vec<ContextCurve>, String> {
    let Some(cursor) = window.sketch_point(pixel, plane)? else {
        return Ok(Vec::new());
    };
    let query = SketchPickQuery::new(sketch, cursor, PlanarPickOptionsV1::default())
        .map_err(|cause| cause.to_string())?;
    let point_ids = sketch.points().iter().map(|p| p.id).collect();
    let curves = eligible_curves(sketch, &query, &point_ids, SketchPickFilter::Curves)?;
    let (witnesses, projected) = project_witnesses(curves.into_iter(), &query, plane, window)?;
    ordered_context_curves(witnesses, projected, pixel, &query)
}

fn ordered_context_curves(
    witnesses: Vec<SketchEntityPickWitness>,
    projected: Vec<[i32; 2]>,
    pixel: [i32; 2],
    query: &SketchPickQuery<'_>,
) -> Result<Vec<ContextCurve>, String> {
    let mut candidates: Vec<_> = reachable_witnesses(witnesses, projected, pixel).collect();
    candidates.sort_by(compare_witnesses);
    candidates
        .into_iter()
        .map(|(_, witness)| {
            let kind = query
                .entity_kind(witness.entity)
                .map_err(|cause| cause.to_string())?;
            Ok((witness, kind))
        })
        .collect()
}
fn closest_projected_witness(
    witnesses: Vec<SketchEntityPickWitness>,
    projected: Vec<[i32; 2]>,
    pixel: [i32; 2],
) -> Option<SketchEntityPickWitness> {
    reachable_witnesses(witnesses, projected, pixel)
        .min_by(compare_witnesses)
        .map(|(_, witness)| witness)
}

fn reachable_witnesses(
    witnesses: Vec<SketchEntityPickWitness>,
    projected: Vec<[i32; 2]>,
    pixel: [i32; 2],
) -> impl Iterator<Item = (f64, SketchEntityPickWitness)> {
    witnesses
        .into_iter()
        .zip(projected)
        .filter_map(move |(witness, at)| {
            let distance = screen_distance(at, pixel);
            (distance <= SKETCH_PICK_REACH).then_some((distance, witness))
        })
}

fn compare_witnesses(
    first: &(f64, SketchEntityPickWitness),
    second: &(f64, SketchEntityPickWitness),
) -> std::cmp::Ordering {
    first
        .0
        .total_cmp(&second.0)
        .then(first.1.entity.cmp(&second.1.entity))
}

pub(super) fn screen_distance(first: [i32; 2], second: [i32; 2]) -> f64 {
    (f64::from(first[0]) - f64::from(second[0])).hypot(f64::from(first[1]) - f64::from(second[1]))
}
