// SPDX-License-Identifier: GPL-2.0-only
use crate::{Error, PartTopologyAccess, encode, part_owner::active_part};
use oblikovati_api::wire::*;
use oblikovati_model::feature::{
    AuthoredFeature, FeatureId, FeatureSequence, part::PartFeatureDefinition,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Native feature lifecycle calls that operate on the active part history.
pub const FEATURE_LIFECYCLE_METHODS: [&str; 4] = [
    METHOD_FEATURES_GET,
    METHOD_FEATURES_RENAME,
    METHOD_FEATURES_REORDER,
    METHOD_FEATURES_SET_SUPPRESSED,
];

/// Read one native feature after refreshing its current evaluation.
pub(crate) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let request: FeatureRefArgs = crate::decode(METHOD_FEATURES_GET, encoded)?;
    let errors = refresh(access, files, document)?;
    detail(files, document, request.id, errors)
}

/// Apply one native feature lifecycle edit, refresh its evaluation, then return its detail.
pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    method: &str,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let id = apply_edit(files, document, method, encoded)?;
    let errors = refresh(access, files, document)?;
    detail(files, document, id, errors)
}

fn apply_edit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    method: &str,
    encoded: &[u8],
) -> Result<u64, Error> {
    match method {
        METHOD_FEATURES_RENAME => rename(files, document, crate::decode(method, encoded)?),
        METHOD_FEATURES_REORDER => reorder(files, document, crate::decode(method, encoded)?),
        METHOD_FEATURES_SET_SUPPRESSED => {
            suppress(files, document, crate::decode(method, encoded)?)
        }
        _ => Err(Error(format!(
            "feature lifecycle {method:?}: expected an implemented lifecycle method"
        ))),
    }
}

fn rename<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    request: RenameFeatureArgs,
) -> Result<u64, Error> {
    files
        .commands(document)
        .map_err(api_error)?
        .run("Rename Feature", |snapshot, _| {
            let features = &mut snapshot.work_geometry;
            require_rename(features, request.id, &request.name)?;
            features
                .rename(FeatureId(request.id), &request.name)
                .map_err(api_error)?;
            Ok::<_, Error>(request.id)
        })
        .map_err(api_error)
}

fn reorder<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    request: ReorderFeatureArgs,
) -> Result<u64, Error> {
    let index = history_index(request.new_index)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run("Reorder Feature", |snapshot, _| {
            snapshot
                .work_geometry
                .reorder(FeatureId(request.id), index)
                .map_err(api_error)?;
            Ok::<_, Error>(request.id)
        })
        .map_err(api_error)
}

fn history_index(index: i64) -> Result<usize, Error> {
    usize::try_from(index).map_err(|cause| {
        Error(format!(
            "features.reorder index {index}: expected a non-negative history index: {cause}"
        ))
    })
}

fn suppress<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    request: SetFeatureSuppressedArgs,
) -> Result<u64, Error> {
    files
        .commands(document)
        .map_err(api_error)?
        .run("Set Feature Suppression", |snapshot, _| {
            snapshot
                .work_geometry
                .set_suppressed(FeatureId(request.id), request.suppressed)
                .map_err(api_error)?;
            Ok::<_, Error>(request.id)
        })
        .map_err(api_error)
}

fn detail<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    id: u64,
    errors: Vec<FeatureError>,
) -> Result<Vec<u8>, Error> {
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let (feature, index) = locate(&snapshot.work_geometry, id)?;
    encode(&FeatureDetailResult {
        feature: feature_detail(feature, index, &errors)?,
    })
}

fn locate(
    features: &FeatureSequence<PartFeatureDefinition>,
    id: u64,
) -> Result<(&AuthoredFeature<PartFeatureDefinition>, usize), Error> {
    let index = features
        .features()
        .iter()
        .position(|feature| feature.id() == FeatureId(id))
        .ok_or_else(|| {
            Error(format!(
                "features.get id {id}: expected an existing native feature"
            ))
        })?;
    Ok((features.get(FeatureId(id)).map_err(api_error)?, index))
}

fn feature_detail(
    feature: &AuthoredFeature<PartFeatureDefinition>,
    index: usize,
    errors: &[FeatureError],
) -> Result<FeatureDetail, Error> {
    let id = feature.id().0;
    Ok(FeatureDetail {
        feature_info: FeatureInfo {
            id,
            name: feature.name().into(),
            kind: feature.definition().kind().into(),
            suppressed: feature.suppressed(),
            health: health(feature.suppressed(), id, errors),
        },
        index: checked_index(id, index)?,
        scalars: None,
        slots: None,
    })
}

fn checked_index(id: u64, index: usize) -> Result<i64, Error> {
    i64::try_from(index).map_err(|cause| {
        Error(format!(
            "feature {id} index {index}: expected an i64: {cause}"
        ))
    })
}

fn require_rename(
    features: &oblikovati_model::feature::FeatureSequence<PartFeatureDefinition>,
    id: u64,
    name: &str,
) -> Result<(), Error> {
    if name.is_empty() {
        return Err(Error(
            "features.rename name \"\": expected a non-empty unique feature name".into(),
        ));
    }
    features.get(FeatureId(id)).map_err(api_error)?;
    if features
        .features()
        .iter()
        .any(|feature| feature.id() != FeatureId(id) && feature.name() == name)
    {
        return Err(Error(format!(
            "features.rename name {name:?}: expected a name unique within the active part"
        )));
    }
    Ok(())
}

fn health(suppressed: bool, id: u64, errors: &[FeatureError]) -> String {
    if suppressed {
        return "suppressed".into();
    }
    errors
        .iter()
        .find(|error| error.id == id)
        .map(|error| error.message.clone())
        .unwrap_or_default()
}

fn refresh<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
) -> Result<Vec<FeatureError>, Error> {
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "feature lifecycle document {document}: expected an injected exact topology owner"
        ))
    })?;
    owner.update(files, document, false).map_err(Error)
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
