// SPDX-License-Identifier: GPL-2.0-only
use crate::{PartTopologyAccess, encode};
mod change;
mod delete;
use oblikovati_api::{Error, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Body commands resolve an index against the current exact topology owner.
pub const BODY_COMMAND_METHODS: [&str; 3] = [
    METHOD_BODY_RENAME,
    METHOD_BODY_SET_VISIBLE,
    METHOD_BODY_DELETE,
];

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    method: &str,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    if method == METHOD_BODY_DELETE {
        return delete::apply(files, access, encoded);
    }
    let document = crate::part_owner::active_part(files)?;
    let request = change::BodyDisplayEdit::parse(method, encoded)?;
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "body display document {document}: expected an injected exact topology owner"
        ))
    })?;
    let bodies = owner.list_bodies(files, document).map_err(Error)?;
    let mut body = indexed_body(&bodies, request.index())?;
    request.apply(files, document, &mut body)?;
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
