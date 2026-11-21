// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn close(&mut self, request: CloseDocumentArgs) -> Result<Vec<u8>, Error> {
        let warnings = self.close_owned(request.id, request.force)?;
        encode(&CloseDocumentsResult {
            closed: 1,
            warnings: (!warnings.is_empty()).then_some(warnings),
        })
    }

    pub(super) fn close_all(&mut self, request: CloseAllDocumentsArgs) -> Result<Vec<u8>, Error> {
        let documents = self
            .files
            .document_close_order()
            .map_err(|cause| Error(cause.to_string()))?;
        let closed = i64::try_from(documents.len()).map_err(|_| {
            Error(format!(
                "documents.closeAll count {}: expected a signed 64-bit count",
                documents.len()
            ))
        })?;
        let mut warnings = Vec::new();
        for id in documents {
            if !self.files.workspace().documents().contains(&id) {
                continue;
            }
            match self.close_owned(id, request.force) {
                Ok(item) => warnings.extend(item),
                Err(error) => {
                    if warnings.is_empty() {
                        return Err(Error(format!("documents.closeAll: closing {id}: {error}")));
                    }
                    let details: Vec<_> = warnings
                        .iter()
                        .map(|warning: &SaveWarning| {
                            format!("document {}: {}", warning.document, warning.detail)
                        })
                        .collect();
                    return Err(Error(format!(
                        "documents.closeAll: closing {id}: {error}; earlier save warnings: {}",
                        details.join("; ")
                    )));
                }
            }
        }
        encode(&CloseDocumentsResult {
            closed,
            warnings: (!warnings.is_empty()).then_some(warnings),
        })
    }

    fn close_owned(&mut self, id: u64, force: bool) -> Result<Vec<SaveWarning>, Error> {
        if force {
            self.files
                .close_discarding_changes(id)
                .map_err(|error| Error(error.to_string()))?;
            return Ok(Vec::new());
        }
        let mut warnings = Vec::new();
        if self
            .files
            .workspace()
            .needs_save(id)
            .map_err(|error| Error(error.to_string()))?
        {
            self.stage_drawing_views(id)?;
            let path = self
                .files
                .path(id)
                .map_err(|error| Error(error.to_string()))?
                .to_owned();
            warnings.extend(self.stage_assembly_view(id, &path)?.unwrap_or_default());
        }
        let result = self
            .files
            .close_saving_changes(id, self.version, self.source)
            .map_err(|error| Error(error.to_string()));
        self.files.clear_assembly_view(id);
        result?;
        Ok(warnings)
    }
}
