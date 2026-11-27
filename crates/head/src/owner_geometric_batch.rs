// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch_geometric::SketchGeometricState;
use oblikovati_api::{
    client::Client,
    wire::{AddConstraintArgs, AddConstraintResult, TransactionBeginArgs},
};
use oblikovati_model::document_sketch::DocumentSketches;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn apply_geometric_requests(
        &mut self,
        tool: &mut SketchGeometricState,
        requests: &[AddConstraintArgs],
    ) -> Result<AddConstraintResult, String> {
        if requests.len() <= 1 {
            return self.add_geometric_requests(requests);
        }
        self.require_edit_document(tool.edit.document)?;
        let before = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .sketches
            .clone();
        Client::new(self.document_host())
            .transactions()
            .begin(&TransactionBeginArgs {
                label: tool.choice.label().into(),
            })
            .map_err(|cause| cause.to_string())?;
        let result = self.add_geometric_requests(requests).and_then(|reply| {
            Client::new(self.document_host())
                .transactions()
                .end()
                .map(|_| reply)
                .map_err(|cause| cause.to_string())
        });
        if result.is_err() {
            self.restore_geometric_tool(tool, &before)?;
        }
        result
    }
    fn add_geometric_requests(
        &mut self,
        requests: &[AddConstraintArgs],
    ) -> Result<AddConstraintResult, String> {
        let (first, rest) = requests.split_first().ok_or_else(|| {
            format!("constraint requests {requests:?}: expected at least one request")
        })?;
        let client = Client::new(self.document_host());
        let first = client
            .constrain()
            .add(first)
            .map_err(|cause| cause.to_string())?;
        rest.iter().try_fold(first, |_, request| {
            client
                .constrain()
                .add(request)
                .map_err(|cause| cause.to_string())
        })
    }
    fn restore_geometric_tool(
        &mut self,
        tool: &mut SketchGeometricState,
        before: &DocumentSketches,
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        // #3828: a commit veto already aborts the group; a constructor refusal does not.
        if session.in_transaction() {
            Client::new(self.document_host())
                .transactions()
                .abort()
                .map_err(|cause| cause.to_string())?;
        }
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|cause| cause.to_string())?;
        if !session.snapshot().sketches.same_recipe(before) {
            return Err(format!(
                "Fix rollback document {}: expected the original sketch recipes",
                tool.edit.document
            ));
        }
        tool.generation = session.generation();
        tool.midpoint_lines.clear();
        self.clear_geometric_picks(tool.edit)
    }
}
