// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentFiles, Error, FileContext, TransactionObserver, WorkspaceObserver};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn synchronize_drawing_references(&mut self) {
        for document in self.files.workspace().documents().to_vec() {
            self.open_drawing_reference(document);
        }
    }

    pub(super) fn synchronize_assembly_references(&mut self) -> Result<(), Error> {
        let documents = self.files.workspace().documents().to_vec();
        for document in documents {
            let Ok(session) = self.files.workspace().session(document) else {
                continue;
            };
            let is_assembly = session.snapshot().assembly.is_some();
            if is_assembly {
                self.resolve_assembly_references(document)?;
            }
        }
        Ok(())
    }
}
