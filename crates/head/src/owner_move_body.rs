// SPDX-License-Identifier: GPL-2.0-only
//! Preview and commit a generation-scoped body motion (#4097).
use super::*;
use oblikovati_gui::GuiMoveBodyRequest;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    fn validate_move_body(&self, request: &GuiMoveBodyRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let workspace = self.files.workspace();
        let session = workspace
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err(format!(
                "move body generation {}: expected current generation {}",
                request.generation,
                session.generation()
            ));
        }
        let guid = workspace
            .identity(request.document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let body = usize::try_from(request.input.body_index)
            .ok()
            .and_then(|index| evaluation.ordered_bodies().nth(index))
            .ok_or_else(|| {
                format!(
                    "move body index {}: expected a current body",
                    request.input.body_index
                )
            })?;
        let current = body.body_reference_key(session.history(), guid)?;
        if current != request.body_key.as_slice() {
            return Err(format!(
                "move body key {:?}: expected current key {:?}",
                request.body_key, current
            ));
        }
        Ok(())
    }
    pub(in crate::owner) fn preview_move_body(
        &mut self,
        request: &GuiMoveBodyRequest,
    ) -> Result<bool, String> {
        self.validate_move_body(request)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let prepared = oblikovati_automation::prepare_move_body(
            session.snapshot(),
            request.body_key.clone(),
            &request.input,
        )
        .map_err(|cause| cause.to_string())?;
        self.preview_selected_blend(request.document, |candidate, ids| {
            prepared.create(candidate, ids)
        })
    }
    pub(in crate::owner) fn commit_move_body(
        &mut self,
        request: &GuiMoveBodyRequest,
    ) -> Result<bool, String> {
        self.validate_move_body(request)?;
        oblikovati_api::client::Client::new(self.document_host())
            .features()
            .add_feature(&request.input)
            .map_err(|cause| cause.to_string())?;
        self.clear_extrusion_preview();
        Ok(true)
    }
}
