// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::{BatchSaveArgs, BatchSaveItem, BatchSaveItemResult, BatchSaveResult};
mod error;
mod preparation;
pub use error::BatchSaveError;

#[derive(Clone, Copy)]
enum SaveOperation {
    Save,
    SaveAs,
    SaveCopyAs,
}

struct PreparedSave {
    document: u64,
    full_document_name: String,
    destination: Result<PathBuf, String>,
}

impl SaveOperation {
    fn parse(operation: &str) -> Result<Self, BatchSaveError> {
        match operation {
            "save" => Ok(Self::Save),
            "saveAs" => Ok(Self::SaveAs),
            "saveCopyAs" => Ok(Self::SaveCopyAs),
            _ => Err(BatchSaveError::Operation(operation.into())),
        }
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Consume one batch, preserving item order and continuing after per-file failures.
    /// Invalid operations, missing sources, or duplicate destinations refuse the whole batch.
    /// Example: `let results = host.batch_save(request, "alpha", source)?;`.
    pub fn batch_save(
        &mut self,
        request: BatchSaveArgs,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<BatchSaveResult, BatchSaveError> {
        let operation = SaveOperation::parse(&request.operation)?;
        let prepared = self.prepare_batch(operation, request.items.unwrap_or_default())?;
        let results: Vec<_> = prepared
            .into_iter()
            .map(|item| self.execute_batch_item(operation, item, version, source))
            .collect();
        Ok(BatchSaveResult {
            saved: results.iter().filter(|item| item.ok).count() as i64,
            results: Some(results),
        })
    }

    fn execute_batch_item(
        &mut self,
        operation: SaveOperation,
        item: PreparedSave,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> BatchSaveItemResult {
        let outcome = item.destination.and_then(|path| {
            self.execute_save_operation(operation, item.document, &path, version, source)
                .map_err(|error| error.to_string())
        });
        BatchSaveItemResult {
            document: item.document,
            full_document_name: item.full_document_name,
            ok: outcome.is_ok(),
            error: outcome.err().unwrap_or_default(),
        }
    }

    fn execute_save_operation(
        &mut self,
        operation: SaveOperation,
        document: u64,
        destination: &Path,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<(), FileWorkspaceError> {
        match operation {
            SaveOperation::Save => self.save(document, version, source),
            SaveOperation::SaveAs => self.save_as(document, destination, version, source),
            SaveOperation::SaveCopyAs => self
                .save_copy(document, destination, None, version, source)
                .map(|_| ()),
        }
    }
}
