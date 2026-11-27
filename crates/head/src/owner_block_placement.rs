// SPDX-License-Identifier: GPL-2.0-only
//! Repeated block placement uses the same typed API as external clients.
use super::*;
use crate::{HeadSketchEdit, HeadWindow, sketch_tool::SketchToolState};
use oblikovati_api::{client::Client, wire::AddSketchBlockArgs};
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportSketchPreview,
};

pub(crate) struct SketchBlockPlacement {
    pub edit: HeadSketchEdit,
    pub generation: u64,
    pub name: String,
    pub preview: Option<ViewportSketchPreview>,
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
    pub(super) fn start_block_placement(&mut self, name: &str) -> Result<bool, String> {
        let (edit, generation) = self.capture_sketch_tool_context()?;
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        if !session
            .snapshot()
            .sketches
            .block_definitions()
            .iter()
            .any(|item| item.name() == name)
        {
            return Err(format!(
                "block definition {name:?}: expected an existing definition"
            ));
        }
        self.retire_sketch_tool()?;
        self.store_sketch_click(
            edit,
            None,
            oblikovati_session::selection::SelectionClickMode::Replace,
        )?;
        self.placement_notice(format!("Click the insertion point for block {name:?}."));
        self.sketch_tool = Some(SketchToolState::Block(Box::new(SketchBlockPlacement {
            edit,
            generation,
            name: name.into(),
            preview: None,
        })));
        Ok(true)
    }
    pub(super) fn process_block_placement(
        &mut self,
        mut tool: Box<SketchBlockPlacement>,
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
            self.command_notice = Some(DesktopCommandNotice::Ready);
            return true;
        }
        let result = self.block_placement_event(&mut tool, event, window);
        let changed = !matches!(result, Ok(false));
        if let Err(cause) = result {
            self.command_notice = Some(DesktopCommandNotice::PlacementFailed { cause });
        }
        self.sketch_tool = Some(SketchToolState::Block(tool));
        changed
    }
    fn block_placement_event(
        &mut self,
        tool: &mut SketchBlockPlacement,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        if window.navigation_active() {
            return Ok(false);
        }
        let (pixel, place) = match event {
            ViewportEvent::Input(ViewportInput::Motion { position, .. }) => (position, false),
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Left,
                pressed: true,
                position,
                modifiers,
            }) if !modifiers.alt && !modifiers.super_key => (position, true),
            _ => return Ok(false),
        };
        let Some(point) = window.sketch_point(pixel, tool.edit.plane)? else {
            tool.preview = None;
            return Ok(true);
        };
        tool.preview = self.block_placement_preview(tool, point)?;
        if place {
            self.commit_block_placement(tool, point)?;
        }
        Ok(true)
    }
    fn block_placement_preview(
        &self,
        tool: &SketchBlockPlacement,
        point: [f64; 2],
    ) -> Result<Option<ViewportSketchPreview>, String> {
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let definition = session
            .snapshot()
            .sketches
            .block_definitions()
            .iter()
            .find(|item| item.name() == tool.name)
            .ok_or_else(|| {
                format!(
                    "block definition {:?}: expected an existing definition",
                    tool.name
                )
            })?;
        let mut geometry = definition.geometry().clone();
        let members = geometry.entity_ids().to_vec();
        if members.is_empty() {
            return Ok(None);
        }
        geometry
            .translate_entities(&members, point)
            .map_err(|cause| cause.to_string())?;
        Ok(Some(ViewportSketchPreview::Resolved {
            plane: tool.edit.plane,
            geometry: Box::new(geometry),
            members,
        }))
    }
    fn commit_block_placement(
        &mut self,
        tool: &mut SketchBlockPlacement,
        point: [f64; 2],
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        let sketch_index = crate::sketch_geometric::wire_sketch_index(
            &session.snapshot().sketches,
            tool.edit.sketch,
        )?;
        let scale = session.snapshot().parameters.units().working_scale();
        let request = AddSketchBlockArgs {
            sketch_index,
            definition: tool.name.clone(),
            position: Some(point.map(|value| value * scale).to_vec()),
            ..Default::default()
        };
        Client::new(self.document_host())
            .sketch()
            .add_block_instance(&request)
            .map_err(|cause| cause.to_string())?;
        tool.generation = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?
            .generation();
        Ok(())
    }
}
