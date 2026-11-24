// SPDX-License-Identifier: GPL-2.0-only
//! Resolve sketch inspection from the current part owner when faces are inputs.
use super::*;
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn part_sketch_inspection(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Option<Vec<u8>>, Error> {
        let document = crate::sketches::owner::active_sketch_host(self.files)?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|error| Error(error.to_string()))?
            .snapshot();
        if !crate::requires_part_replay(snapshot) {
            return Ok(None);
        }
        if self.part_update.is_none() && self.part_topology.is_none() {
            return Ok(None);
        }
        if !crate::sketches::inspection::query_reads_work_geometry(snapshot, method, request)? {
            return Ok(None);
        }
        let owner: &mut dyn crate::PartUpdateAccess<O, W, F> = match self.part_update.as_deref_mut()
        {
            Some(owner) => owner,
            None => self.part_topology.as_deref_mut().ok_or_else(|| {
                Error(format!(
                    "document {document} sketch query: expected a part replay owner"
                ))
            })?,
        };
        let datums = owner
            .datum_evaluation(self.files, document)
            .map_err(Error)?;
        let editing = self
            .sketch_edit
            .as_ref()
            .and_then(|edit| edit.current(self.files.workspace()));
        let focused = self
            .sketch_edit
            .as_ref()
            .and_then(|edit| edit.focused_identity(self.files.workspace()));
        let editing =
            crate::file_queries::pending_sketch_edit(self.files.workspace(), editing, focused);
        crate::sketches::inspection::query_with_datums(
            self.files,
            method,
            request,
            editing,
            Some(&datums),
        )
        .map(Some)
    }
}
