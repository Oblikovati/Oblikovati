// SPDX-License-Identifier: GPL-2.0-only
//! One GUI action groups the existing model-reference and base-view API calls.
use super::*;
use oblikovati_api::{
    client::Client,
    types::{DOCUMENT_DRAWING, DOCUMENT_PART},
    wire::{AddBaseViewArgs, SetModelReferenceArgs, TransactionBeginArgs},
};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn create_drawing_base_view(
        &mut self,
        request: &oblikovati_gui::GuiBaseViewRequest,
    ) -> Result<bool, String> {
        let reference = self.drawing_source_reference(request)?;
        let client = Client::new(self.document_host());
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "Add View".into(),
            })
            .map_err(|cause| cause.to_string())?;
        let result = (|| {
            client
                .drawing()
                .set_model_reference(&SetModelReferenceArgs {
                    full_document_name: reference,
                })?;
            client.drawing_views().add_base(&AddBaseViewArgs {
                orientation: request.orientation.clone(),
                scale: request.scale,
                center_xmm: request.center[0],
                center_ymm: request.center[1],
                ..Default::default()
            })?;
            Ok::<_, oblikovati_api::Error>(())
        })();
        if let Err(cause) = result {
            client.transactions().abort().map_err(|abort| {
                format!("drawing view failed: {cause}; transaction abort failed: {abort}")
            })?;
            return Err(cause.to_string());
        }
        client
            .transactions()
            .end()
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn drawing_source_reference(
        &self,
        request: &oblikovati_gui::GuiBaseViewRequest,
    ) -> Result<String, String> {
        let workspace = self.files.workspace();
        if workspace.active_document() != Some(request.drawing)
            || self
                .files
                .document_type(request.drawing)
                .map_err(|cause| cause.to_string())?
                != DOCUMENT_DRAWING
        {
            return Err(format!(
                "drawing {}: expected the active drawing",
                request.drawing
            ));
        }
        let session = workspace
            .session(request.drawing)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "drawing {}: expected no open transaction",
                request.drawing
            ));
        }
        if self
            .files
            .document_type(request.source)
            .map_err(|cause| cause.to_string())?
            != DOCUMENT_PART
        {
            return Err(format!(
                "drawing source {}: expected an open part",
                request.source
            ));
        }
        let reference = match self.files.path(request.source) {
            Ok(path) => path
                .to_str()
                .ok_or_else(|| format!("drawing source path {path:?}: expected UTF-8"))?
                .to_string(),
            Err(_) => workspace
                .display_name(request.source)
                .map_err(|cause| cause.to_string())?
                .to_string(),
        };
        if self
            .files
            .resolve_parameter_source(&reference)
            .map_err(|cause| cause.to_string())?
            != Some(request.source)
        {
            return Err(format!(
                "drawing source {reference:?}: expected one matching open part"
            ));
        }
        let drawing = session
            .snapshot()
            .drawing
            .as_ref()
            .ok_or("active drawing: expected drawing content")?;
        if drawing.model_reference() != reference
            && drawing
                .sheets()
                .iter()
                .any(|sheet| !sheet.views.is_empty() || !sheet.annotations.is_empty())
        {
            return Err(format!(
                "drawing source {reference:?}: expected the existing model reference {:?}",
                drawing.model_reference()
            ));
        }
        Ok(reference)
    }
}
