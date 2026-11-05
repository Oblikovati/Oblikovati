// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use std::collections::BTreeSet;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    pub(super) fn prepare_batch(
        &mut self,
        operation: SaveOperation,
        items: Vec<BatchSaveItem>,
    ) -> Result<Vec<PreparedSave>, BatchSaveError> {
        for item in &items {
            self.retained_file(item.document)
                .map_err(BatchSaveError::Source)?;
        }
        let prepared: Vec<_> = items
            .into_iter()
            .map(|item| self.prepare_batch_item(operation, item))
            .collect();
        require_distinct_destinations(&prepared)?;
        Ok(prepared)
    }

    fn prepare_batch_item(
        &mut self,
        operation: SaveOperation,
        item: BatchSaveItem,
    ) -> PreparedSave {
        let (full_document_name, destination) = match operation {
            SaveOperation::Save => match self.path(item.document) {
                Ok(path) => bound_batch_destination(path),
                Err(error) => (String::new(), Err(error.to_string())),
            },
            SaveOperation::SaveAs | SaveOperation::SaveCopyAs => {
                let requested = Path::new(&item.target_file_name);
                let resolved = checked_path(requested, self.files.save_path(requested))
                    .map_err(|error| error.to_string());
                (item.target_file_name, resolved)
            }
        };
        PreparedSave {
            document: item.document,
            full_document_name,
            destination,
        }
    }
}

fn bound_batch_destination(path: &Path) -> (String, Result<PathBuf, String>) {
    match path.to_str() {
        Some(name) => (name.into(), Ok(path.into())),
        None => (
            String::new(),
            Err(format!(
                "batch source path {path:?}: expected UTF-8 for the API document name"
            )),
        ),
    }
}

fn require_distinct_destinations(prepared: &[PreparedSave]) -> Result<(), BatchSaveError> {
    let mut destinations = BTreeSet::new();
    for path in prepared
        .iter()
        .filter_map(|item| item.destination.as_ref().ok())
    {
        if !destinations.insert(path) {
            return Err(BatchSaveError::DuplicateDestination(path.clone()));
        }
    }
    Ok(())
}
