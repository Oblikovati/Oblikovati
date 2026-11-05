// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART};
pub(super) fn active_sketch_host<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<u64, Error> {
    crate::part_owner::require_active_kind(
        files,
        &[DOCUMENT_PART, DOCUMENT_ASSEMBLY],
        "part or assembly document",
    )
}
