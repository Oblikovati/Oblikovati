// SPDX-License-Identifier: GPL-2.0-only
//! Exact points for a running command, and what the sketch shows of its relations.
#[path = "owner_relation_marks.rs"]
mod relation_marks;
#[path = "owner_sketch_context.rs"]
mod sketch_context;
use super::*;
use crate::HeadWindow;
use crate::{placement_input::SketchPlacementState, sketch_tool::SketchToolState};
use oblikovati_gui::{GuiPrecisePoint, GuiTriadCommand};
use oblikovati_model::{
    quantity::Dimension,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
};
use relation_marks::relation_marks;
pub(super) use sketch_context::SketchContextTarget;
use sketch_context::context_target;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Place the running command's next point at typed coordinates, in the document's units.
    /// Example: `owner.precise_point_from_gui(&point)` with X "5" and Y "6".
    pub fn precise_point_from_gui(&mut self, point: &GuiPrecisePoint) -> Result<bool, String> {
        let Some(tool) = self.sketch_tool.take() else {
            return Err(format!(
                "precise point {point:?}: expected a running sketch command"
            ));
        };
        let (tool, result) = match tool {
            SketchToolState::Placement(mut tool) => {
                let result = self.place_precise_point(&mut tool, point);
                (SketchToolState::Placement(tool), result)
            }
            SketchToolState::Move(mut tool) => {
                let result = self.precise_move_point(&mut tool, point);
                (SketchToolState::Move(tool), result)
            }
            other => (
                other,
                Err(format!(
                    "precise point {point:?}: expected a point-taking command"
                )),
            ),
        };
        self.sketch_tool = Some(tool);
        result.map(|()| true)
    }
    fn place_precise_point(
        &mut self,
        tool: &mut SketchPlacementState,
        point: &GuiPrecisePoint,
    ) -> Result<(), String> {
        let position = self.precise_position(tool.document, point)?;
        match tool.tool {
            crate::SketchPlacementTool::Line => self.line_place_exact(tool, position),
            crate::SketchPlacementTool::Point => self.commit_point(tool, position, None),
            _ => self.place_exact_point(tool, position),
        }
    }
    pub(super) fn precise_position(
        &self,
        document: u64,
        point: &GuiPrecisePoint,
    ) -> Result<[f64; 2], String> {
        use oblikovati_gui::GuiPreciseMode::{DistanceAndAngle, XAndAngle, XAndY, YAndAngle};
        let first_is_length = self.typed_value(document, &point.first, Dimension::Length)?;
        let second_dimension = match point.mode {
            XAndY => Dimension::Length,
            _ => Dimension::Angle,
        };
        let second = self.typed_value(document, &point.second, second_dimension)?;
        let offset = match point.mode {
            XAndY => [first_is_length, second],
            XAndAngle => [first_is_length, first_is_length * second.tan()],
            YAndAngle => [first_is_length / second.tan(), first_is_length],
            DistanceAndAngle => [
                first_is_length * second.cos(),
                first_is_length * second.sin(),
            ],
        };
        if offset.iter().any(|value| !value.is_finite()) {
            return Err(format!(
                "precise point {point:?}: expected an angle that reaches a point"
            ));
        }
        let origin = self
            .sketch_memory
            .triad
            .filter(|_| point.relative)
            .unwrap_or_default();
        Ok([origin[0] + offset[0], origin[1] + offset[1]])
    }
    fn typed_value(&self, document: u64, text: &str, dimension: Dimension) -> Result<f64, String> {
        self.files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .parameters
            .evaluate_scalar_expression(text, dimension)
            .map(|quantity| quantity.value)
            .map_err(|cause| format!("value {text:?}: expected a {dimension:?}: {cause}"))
    }
    /// Place one point and stay ready for the next, as the command does on every click.
    pub(super) fn commit_point(
        &mut self,
        tool: &mut SketchPlacementState,
        position: [f64; 2],
        coincident_with: Option<SketchEntityId>,
    ) -> Result<(), String> {
        let placement = tool.point_placement(position, coincident_with);
        self.files
            .commands(tool.document)
            .map_err(|cause| cause.to_string())?
            .create_sketch_point(tool.sketch, &placement)
            .map_err(|cause| format!("point {position:?}: {cause}"))?;
        let generation = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?
            .generation();
        tool.restart(generation);
        self.placement_notice(tool.prompt());
        Ok(())
    }
    /// A pointer click for Point: landing on a vertex shares its position and adds a relation.
    pub(super) fn place_point_at_pointer(
        &mut self,
        tool: &mut SketchPlacementState,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
    ) -> Result<(), String> {
        let Some(at) = window.sketch_point(pixel, tool.plane)? else {
            return Ok(());
        };
        let acquired = self.point_vertex_under_pointer(tool, window, pixel)?;
        let position = acquired.map_or(at, |(position, _)| position);
        self.commit_point(tool, position, acquired.map(|(_, vertex)| vertex))
    }
    /// What the toolbar acts on: any placement command's next point.
    pub(super) fn gui_precise_target(&self) -> Option<oblikovati_gui::GuiPreciseTarget> {
        if let Some(tool) = self.placement_state() {
            return Some(self.precise_target_at(tool.document, tool.pointer_position()));
        }
        let (edit, pointer) = self.move_precise_target()?;
        Some(self.precise_target_at(edit.document, pointer))
    }
    fn precise_target_at(
        &self,
        document: u64,
        pointer: Option<[f64; 2]>,
    ) -> oblikovati_gui::GuiPreciseTarget {
        let origin = self.sketch_memory.triad.unwrap_or_default();
        let relative = pointer.map(|at| std::array::from_fn(|axis| at[axis] - origin[axis]));
        let units = self
            .files
            .workspace()
            .session(document)
            .ok()
            .map(|session| {
                let units = session.snapshot().parameters.units();
                [
                    units.preferred_name(Dimension::Length).to_owned(),
                    units.preferred_name(Dimension::Angle).to_owned(),
                ]
            })
            .unwrap_or_default();
        oblikovati_gui::GuiPreciseTarget {
            pointer: self.precise_pointer_at(document, pointer),
            relative_pointer: self.precise_pointer_at(document, relative),
            units,
            triad: None,
            origin_triad: None,
        }
    }
    fn precise_pointer_at(&self, document: u64, at: Option<[f64; 2]>) -> Option<[String; 2]> {
        let at = at?;
        let session = self.files.workspace().session(document).ok()?;
        let units = session.snapshot().parameters.units();
        let shown = at.map(|value| {
            let quantity = oblikovati_model::quantity::Quantity::new(value, Dimension::Length);
            // The display form ends in its unit; a field holds the bare number.
            units
                .format_display(quantity)
                .ok()
                .and_then(|text| text.split_whitespace().next().map(str::to_owned))
        });
        let [x, y] = shown;
        Some([x?, y?])
    }
    /// Where the triad sits on the running command's plane; example: for the frame to project.
    pub fn precise_triad_position(&self) -> Option<(SketchPlane, [f64; 2])> {
        let plane = if let Some(tool) = self.placement_state() {
            tool.plane
        } else {
            self.move_precise_target()?.0.plane
        };
        Some((plane, self.sketch_memory.triad.unwrap_or_default()))
    }
    /// Move the triad by two picks, or put it back; example: from the toolbar's buttons.
    pub fn precise_triad_from_gui(&mut self, command: GuiTriadCommand) -> Result<bool, String> {
        match command {
            GuiTriadCommand::ResetToOrigin => self.sketch_memory.triad = None,
            GuiTriadCommand::Reposition => {
                self.sketch_memory.triad_pick = Some(TriadPick::Source);
                self.placement_notice("Select the triad's origin".into());
            }
        }
        Ok(true)
    }
    /// While the triad is being moved, a press belongs to it; Escape returns to the command.
    pub(super) fn triad_pick_event(
        &mut self,
        tool: &mut SketchPlacementState,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let Some(pick) = self.sketch_memory.triad_pick else {
            return Ok(false);
        };
        if cancels_pick(event) {
            self.sketch_memory.triad_pick = None;
            self.placement_notice(tool.prompt());
            return Ok(true);
        }
        let Some(pixel) = left_press(event) else {
            return Ok(false);
        };
        self.sketch_memory.triad_pick = match pick {
            TriadPick::Source => Some(TriadPick::Target),
            TriadPick::Target => {
                self.move_triad(tool, window, pixel)?;
                None
            }
        };
        let prompt = match self.sketch_memory.triad_pick {
            Some(_) => "Select the triad's new position".to_owned(),
            None => tool.prompt(),
        };
        self.placement_notice(prompt);
        Ok(true)
    }
    fn move_triad(
        &mut self,
        tool: &SketchPlacementState,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
    ) -> Result<(), String> {
        let acquired = self.vertex_under_pointer(tool, window, pixel)?;
        let at = acquired
            .map(|(position, _)| position)
            .or(window.sketch_point(pixel, tool.plane)?);
        self.sketch_memory.triad = at;
        Ok(())
    }
    /// Change what the next points are; a running Point command asks accordingly.
    pub fn toggle_center_point_format_from_gui(&mut self) -> Result<bool, String> {
        self.sketch_memory.center_point = !self.sketch_memory.center_point;
        let center = self.sketch_memory.center_point;
        let Some(SketchToolState::Placement(tool)) = self.sketch_tool.as_mut() else {
            return Ok(true);
        };
        tool.center_point = center;
        let prompt = tool.prompt();
        self.refresh_point_role_preview()?;
        self.placement_notice(prompt);
        Ok(true)
    }
    /// Change one point's role as a history step; example: from the canvas menu.
    pub fn set_center_point_from_gui(&mut self, point: u64, center: bool) -> Result<bool, String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("point {point} center {center}: expected an edited sketch"))?;
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_center_point(edit.sketch, SketchEntityId(point), center)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    /// Remember what a right press landed on, for the canvas menu its release opens.
    pub(super) fn note_sketch_context(
        &mut self,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<(), String> {
        let ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Right,
            pressed: true,
            position,
            ..
        }) = event
        else {
            return Ok(());
        };
        // A failed new hit test must not leave the previous menu actionable (#4041).
        self.sketch_memory.context = None;
        let Some(edit) = self.active_sketch_edit() else {
            return Ok(());
        };
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
        let geometry = sketch.geometry();
        let token = self
            .sketch_memory
            .context_serial
            .checked_add(1)
            .ok_or_else(|| {
                "sketch context token overflow: expected an unused menu identity".to_owned()
            })?;
        let mut target = context_target(geometry, edit, session.generation(), position, window)?;
        if let Some(oblikovati_session::selection::SketchPickTarget::BlockInstance(instance)) =
            self.resolve_sketch_target(edit, position, window)?
        {
            target.block = Some(instance);
            target.point = None;
            target.curve = None;
            target.candidates.clear();
        }
        if target.block.is_none() {
            target.text = super::sketch_text::pick_existing_text(
                geometry,
                edit.plane,
                position,
                snapshot.parameters.units().working_scale(),
                window,
            )?;
        }
        if let Some((sketch_id, candidates)) = window.hovered_constraint_glyphs() {
            let relation_ids: std::collections::BTreeSet<_> =
                geometry.geometric_relation_ids().collect();
            let candidates: Vec<_> = candidates
                .into_iter()
                .filter(|candidate| {
                    sketch_id == edit.sketch
                        && sketch.constraints_shown()
                        && !sketch
                            .hidden_constraint_glyphs()
                            .contains(&candidate.constraint)
                        && relation_ids.contains(&candidate.constraint)
                })
                .collect();
            if !candidates.is_empty() {
                target.glyph = (candidates.len() == 1).then_some(candidates[0].constraint);
                target.glyph_candidates = candidates;
                target.block = None;
                target.text = None;
                target.point = None;
                target.curve = None;
                target.candidates.clear();
            }
        }
        target.token = token;
        self.sketch_memory.context_serial = token;
        self.sketch_memory.context = Some(target);
        Ok(())
    }
    /// Show or hide the edited sketch's relation marks; the choice is a history step.
    pub(super) fn set_constraints_shown(&mut self, shown: bool) -> Result<(), String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| format!("constraints shown {shown}: expected an edited sketch"))?;
        let index = self.sketch_glyph_wire_index(edit)?;
        oblikovati_api::client::Client::new(self.document_host())
            .constrain()
            .set_all_glyphs_visible(&oblikovati_api::wire::AllConstraintGlyphsVisibilityArgs {
                sketch_index: index,
                visible: shown,
            })
            .map(|_| ())
            .map_err(|cause| cause.to_string())
    }
    /// Each shown relation with the sketch point its mark sits at, for the frame to project.
    pub fn sketch_constraint_marks(&self) -> Vec<HeadConstraintMark> {
        let Some(edit) = self.active_sketch_edit() else {
            return Vec::new();
        };
        let Ok(session) = self.files.workspace().session(edit.document) else {
            return Vec::new();
        };
        let Ok(sketch) = session.snapshot().sketches.get(edit.sketch) else {
            return Vec::new();
        };
        if !sketch.constraints_shown() {
            return Vec::new();
        }
        relation_marks(sketch.geometry())
            .into_iter()
            .filter(|mark| !sketch.hidden_constraint_glyphs().contains(&mark.id))
            .map(|mark| HeadConstraintMark {
                sketch: edit.sketch,
                constraint: mark.id,
                plane: edit.plane,
                at: mark.at,
                text: mark.text,
                related: mark.related,
            })
            .collect()
    }
}
/// What the running sketch commands remember between events.
pub(super) struct SketchCommandMemory {
    /// New points are center points, unless a person turned that off.
    pub center_point: bool,
    pub repeat_command: Option<oblikovati_gui::GuiSketchRepeat>,
    pub move_optimize_single: bool,
    pub move_copy: bool,
    /// Where relative values are measured from; none means the sketch origin.
    pub triad: Option<[f64; 2]>,
    pub triad_pick: Option<TriadPick>,
    pub context: Option<SketchContextTarget>,
    pub context_serial: u64,
}
impl Default for SketchCommandMemory {
    fn default() -> Self {
        Self {
            center_point: true,
            repeat_command: None,
            move_optimize_single: true,
            move_copy: false,
            triad: None,
            triad_pick: None,
            context: None,
            context_serial: 0,
        }
    }
}
/// The two picks that move the triad.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum TriadPick {
    Source,
    Target,
}
fn cancels_pick(event: ViewportEvent) -> bool {
    matches!(
        event,
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::Escape,
            pressed: true,
            ..
        }) | ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Cancel,
            pressed: true,
        })
    )
}
fn left_press(event: ViewportEvent) -> Option<[i32; 2]> {
    match event {
        ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed: true,
            position,
            ..
        }) => Some(position),
        _ => None,
    }
}
/// One relation's mark as a sketch point on its plane, ready for a window to project.
#[derive(Clone, Debug, PartialEq)]
pub struct HeadConstraintMark {
    pub sketch: oblikovati_model::document_sketch::DocumentSketchId,
    pub constraint: oblikovati_model::sketch::SketchConstraintId,
    pub plane: oblikovati_model::sketch::SketchPlane,
    pub at: [f64; 2],
    pub text: String,
    pub related: Vec<SketchEntityId>,
}

#[cfg(test)]
#[path = "../tests/owner/sketch_context_other.rs"]
mod context_other_tests;
