// SPDX-License-Identifier: GPL-2.0-only
use crate::{Error, PartTopologyAccess, encode, part_owner::active_part};
use oblikovati_api::{json::EmptyObject, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Read the active part's exclusive replay cutoff.
pub(crate) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    no_arguments(METHOD_DOCUMENT_GET_END_OF_PART, encoded)?;
    let document = active_part(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    encode(&result(snapshot.work_geometry.end_of_part()))
}

/// Move the active part's cutoff and refresh its derived native result.
pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let request: SetEndOfPartArgs = crate::decode(METHOD_DOCUMENT_SET_END_OF_PART, encoded)?;
    let owner = access
        .as_deref_mut()
        .ok_or_else(|| missing_owner(document))?;
    let cutoff = cutoff(request.position)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run("Set End of Part", |snapshot, _| {
            snapshot
                .work_geometry
                .set_end_of_part(cutoff)
                .map_err(api_error)?;
            Ok::<_, Error>(result(snapshot.work_geometry.end_of_part()))
        })
        .map_err(api_error)
        .and_then(|reply| refresh(files, owner, document, reply))
}

fn refresh<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: &mut dyn PartTopologyAccess<O, W, F>,
    document: u64,
    reply: EndOfPartResult,
) -> Result<Vec<u8>, Error> {
    owner.update(files, document, false).map_err(Error)?;
    encode(&reply)
}

fn cutoff(position: i64) -> Result<Option<usize>, Error> {
    if position < 0 {
        return Ok(None);
    }
    usize::try_from(position).map(Some).map_err(|cause| {
        Error(format!(
            "end-of-part position {position}: expected a valid index: {cause}"
        ))
    })
}

fn result(cutoff: Option<usize>) -> EndOfPartResult {
    let position = cutoff
        .and_then(|index| i64::try_from(index).ok())
        .unwrap_or(-1);
    EndOfPartResult {
        position,
        rolled_back: cutoff.is_some(),
    }
}

fn no_arguments(method: &str, encoded: &[u8]) -> Result<(), Error> {
    if encoded.is_empty() {
        return Ok(());
    }
    let _: EmptyObject = crate::decode(method, encoded)?;
    Ok(())
}

fn missing_owner(document: u64) -> Error {
    Error(format!(
        "end-of-part document {document}: expected an injected exact topology owner"
    ))
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
