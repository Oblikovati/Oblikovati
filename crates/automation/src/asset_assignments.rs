// SPDX-License-Identifier: GPL-2.0-only
use crate::{PartTopologyAccess, TopologyEntityKind, decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::assets::{
    AppearanceAssignmentTarget, AssignmentReference, DocumentAssets, MaterialAssignmentTarget,
    ProjectAssetLibrary,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
mod request;
use request::AssignmentChange;
/// Assignment routes require a part owner; entity targets additionally require exact topology.
pub const ASSET_ASSIGNMENT_METHODS: [&str; 2] =
    [METHOD_MODEL_ASSIGN_MATERIAL, METHOD_MODEL_ASSIGN_APPEARANCE];
pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    method: &str,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let change = request::parse(method, encoded)?;
    if let Some((key, kind)) = change.entity()? {
        access
            .as_deref_mut()
            .ok_or_else(|| {
                Error(format!(
                    "assignment target {kind:?}: expected an injected exact topology verifier"
                ))
            })?
            .verify(files, document, &key, kind)
            .map_err(Error)?;
    }
    let candidate = prepare(files, document, &change)?;
    files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .run("Assign asset", |snapshot, _| {
            snapshot.assets = candidate;
            Ok::<(), String>(())
        })
        .map_err(|cause| Error(cause.to_string()))?;
    encode(&OKResult { ok: true })
}
fn prepare<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    change: &AssignmentChange,
) -> Result<DocumentAssets, Error> {
    let assets = &files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?
        .snapshot()
        .assets;
    let catalog = files
        .builtin_catalog()
        .map_err(|cause| Error(cause.to_string()))?;
    let empty = ProjectAssetLibrary::default();
    let project = files
        .active_project()
        .map_or(&empty, |project| &project.assets);
    match change {
        AssignmentChange::Material(target, id) => {
            assets.with_material_assignment(project, catalog, target, id)
        }
        AssignmentChange::Appearance(target, id) => {
            assets.with_appearance_assignment(project, catalog, target, id)
        }
    }
    .map_err(Error)
}
