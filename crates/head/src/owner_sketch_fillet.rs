// SPDX-License-Identifier: GPL-2.0-only
//! Two-line sketch fillets use the same corner command as the public host.
use super::sketch_picking::{SketchPickFilter, ordinary_click};
use super::*;
use crate::{HeadSketchEdit, HeadWindow, sketch_tool::SketchToolState};
use oblikovati_api::{client::Client, wire::AddSketchEntityArgs};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
use oblikovati_session::selection::SelectionClickMode;
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey};

// Match the Go command's 0.5 cm initial radius until the radius editor is present.
const DEFAULT_RADIUS: &str = "5 mm";

pub(crate) struct SketchFilletState {
    pub(crate) edit: HeadSketchEdit,
    pub(crate) generation: u64,
    pub(super) first: Option<SketchEntityId>,
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
    /// Enter the selected sketch and wait for a line; for example, from the Part Sketch ribbon.
    pub(super) fn start_sketch_fillet(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        self.select_placement_sketch(document, sketch)?;
        self.retire_sketch_tool()?;
        let (edit, generation) = self.capture_sketch_tool_context()?;
        self.clear_geometric_picks(edit)?;
        self.sketch_tool = Some(SketchToolState::Fillet(Box::new(SketchFilletState {
            edit,
            generation,
            first: None,
        })));
        self.sketch_fillet_prompt(false);
        Ok(())
    }

    pub(super) fn process_sketch_fillet(
        &mut self,
        mut tool: Box<SketchFilletState>,
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
            return self.cancel_sketch_fillet(tool.edit);
        }
        let outcome = self.pick_sketch_fillet_line(&mut tool, event, window);
        self.sketch_tool = Some(SketchToolState::Fillet(tool));
        outcome.unwrap_or_else(|cause| {
            self.command_notice = Some(DesktopCommandNotice::AuthoringFailed { cause });
            true
        })
    }

    fn cancel_sketch_fillet(&mut self, edit: HeadSketchEdit) -> bool {
        self.command_notice = Some(match self.clear_geometric_picks(edit) {
            Ok(()) => DesktopCommandNotice::Ready,
            Err(cause) => DesktopCommandNotice::AuthoringFailed { cause },
        });
        true
    }

    fn pick_sketch_fillet_line(
        &mut self,
        tool: &mut SketchFilletState,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let Some((pixel, _)) = ordinary_click(event).filter(|_| !window.navigation_active()) else {
            return Ok(false);
        };
        let Some(hit) =
            self.resolve_sketch_hit(tool.edit, pixel, window, SketchPickFilter::Lines)?
        else {
            return Ok(false);
        };
        if tool.first == Some(hit.entity) {
            return Ok(false);
        }
        self.accept_sketch_fillet_line(tool, hit.entity)?;
        Ok(true)
    }

    fn accept_sketch_fillet_line(
        &mut self,
        tool: &mut SketchFilletState,
        line: SketchEntityId,
    ) -> Result<(), String> {
        let Some(first) = tool.first.take() else {
            self.store_sketch_click(tool.edit, Some(line), SelectionClickMode::Replace)?;
            tool.first = Some(line);
            self.sketch_fillet_prompt(true);
            return Ok(());
        };
        self.clear_geometric_picks(tool.edit)?;
        self.commit_sketch_fillet(tool.edit, [first, line])?;
        tool.generation = self.capture_sketch_tool_context()?.1;
        self.sketch_fillet_prompt(false);
        Ok(())
    }

    fn commit_sketch_fillet(
        &mut self,
        edit: HeadSketchEdit,
        lines: [SketchEntityId; 2],
    ) -> Result<(), String> {
        let request = AddSketchEntityArgs {
            sketch_index: self.fillet_sketch_index(edit)?,
            kind: "fillet".into(),
            entity_refs: Some(lines.into_iter().map(|line| line.0).collect()),
            radius: DEFAULT_RADIUS.into(),
            construction: self.sketch_formats.construction,
            ..Default::default()
        };
        Client::new(self.document_host())
            .sketch()
            .add_entity(&request)
            .map(|_| ())
            .map_err(|cause| cause.to_string())
    }

    fn fillet_sketch_index(&self, edit: HeadSketchEdit) -> Result<i64, String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        crate::sketch_geometric::wire_sketch_index(&session.snapshot().sketches, edit.sketch)
    }

    fn sketch_fillet_prompt(&mut self, has_first: bool) {
        let next = if has_first { "second" } else { "first" };
        self.placement_notice(format!("Fillet {DEFAULT_RADIUS}: Select {next} line"));
    }
}
