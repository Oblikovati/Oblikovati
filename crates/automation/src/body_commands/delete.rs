// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    AuthoredFeature,
    part::{DeleteBodyDefinition, PartFeatureDefinition},
};
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let request: BodyIndexArgs = crate::decode(METHOD_BODY_DELETE, encoded)?;
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "body deletion document {document}: expected an injected exact topology owner"
        ))
    })?;
    let target = resolve_target(files, owner, document, request.body_index)?;
    append_delete(files, document, target)?;
    encode(&BodyListResult {
        bodies: Some(owner.list_bodies(files, document).map_err(Error)?),
    })
}
fn append_delete<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    target: DeleteBodyDefinition,
) -> Result<(), Error> {
    files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .run("Delete Body", |snapshot, ids| {
            let id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
            let definition = PartFeatureDefinition::DeleteBody(target);
            let reads = definition.reads();
            snapshot
                .work_geometry
                .append(AuthoredFeature::new(id, "delete-body", definition, reads))
                .map_err(|cause| cause.to_string())?;
            Ok::<(), String>(())
        })
        .map_err(|cause| Error(cause.to_string()))
}

fn resolve_target<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: &mut dyn PartTopologyAccess<O, W, F>,
    document: u64,
    index: i64,
) -> Result<DeleteBodyDefinition, Error> {
    let body = indexed_body(&owner.list_bodies(files, document).map_err(Error)?, index)?;
    let key = crate::reference_bytes::hex_bytes(&body.key)?;
    owner
        .verify(files, document, &key, crate::TopologyEntityKind::Body)
        .map_err(Error)?;
    DeleteBodyDefinition::new(key).map_err(Error)
}
