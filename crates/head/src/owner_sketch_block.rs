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
    pub(super) fn select_inference_geometry(
        &mut self,
        entities: Option<&[u64]>,
    ) -> Result<bool, String> {
        let Some(edit) = self.active_sketch_edit() else {
            return if entities.is_none() {
                Ok(false)
            } else {
                Err("inference selection: expected an active sketch".into())
            };
        };
        if let Some(ids) = entities.filter(|ids| !ids.is_empty()) {
            let scope = oblikovati_model::sketch::SketchInferenceCandidates::new(
                oblikovati_api::types::INFERENCE_SCOPE_SELECTED_GEOMETRY,
                &ids.iter()
                    .copied()
                    .map(oblikovati_model::sketch::SketchEntityId)
                    .collect::<Vec<_>>(),
            )
            .map_err(|error| error.to_string())?;
            let session = self
                .files
                .workspace()
                .session(edit.document)
                .map_err(|error| error.to_string())?;
            scope
                .validate_geometry(
                    session
                        .snapshot()
                        .sketches
                        .get(edit.sketch)
                        .map_err(|error| error.to_string())?
                        .geometry(),
                )
                .map_err(|error| error.to_string())?;
        }
        self.store_sketch_click(edit, None, SelectionClickMode::Replace)?;
        for id in entities.unwrap_or_default() {
            self.store_sketch_click(
                edit,
                Some(oblikovati_model::sketch::SketchEntityId(*id)),
                SelectionClickMode::Toggle,
            )?;
        }
        Ok(true)
    }
    pub(super) fn set_active_inference_scope(
        &mut self,
        scope: &oblikovati_api::wire::SketchInferenceScopeView,
    ) -> Result<bool, String> {
        let edit = self
            .active_sketch_edit()
            .ok_or("inference scope: expected an active sketch")?;
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|error| error.to_string())?;
        let index =
            crate::sketch_geometric::wire_sketch_index(&session.snapshot().sketches, edit.sketch)?;
        if scope.sketch_index != index {
            return Err(format!(
                "inference scope index {}: expected active sketch index {index}",
                scope.sketch_index
            ));
        }
        oblikovati_api::client::Client::new(self.document_host())
            .sketch()
            .set_inference_scope(scope)
            .map_err(|error| error.to_string())?;
        Ok(true)
    }
    pub(super) fn select_browser_block(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        instance: oblikovati_model::sketch::SketchEntityId,
        toggle: bool,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        if !self
            .active_sketch_edit()
            .is_some_and(|edit| edit.document == document && edit.sketch == sketch)
        {
            return Err(format!(
                "browser block {instance:?} in sketch {sketch:?}: expected the active edited sketch"
            ));
        }
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_selection(
                sketch,
                Some(SketchPickTarget::BlockInstance(instance)),
                SelectionClickMode::from_modifiers(toggle, false),
                self.sketch_tool.is_some(),
            )
            .map(|count| count.is_some())
            .map_err(|cause| cause.to_string())
    }
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
