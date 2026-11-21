// SPDX-License-Identifier: GPL-2.0-only
//! Route document recomputation through the retained exact evaluation owner.
use super::*;
use oblikovati_api::json::EmptyObject;

/// Canonical part update and freshness routes, subject to an injected evaluation owner.
pub const DOCUMENT_UPDATE_METHODS: [&str; 3] = [
    METHOD_DOCUMENTS_UPDATE,
    METHOD_DOCUMENTS_REBUILD,
    METHOD_DOCUMENTS_REQUIRES_UPDATE,
];

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: &mut Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    access: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    method: &str,
    bytes: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let owner: &mut dyn crate::PartUpdateAccess<O, W, F> = match replay.as_deref_mut() {
        Some(owner) => owner,
        None => access
            .as_deref_mut()
            .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>)
            .ok_or_else(|| {
                Error(format!(
                    "document {document} update: expected a part evaluation owner"
                ))
            })?,
    };
    if method == METHOD_DOCUMENTS_REQUIRES_UPDATE {
        if !bytes.is_empty() {
            let _: EmptyObject = decode(method, bytes)?;
        }
        return encode(&RequiresUpdateResult {
            requires_update: owner.part_requires_update(files, document).map_err(Error)?,
        });
    }
    let request: UpdateDocumentArgs = decode(method, bytes)?;
    let errors = owner.update_part(files, document).map_err(Error)?;
    if !request.accept_errors_and_continue && !errors.is_empty() {
        return Err(Error(format!(
            "{method}: {} feature(s) failed after recompute: {}",
            errors.len(),
            errors[0].name
        )));
    }
    encode(&UpdateDocumentResult {
        requires_update: owner.part_requires_update(files, document).map_err(Error)?,
        errors: (!errors.is_empty()).then_some(errors),
    })
}
