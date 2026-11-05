// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn close(&mut self, request: CloseDocumentArgs) -> Result<Vec<u8>, Error> {
        let response = encode(&CloseDocumentsResult { closed: 1 })?;
        self.close_owned(request.id, request.force)?;
        Ok(response)
    }

    pub(super) fn close_all(&mut self, request: CloseAllDocumentsArgs) -> Result<Vec<u8>, Error> {
        // A close changes focus and removes ownership; iterate the original open order.
        let documents = self.files.workspace().documents().to_vec();
        let closed = i64::try_from(documents.len()).map_err(|_| {
            Error(format!(
                "documents.closeAll count {}: expected a signed 64-bit count",
                documents.len()
            ))
        })?;
        let response = encode(&CloseDocumentsResult { closed })?;
        for id in documents {
            self.close_owned(id, request.force)
                .map_err(|error| Error(format!("documents.closeAll: closing {id}: {error}")))?;
        }
        Ok(response)
    }

    fn close_owned(&mut self, id: u64, force: bool) -> Result<(), Error> {
        if force {
            self.files
                .close_discarding_changes(id)
                .map_err(|error| Error(error.to_string()))?;
        } else {
            self.files
                .close_saving_changes(id, self.version, self.source)
                .map_err(|error| Error(error.to_string()))?;
        }
        Ok(())
    }
}
