// SPDX-License-Identifier: GPL-2.0-only
//! Place a component through the shared document owner.
use super::*;

pub(super) fn place<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: PlaceOccurrenceArgs,
) -> Result<Vec<u8>, Error> {
    let owner = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("assembly.place: expected an active assembly".into()))?;
    let target = files
        .workspace()
        .session(args.document)
        .map_err(|cause| Error(cause.to_string()))?;
    let children = if target.snapshot().assembly.is_some() {
        Some(crate::file_queries::assembly::rows(
            files,
            args.document,
            1,
        )?)
    } else {
        None
    };
    files
        .place_occurrence(
            owner,
            args.document,
            &args.name,
            args.transform,
            oblikovati_martensite::validate_occurrence_placement,
            |snapshot, id| {
                let record = snapshot
                    .assembly
                    .as_ref()
                    .and_then(|records| records.iter().find(|row| row.session_id == id))
                    .ok_or_else(|| {
                        format!("occurrence {id}: expected a newly placed occurrence")
                    })?;
                let occurrence = crate::file_queries::assembly::occurrence_info(record, children)
                    .map_err(|cause| cause.to_string())?;
                encode(&OccurrenceResult { occurrence }).map_err(|cause| cause.to_string())
            },
        )
        .map_err(Error)
}
