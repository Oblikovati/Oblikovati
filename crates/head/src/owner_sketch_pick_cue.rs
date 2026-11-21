// SPDX-License-Identifier: GPL-2.0-only
//! Transient, generation-checked identification of a canvas selection (#4042).
use super::*;
use crate::{DesktopCommandNotice, HeadSketchEdit, HeadWindow};
use oblikovati_session::selection::SketchPickTarget;
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput};

#[derive(PartialEq)]
pub(super) struct SketchPickCue {
    source: CueSource,
    edit: HeadSketchEdit,
    generation: u64,
    target: SketchPickTarget,
    cue: oblikovati_gui::GuiSketchPickCue,
}

#[derive(Clone, Copy, PartialEq)]
enum CueSource {
    Selection,
    Hover,
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn update_sketch_pick_cue(
        &mut self,
        event: ViewportEvent,
        changed: bool,
        window: &dyn crate::HeadWindow,
    ) -> bool {
        match event {
            ViewportEvent::Input(ViewportInput::Motion { position, .. }) => {
                self.refresh_ready_hover(position, window)
            }
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Left,
                pressed: true,
                position,
                modifiers,
            }) if changed => self.cue_after_click(position, modifiers, window),
            ViewportEvent::Input(ViewportInput::Button { pressed: true, .. })
            | ViewportEvent::Input(ViewportInput::Wheel { .. })
            | ViewportEvent::Resized(_)
            | ViewportEvent::Focus(false)
            | ViewportEvent::Mapped(false) => self.sketch_pick_cue.take().is_some(),
            _ => false,
        }
    }
    fn cue_after_click(
        &mut self,
        pixel: [i32; 2],
        modifiers: oblikovati_viewport::ViewportModifiers,
        window: &dyn crate::HeadWindow,
    ) -> bool {
        self.sketch_pick_cue = None;
        if modifiers.shift || modifiers.control || modifiers.alt || modifiers.super_key {
            return false;
        }
        self.record_single_pick_cue(pixel, window);
        true
    }
    fn refresh_ready_hover(&mut self, pixel: [i32; 2], window: &dyn HeadWindow) -> bool {
        let (next, failed) = match self.make_hover_cue(pixel, window) {
            Ok(next) => (next, false),
            Err(cause) => {
                self.command_notice = Some(DesktopCommandNotice::HoverFailed { cause });
                (None, true)
            }
        };
        let changed = self.sketch_pick_cue != next || failed;
        self.sketch_pick_cue = next;
        changed
    }
    fn make_hover_cue(
        &self,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<Option<SketchPickCue>, String> {
        let Some(edit) = self.active_sketch_edit() else {
            return Ok(None);
        };
        if window.navigation_active() {
            return Ok(None);
        }
        let Some(hit) = self.resolve_sketch_hit(
            edit,
            pixel,
            window,
            super::sketch_picking::SketchPickFilter::Any,
        )?
        else {
            return Ok(None);
        };
        self.cue_for_hover(edit, pixel, hit).map(Some)
    }
    fn cue_for_hover(
        &self,
        edit: HeadSketchEdit,
        pixel: [i32; 2],
        hit: oblikovati_martensite::planar_curve::sketch::SketchEntityPickWitness,
    ) -> Result<SketchPickCue, String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|e| e.to_string())?;
        let sketch = session
            .snapshot()
            .sketches
            .get(edit.sketch)
            .map_err(|e| e.to_string())?;
        make_checked_cue(
            edit,
            session.generation(),
            sketch,
            SketchPickTarget::Entity(hit.entity),
            pixel,
            hit.position,
            CueSource::Hover,
        )
    }
    pub(super) fn ready_hover_feedback(
        &self,
    ) -> Option<oblikovati_viewport::ViewportSketchFeedback> {
        let cue = self.sketch_pick_cue.as_ref()?;
        if cue.source != CueSource::Hover || self.gui_sketch_pick_cue().is_none() {
            return None;
        }
        let SketchPickTarget::Entity(entity) = cue.target else {
            return None;
        };
        Some(oblikovati_viewport::ViewportSketchFeedback::Entities(
            oblikovati_viewport::ViewportSketchHover {
                sketch: cue.edit.sketch,
                entities: vec![entity],
            },
        ))
    }
    fn record_single_pick_cue(&mut self, pixel: [i32; 2], window: &dyn crate::HeadWindow) {
        if !matches!(
            self.command_notice,
            Some(DesktopCommandNotice::SelectionChanged { count: 1 })
        ) {
            return;
        }
        match self.make_sketch_pick_cue(pixel, window) {
            Ok(cue) => self.sketch_pick_cue = cue,
            Err(cause) => {
                self.command_notice = Some(DesktopCommandNotice::SelectionCueFailed { cause })
            }
        }
    }
    fn make_sketch_pick_cue(
        &self,
        pixel: [i32; 2],
        window: &dyn crate::HeadWindow,
    ) -> Result<Option<SketchPickCue>, String> {
        let Some(edit) = self.active_sketch_edit() else {
            return Ok(None);
        };
        let Some(position) = window.sketch_point(pixel, edit.plane)? else {
            return Ok(None);
        };
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|e| e.to_string())?;
        let picks = session.selected_sketch_picks(edit.sketch);
        let [pick] = picks.as_slice() else {
            return Ok(None);
        };
        let sketch = session
            .snapshot()
            .sketches
            .get(edit.sketch)
            .map_err(|e| e.to_string())?;
        make_checked_cue(
            edit,
            session.generation(),
            sketch,
            pick.target(),
            pixel,
            position,
            CueSource::Selection,
        )
        .map(Some)
    }
    pub(super) fn gui_sketch_pick_cue(&self) -> Option<oblikovati_gui::GuiSketchPickCue> {
        let cue = self.sketch_pick_cue.as_ref()?;
        if self.sketch_tool.is_some() || self.active_sketch_edit() != Some(cue.edit) {
            return None;
        }
        let session = self.files.workspace().session(cue.edit.document).ok()?;
        if session.generation() != cue.generation {
            return None;
        }
        if cue.source == CueSource::Selection {
            let picks = session.selected_sketch_picks(cue.edit.sketch);
            if picks.len() != 1 || picks[0].target() != cue.target {
                return None;
            }
        }
        Some(cue.cue.clone())
    }
}

