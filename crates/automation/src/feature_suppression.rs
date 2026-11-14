// SPDX-License-Identifier: GPL-2.0-only
//! Feature suppression through the document's transactional replay path.
use oblikovati_api::{Error, wire};
use oblikovati_model::{feature::FeatureId, history::DocumentSnapshot};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn crate::ExtrudeRegionAccess>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let request: wire::SetFeatureSuppressedArgs =
        crate::decode(wire::METHOD_FEATURES_SET_SUPPRESSED, encoded)?;
    let backend = backend
        .ok_or_else(|| Error("feature suppression: expected a feature replay service".into()))?;
    let document = crate::part_owner::active_part(files)?;
    let id = FeatureId(request.id);
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Set feature suppression",
            |snapshot, _| {
                snapshot
                    .set_feature_suppressed(id, request.suppressed)
                    .map_err(api_error)?;
                Ok(())
            },
            |candidate, ()| detail(candidate, id, backend),
        )
        .map_err(api_error)
}

fn detail(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
    backend: &dyn crate::ExtrudeRegionAccess,
) -> Result<Vec<u8>, Error> {
    let feature = snapshot.work_geometry.get(id).map_err(api_error)?;
    let index = snapshot
        .work_geometry
        .features()
        .iter()
        .position(|item| item.id() == id)
        .ok_or_else(|| Error(format!("feature {id:?}: expected a history position")))?;
    let health = backend.feature_health(snapshot, id).map_err(Error)?;
    crate::encode(&wire::FeatureDetailResult {
        feature: wire::FeatureDetail {
            feature_info: wire::FeatureInfo {
                id: id.0,
                name: feature.name().into(),
                kind: feature.definition().kind().into(),
                suppressed: snapshot.feature_suppressed(id).map_err(api_error)?,
                health,
            },
            index: i64::try_from(index).map_err(api_error)?,
            scalars: crate::feature_lifecycle::prism_scalars(feature, snapshot)?,
            slots: None,
        },
    })
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
