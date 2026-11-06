// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_picking::{SketchPickFilter, ordinary_click};
use super::*;
use crate::{
    HeadSketchEdit, HeadWindow, sketch_geometric::SketchGeometricState,
    sketch_tool::SketchToolState,
};
use oblikovati_gui::GuiSketchConstraint;
use oblikovati_session::selection::{SelectionClickMode, SketchPickTarget};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn start_geometric_tool(
        &mut self,
        choice: GuiSketchConstraint,
    ) -> Result<bool, String> {
        self.synchronize_sketch_edit();
        self.retire_sketch_tool()?;
        let tool = self.geometric_state(choice)?;
        self.clear_geometric_picks(tool.edit)?;
        self.geometric_prompt(&tool);
        self.sketch_tool = Some(SketchToolState::Geometric(Box::new(tool)));
        Ok(true)
    }
    fn geometric_state(&self, choice: GuiSketchConstraint) -> Result<SketchGeometricState, String> {
        let (edit, generation) = self.capture_sketch_tool_context()?;
        Ok(SketchGeometricState {
            edit,
            generation,
            choice,
            midpoint_lines: Default::default(),
        })
    }
    pub(super) fn process_geometric_event(
        &mut self,
        mut tool: Box<SketchGeometricState>,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        if matches!(
            event,
            ViewportEvent::Input(ViewportInput::Key {
                key: ViewportKey::Cancel,
                pressed: true
            })
        ) {
            return self.cancel_geometric_tool(&tool);
        }
        let outcome = self.geometric_pick(&mut tool, event, window);
        let changed = !matches!(&outcome, Ok(None));
        match outcome {
            Ok(Some(true)) => return true,
            Err(cause) => {
                self.command_notice = Some(DesktopCommandNotice::AuthoringFailed { cause })
            }
            _ => {}
        }
        self.sketch_tool = Some(SketchToolState::Geometric(tool));
        changed
    }
    fn geometric_pick(
        &mut self,
        tool: &mut SketchGeometricState,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<Option<bool>, String> {
        if window.navigation_active() {
            return Ok(None);
        }
        let Some((pixel, _)) = ordinary_click(event) else {
            return Ok(None);
        };
        let Some(hit) = self.resolve_sketch_hit(
            tool.edit,
            pixel,
            window,
            SketchPickFilter::Geometric(tool.choice),
        )?
        else {
            return Ok(None);
        };
        if !self.append_geometric_pick(tool, hit.entity, pixel, window)? {
            return Ok(None);
        }
        self.finish_geometric_pick(tool).map(Some)
    }
    fn finish_geometric_pick(&mut self, tool: &mut SketchGeometricState) -> Result<bool, String> {
        let requests = self.geometric_request(tool)?;
        if requests.is_empty() {
            self.geometric_prompt(tool);
            return Ok(false);
        };
        let result = self.apply_geometric_requests(tool, &requests)?;
        self.clear_geometric_picks(tool.edit)?;
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: format!(
                "{} applied. Sketch DOF: {}.",
                tool.choice.label(),
                result.dof
            ),
        });
        Ok(true)
    }
    fn geometric_midpoint_hit(
        &self,
        tool: &SketchGeometricState,
        entity: oblikovati_model::sketch::SketchEntityId,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        if tool.choice != GuiSketchConstraint::Coincident {
            return Ok(false);
        }
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let geometry = session
            .snapshot()
            .sketches
            .get(tool.edit.sketch)
            .map_err(|cause| cause.to_string())?
            .geometry();
        let Some(midpoint) = crate::sketch_geometric::line_midpoint(geometry, entity)? else {
            return Ok(false);
        };
        let projected = window.sketch_pixel(tool.edit.plane, midpoint)?;
        // Go Coincident records a line midpoint snap only on the first accepted pick.
        Ok(super::sketch_picking::screen_distance(pixel, projected) <= 12.)
    }
    fn append_geometric_pick(
        &mut self,
        tool: &mut SketchGeometricState,
        entity: oblikovati_model::sketch::SketchEntityId,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let current = self.current_geometric_picks(tool)?;
        let target = SketchPickTarget::Entity(entity);
        if current.iter().any(|pick| pick.target() == target) {
            return Ok(false);
        }
        let midpoint = self.geometric_midpoint_hit(tool, entity, pixel, window)?;
        let mode = if current.is_empty() {
            SelectionClickMode::Replace
        } else {
            SelectionClickMode::Toggle
        };
        self.store_sketch_click(tool.edit, Some(entity), mode)?;
        if midpoint {
            tool.midpoint_lines.insert(entity);
        }
        Ok(true)
    }
    fn geometric_request(
        &self,
        tool: &SketchGeometricState,
    ) -> Result<Vec<oblikovati_api::wire::AddConstraintArgs>, String> {
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let placed = session
            .snapshot()
            .sketches
            .get(tool.edit.sketch)
            .map_err(|cause| cause.to_string())?;
        let index = crate::sketch_geometric::wire_sketch_index(
            &session.snapshot().sketches,
            tool.edit.sketch,
        )?;
        tool.request(
            placed.geometry(),
            &self.current_geometric_picks(tool)?,
            index,
        )
    }
    fn current_geometric_picks(
        &self,
        tool: &SketchGeometricState,
    ) -> Result<Vec<oblikovati_session::selection::SketchPick>, String> {
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        Ok(session.selected_sketch_picks(tool.edit.sketch))
    }
    fn geometric_prompt(&mut self, tool: &SketchGeometricState) {
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: format!("{}: {}", tool.choice.label(), tool.prompt()),
        });
    }
    fn cancel_geometric_tool(&mut self, tool: &SketchGeometricState) -> bool {
        self.command_notice = Some(match self.clear_geometric_picks(tool.edit) {
            Ok(()) => DesktopCommandNotice::PlacementPrompt {
                message: format!("{} canceled.", tool.choice.label()),
            },
            Err(cause) => DesktopCommandNotice::AuthoringFailed { cause },
        });
        true
    }
    pub(super) fn clear_geometric_picks(&mut self, edit: HeadSketchEdit) -> Result<(), String> {
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .clear_transient_selection();
        Ok(())
    }
}
