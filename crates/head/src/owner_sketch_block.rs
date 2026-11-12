// SPDX-License-Identifier: GPL-2.0-only
//! The block dialog uses the document's checked selection and typed sketch API.
use super::*;
use oblikovati_session::selection::{SelectionClickMode, SketchPickTarget};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_block_selection(&mut self, active: bool) -> Result<bool, String> {
        if !active {
            return self.end_block_selection();
        }
        let edit = self
            .active_sketch_edit()
            .ok_or("block selection: expected an active sketch")?;
        self.store_sketch_click(edit, None, SelectionClickMode::Replace)?;
        self.block_selection = Some(edit);
        self.block_created = None;
        Ok(true)
    }
    fn end_block_selection(&mut self) -> Result<bool, String> {
        let old = self.block_selection.take();
        if let Some(edit) = old
            && self.active_sketch_edit() == Some(edit)
        {
            self.store_sketch_click(edit, None, SelectionClickMode::Replace)?;
        }
        Ok(old.is_some())
    }
    pub(super) fn create_selected_sketch_block(&mut self, name: &str) -> Result<bool, String> {
        let request = self.block_creation_request(name)?;
        oblikovati_api::client::Client::new(self.document_host())
            .sketch()
            .create_block_definition(&request)
            .map_err(|cause| cause.to_string())?;
        self.block_created = Some(name.into());
        Ok(true)
    }
    fn block_creation_request(
        &self,
        name: &str,
    ) -> Result<oblikovati_api::wire::CreateBlockDefinitionArgs, String> {
        let edit = self
            .active_sketch_edit()
            .filter(|edit| Some(*edit) == self.block_selection)
            .ok_or("create block: expected the active block selection")?;
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|cause| cause.to_string())?;
        let index =
            crate::sketch_geometric::wire_sketch_index(&session.snapshot().sketches, edit.sketch)?;
        let entities: Vec<_> = session
            .selected_sketch_picks(edit.sketch)
            .iter()
            .filter_map(|pick| match pick.target() {
                SketchPickTarget::Entity(id) => Some(id.0),
                _ => None,
            })
            .collect();
        if entities.is_empty() {
            return Err("create block selection []: expected one or more sketch entities".into());
        }
        let request = oblikovati_api::wire::CreateBlockDefinitionArgs {
            name: name.into(),
            source_sketch_index: index,
            entity_refs: Some(entities),
        };
        Ok(request)
    }
}
