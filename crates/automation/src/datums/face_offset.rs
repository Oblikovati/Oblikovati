// SPDX-License-Identifier: GPL-2.0-only
//! Bind face offsets before their transaction and replay the private candidate.
use super::*;
use oblikovati_model::feature::work_plane::WorkPlaneReference;

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: &mut Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    request: &[u8],
) -> Result<Option<Vec<u8>>, Error> {
    let args: CreateWorkPlaneArgs = decode(METHOD_WORK_PLANES_CREATE, request)?;
    if !planes::extended::has_topology_ref(&args) {
        return Ok(None);
    }
    let document = active_part(files)?;
    let owner: &mut dyn crate::PartUpdateAccess<O, W, F> = match replay.as_deref_mut() {
        Some(owner) => owner,
        None => topology
            .as_deref_mut()
            .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>)
            .ok_or_else(|| {
                Error(format!(
                    "document {document} topology work plane: expected a part replay owner"
                ))
            })?,
    };
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let prepared = planes::extended::prepare_topology_plane(&args, snapshot)?;
    if let PreparedGeometry::Offset {
        base: WorkPlaneReference::Face(face),
        ..
    } = &prepared.definition
    {
        owner
            .resolve_planar_face(files, document, face)
            .map_err(Error)?;
    }
    let (expected, reply) = preview(files, owner, document, &prepared)?;
    commit(files, document, prepared, expected, reply).map(Some)
}
fn commit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    prepared: PreparedDatum,
    expected: FeatureId,
    reply: Vec<u8>,
) -> Result<Vec<u8>, Error> {
    files.commands(document).map_err(api_error)?.run("Work Plane", |snapshot, ids| {
        let id = insert_feature(snapshot, ids, prepared)?;
        if id != expected {
            return Err(Error(format!("face offset preview feature {expected:?}, committed {id:?}: expected matching reservations")));
        }
        Ok(reply)
    }).map_err(api_error)
}
fn preview<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: &mut dyn crate::PartUpdateAccess<O, W, F>,
    document: u64,
    prepared: &PreparedDatum,
) -> Result<(FeatureId, Vec<u8>), Error> {
    let session = files.workspace().session(document).map_err(api_error)?;
    session
        .preview_projected(
            |snapshot, ids| insert_feature(snapshot, ids, prepared.clone()),
            |candidate, id| {
                let evaluation = owner
                    .preview_datums(files, document, candidate)
                    .map_err(Error)?;
                let reply = rows::creation_reply_with_evaluation(
                    candidate,
                    id,
                    prepared.collection,
                    &evaluation,
                )?;
                Ok((id, reply))
            },
        )
        .map_err(api_error)
}
