// SPDX-License-Identifier: GPL-2.0-only
use crate::{PartTopologyAccess, decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Body metadata edits resolve an index against the current exact topology owner.
pub const BODY_COMMAND_METHODS: [&str; 1] = [METHOD_BODY_RENAME];

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    method: &str,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let request: BodyRenameArgs = decode(method, encoded)?;
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "body rename document {document}: expected an injected exact topology owner"
        ))
    })?;
    let bodies = owner.list_bodies(files, document).map_err(Error)?;
    let mut body = indexed_body(&bodies, request.body_index)?;
    store_name(files, document, &body.key, &request.name)?;
    body.name = if request.name.is_empty() {
        format!("Solid{}", request.body_index + 1)
    } else {
        request.name
    };
    encode(&BodyInfoResult { body })
}

fn indexed_body(bodies: &[BodyInfo], index: i64) -> Result<BodyInfo, Error> {
    usize::try_from(index)
        .ok()
        .and_then(|index| bodies.get(index))
        .cloned()
        .ok_or_else(|| {
            Error(format!(
                "body index {index}: expected an index in 0..{}",
                bodies.len()
            ))
        })
}

fn store_name<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    key: &str,
    name: &str,
) -> Result<(), Error> {
    files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .run("Rename Body", |snapshot, _| {
            if name.is_empty() {
                snapshot.body_names.remove(key);
            } else {
                snapshot.body_names.insert(key.into(), name.into());
            }
            Ok::<(), String>(())
        })
        .map_err(|cause| Error(cause.to_string()))
}
