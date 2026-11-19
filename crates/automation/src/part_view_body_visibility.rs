// SPDX-License-Identifier: GPL-2.0-only
//! Resolve all body identities before applying one part-view visibility change (#4118).
use oblikovati_api::{
    Error,
    contract::PartDesignView as _,
    wire::{BodyInfo, PartDesignViewResult, SetPartDesignViewBodyVisibilityArgs},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::collections::BTreeSet;

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    request: SetPartDesignViewBodyVisibilityArgs,
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    require_active_view(files, document, request.id)?;
    let available = resolved_keys(files, access, document)?;
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let mut candidate = session
        .snapshot()
        .part_design_views
        .clone()
        .ok_or_else(|| Error("part view visibility: expected retained views".into()))?;
    if !candidate.get(request.id).map_err(Error)?.locked() {
        candidate
            .set_body_visibility(
                request.id,
                request.body_keys.as_deref().unwrap_or_default(),
                &available,
                request.visible,
            )
            .map_err(Error)?;
    }
    let reply = crate::encode(&PartDesignViewResult {
        view: crate::part_design_views::info(candidate.get(request.id).map_err(Error)?),
    })?;
    files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .set_part_view_body_visibility(
            request.body_keys.as_deref().unwrap_or_default(),
            &available,
            request.visible,
        )
        .map_err(Error)?;
    Ok(reply)
}
pub(crate) fn resolved_keys<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    document: u64,
) -> Result<BTreeSet<String>, Error> {
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "part view document {document}: expected an exact topology owner"
        ))
    })?;
    unique_keys(owner.list_bodies(files, document).map_err(Error)?)
}
fn require_active_view<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    id: u64,
) -> Result<(), Error> {
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let active = session
        .snapshot()
        .part_design_views
        .as_ref()
        .and_then(|views| views.active());
    if active.is_none_or(|view| view.id() != id) {
        return Err(Error(format!(
            "part view ID {id}: expected the active part view"
        )));
    }
    Ok(())
}
fn unique_keys(bodies: Vec<BodyInfo>) -> Result<BTreeSet<String>, Error> {
    let mut keys = BTreeSet::new();
    for body in bodies {
        if body.key.is_empty() || !keys.insert(body.key.clone()) {
            return Err(Error(format!(
                "body reference key {:?}: expected a nonempty unambiguous current key",
                body.key
            )));
        }
    }
    Ok(keys)
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn body_key_inventory_refuses_missing_or_ambiguous_references() {
        let body = |key: &str| BodyInfo {
            key: key.into(),
            ..Default::default()
        };
        assert_eq!(unique_keys(vec![body("aa"), body("bb")]).unwrap().len(), 2);
        assert!(unique_keys(vec![body("aa"), body("aa")]).is_err());
        assert!(unique_keys(vec![body("")]).is_err());
    }
}
