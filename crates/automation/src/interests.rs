// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{Error, wire::*};
use oblikovati_model::interest::DocumentInterests;
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

pub(super) fn list<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: ListDocumentInterestsArgs,
) -> Result<ListDocumentInterestsResult, Error> {
    Ok(ListDocumentInterestsResult {
        // Go Records returns an allocated empty slice, serialized as [] (#3684).
        interests: Some(owned(workspace, request.document)?.records().to_vec()),
    })
}

pub(super) fn has<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: HasDocumentInterestArgs,
) -> Result<HasDocumentInterestResult, Error> {
    Ok(HasDocumentInterestResult {
        has_interest: owned(workspace, request.document)?.has_interest(&request.client),
    })
}

fn owned<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    document: u64,
) -> Result<&DocumentInterests, Error> {
    Ok(&workspace
        .session(document)
        .map_err(|error| Error(error.to_string()))?
        .snapshot()
        .interests)
}

pub(super) fn add(
    interests: &mut DocumentInterests,
    request: AddDocumentInterestArgs,
) -> Result<OKResult, Error> {
    interests
        .put(request.interest)
        .map_err(|error| Error(error.to_string()))?;
    Ok(OKResult { ok: true })
}

pub(super) fn remove(
    interests: &mut DocumentInterests,
    request: RemoveDocumentInterestArgs,
) -> Result<OKResult, Error> {
    if !interests.remove(&request.client_id, &request.name) {
        return Err(Error(format!(
            "document {} interest ({:?}, {:?}): expected an existing client/name pair",
            request.document, request.client_id, request.name
        )));
    }
    Ok(OKResult { ok: true })
}
