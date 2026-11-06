// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_picking::{SketchPickFilter, ordinary_click};
use super::*;
use crate::{HeadWindow, sketch_dimension::SketchDimensionState, sketch_tool::SketchToolState};
use oblikovati_gui::GuiDimensionAction;
use oblikovati_session::selection::SelectionClickMode;
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
    pub(super) fn apply_dimension_intent(
        &mut self,
        action: &GuiDimensionAction,
    ) -> Result<bool, String> {
        match action {
            GuiDimensionAction::Start => self.start_dimension_tool(),
            GuiDimensionAction::Apply {
                id,
                expression,
                driven,
            } => self.accept_dimension_draft(*id, expression, *driven),
            GuiDimensionAction::Cancel { id } => self.cancel_dimension_draft(*id),
        }
    }
    fn start_dimension_tool(&mut self) -> Result<bool, String> {
        self.synchronize_sketch_edit();
        self.retire_sketch_tool()?;
        let (edit, generation) = self.capture_sketch_tool_context()?;
        let serial = self.dimension_serial.checked_add(1).ok_or_else(|| {
            format!(
                "dimension draft sequence {}: expected an available u64 identity",
                self.dimension_serial
            )
        })?;
        self.clear_geometric_picks(edit)?;
        self.dimension_serial = serial;
        self.sketch_tool = Some(SketchToolState::Dimension(Box::new(SketchDimensionState {
            edit,
            generation,
            serial,
            placed: None,
            cursor: None,
        })));
        self.dimension_prompt(false);
        Ok(true)
    }
    fn accept_dimension_draft(
        &mut self,
        id: u64,
        expression: &str,
        driven: bool,
    ) -> Result<bool, String> {
        let tool = self.checked_dimension_draft(id)?;
        self.require_edit_document(tool.edit.document)?;
        let edit = tool.edit;
        let mut request = self.dimension_request(tool)?;
        if !expression.trim().is_empty() {
            request.expression = expression.into();
        }
        request.driven = driven;
        self.validate_dimension_source(edit, &request)?;
        let result = oblikovati_api::client::Client::new(self.document_host())
            .dimension()
            .add(&request)
            .map_err(|cause| cause.to_string())?;
        self.clear_geometric_picks(edit)?;
        self.sketch_tool = None;
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: format!(
                "Dimension {} applied. Sketch DOF: {}.",
                result.parameter, result.dof
            ),
        });
        Ok(true)
    }
    fn cancel_dimension_draft(&mut self, id: u64) -> Result<bool, String> {
        self.checked_dimension_draft(id)?;
        self.retire_sketch_tool()?;
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: "Dimension canceled.".into(),
        });
        Ok(true)
    }
    pub(super) fn process_dimension_event(
        &mut self,
        mut tool: Box<SketchDimensionState>,
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
            self.command_notice = Some(match self.clear_geometric_picks(tool.edit) {
                Ok(()) => DesktopCommandNotice::PlacementPrompt {
                    message: "Dimension canceled.".into(),
                },
                Err(cause) => DesktopCommandNotice::AuthoringFailed { cause },
            });
            return true;
        }
        let changed = match self.dimension_event(&mut tool, event, window) {
            Ok(changed) => changed,
            Err(cause) => {
                self.command_notice = Some(DesktopCommandNotice::AuthoringFailed { cause });
                true
            }
        };
        self.sketch_tool = Some(SketchToolState::Dimension(tool));
        changed
    }
    fn dimension_event(
        &mut self,
        tool: &mut SketchDimensionState,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        if tool.placed.is_some() || window.navigation_active() {
            return Ok(false);
        }
        if let ViewportEvent::Input(ViewportInput::Motion { position, .. }) = event {
            let previous = tool.cursor.take();
            if self.dimension_ready(tool)? {
                tool.cursor = window.sketch_point(position, tool.edit.plane)?;
            }
            return Ok(previous != tool.cursor);
        }
        let Some((pixel, _)) = ordinary_click(event) else {
            return Ok(false);
        };
        if self.pick_dimension_operand(tool, pixel, window)? {
            return Ok(true);
        }
        if !self.dimension_ready(tool)? {
            return Ok(false);
        }
        let Some(position) = window.sketch_point(pixel, tool.edit.plane)? else {
            return Ok(false);
        };
        tool.placed = Some(position);
        tool.cursor = Some(position);
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: "Dimension: enter an expression or keep the measured value.".into(),
        });
        Ok(true)
    }
    fn pick_dimension_operand(
        &mut self,
        tool: &SketchDimensionState,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let Some(hit) =
            self.resolve_sketch_hit(tool.edit, pixel, window, SketchPickFilter::Dimension)?
        else {
            return Ok(false);
        };
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
        let picks = session.selected_sketch_picks(tool.edit.sketch);
        if !tool.wants(geometry, &picks, hit.entity)? {
            return Ok(false);
        }
        let mode = if picks.is_empty() {
            SelectionClickMode::Replace
        } else {
            SelectionClickMode::Toggle
        };
        self.store_sketch_click(tool.edit, Some(hit.entity), mode)?;
        self.dimension_prompt(self.dimension_ready(tool)?);
        Ok(true)
    }
    fn dimension_prompt(&mut self, ready: bool) {
        let message = if ready {
            "Dimension: click to place the label, or select a second line for an angle."
        } else {
            "Dimension: select points, a line, a circle or two lines. Escape cancels."
        };
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: message.into(),
        });
    }
}
