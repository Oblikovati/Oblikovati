// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{client::Client, wire::SetSketchPropertyArgs};
use oblikovati_model::document_sketch::DocumentSketchId;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_browser_sketch_visibility(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        visible: bool,
    ) -> Result<bool, String> {
        self.require_edit_document(document)?;
        let (generation, sketch_index) = self.browser_visibility_source(document, sketch)?;
        Client::new(self.document_host())
            .sketch()
            .set_property(&SetSketchPropertyArgs {
                sketch_index,
                property: "visible".into(),
                value: visible.to_string(),
            })
            .map_err(|cause| cause.to_string())?;
        let changed = self.browser_visibility_source(document, sketch)?.0 != generation;
        if changed {
            self.check_sketch_tool_context();
        }
        Ok(changed)
    }
    fn browser_visibility_source(
        &self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<(u64, i64), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let index =
            crate::sketch_geometric::wire_sketch_index(&session.snapshot().sketches, sketch)?;
        Ok((session.generation(), index))
    }
}