fn make_checked_cue(
    edit: HeadSketchEdit,
    generation: u64,
    sketch: &oblikovati_model::document_sketch::PlacedSketch,
    target: SketchPickTarget,
    pixel: [i32; 2],
    position: [f64; 2],
    source: CueSource,
) -> Result<SketchPickCue, String> {
    let label = pick_kind(sketch.geometry(), target)?;
    Ok(SketchPickCue {
        source,
        edit,
        generation,
        target,
        cue: oblikovati_gui::GuiSketchPickCue {
            pixel,
            text: format!("1. {label} ({})", sketch.name()),
            plane: edit.plane,
            position,
        },
    })
}

fn pick_kind(
    sketch: &oblikovati_model::sketch::Sketch2D,
    target: SketchPickTarget,
) -> Result<String, String> {
    match target {
        SketchPickTarget::Entity(id) => {
            let kind = sketch.entity_kind(id).map_err(|e| e.to_string())?;
            Ok(kind_label(kind.0.as_ref()).to_owned())
        }
        SketchPickTarget::BlockInstance(_) => Ok("Block".into()),
        SketchPickTarget::Dimension(_) => Ok("Dimension".into()),
        SketchPickTarget::GeometricConstraint(_) => Ok("Constraint".into()),
    }
}
fn kind_label(kind: &str) -> &str {
    match kind {
        "point" => "Point",
        "line" => "Line",
        "circle" => "Circle",
        "arc" => "Arc",
        "ellipse" => "Ellipse",
        "spline" | "fixedSpline" | "controlPointSpline" | "offsetSpline" => "Spline",
        _ => kind,
    }
}

#[cfg(test)]
#[path = "../tests/owner/pick_cue/kind.rs"]
mod tests;
