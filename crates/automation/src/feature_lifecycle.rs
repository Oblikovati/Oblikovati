// SPDX-License-Identifier: GPL-2.0-only
use crate::{Error, PartTopologyAccess, encode, part_owner::active_part};
use oblikovati_api::wire::*;
use oblikovati_model::{
    feature::{AuthoredFeature, FeatureId, FeatureSequence, part::PartFeatureDefinition},
    history::DocumentSnapshot,
    quantity::{Dimension, Quantity},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
mod blend_edit;
mod prism_edit;
pub use blend_edit::{apply_chamfer_distance_edit, apply_fillet_radius_edit};
pub use prism_edit::{apply_feature_distance_edit, apply_feature_extrusion_edit};

/// Native feature lifecycle calls that operate on the active part history.
pub const FEATURE_LIFECYCLE_METHODS: [&str; 5] = [
    METHOD_FEATURES_DELETE,
    METHOD_FEATURES_EDIT,
    METHOD_FEATURES_GET,
    METHOD_FEATURES_RENAME,
    METHOD_FEATURES_REORDER,
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
    fillet: Option<&dyn crate::FilletAccess>,
    method: &str,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    require_owner(access, document)?;
    let id = apply_edit(files, access, document, fillet, method, encoded)?;
    let errors = refresh(access, files, document)?;
    detail(files, document, id, errors)
}

/// Delete one unreferenced native feature and refresh the retained native evaluation.
pub(crate) fn delete<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    require_owner(access, document)?;
    let request: FeatureRefArgs = crate::decode(METHOD_FEATURES_DELETE, encoded)?;
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "document {document}: expected a feature replay owner"
        ))
    })?;
    crate::checkpoint_replay::edit_with_owner(
        files,
        owner,
        document,
        "Delete Feature",
        |snapshot, _| delete_owned_feature(snapshot, FeatureId(request.id)),
    )?;
    refresh(access, files, document)?;
    encode(&DeleteFeatureResult {
        id: request.id,
        deleted: true,
    })
}

fn delete_owned_feature(snapshot: &mut DocumentSnapshot, id: FeatureId) -> Result<(), Error> {
    let feature = snapshot.work_geometry.get(id).map_err(api_error)?;
    if feature.owned_scalars().is_empty() {
        snapshot.work_geometry.remove(id).map_err(api_error)?;
        return Ok(());
    }
    snapshot.delete_datum(id).map_err(api_error)?;
    crate::model_states::record_parameter_edits(snapshot)
}

fn apply_edit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    document: u64,
    fillet: Option<&dyn crate::FilletAccess>,
    method: &str,
    encoded: &[u8],
) -> Result<u64, Error> {
    match method {
        METHOD_FEATURES_EDIT => edit(
            files,
            access,
            document,
            fillet,
            crate::decode(method, encoded)?,
        ),
        METHOD_FEATURES_RENAME => rename(files, access, document, crate::decode(method, encoded)?),
        METHOD_FEATURES_REORDER => {
            reorder(files, access, document, crate::decode(method, encoded)?)
        }
        _ => Err(Error(format!(
            "feature lifecycle {method:?}: expected an implemented lifecycle method"
        ))),
    }
}

fn edit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    document: u64,
    fillet: Option<&dyn crate::FilletAccess>,
    request: EditFeatureArgs,
) -> Result<u64, Error> {
    if let Some(kind) = blend_edit::kind(files, document, request.id)? {
        return blend_edit::edit(files, access, document, fillet, &request, kind);
    }
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "document {document}: expected a feature replay owner"
        ))
    })?;
    let (id, _) = crate::checkpoint_replay::edit_with_owner(
        files,
        owner,
        document,
        "Edit Feature",
        |snapshot, _| {
            apply_feature_distance_edit(snapshot, &request)?;
            Ok::<_, Error>(request.id)
        },
    )?;
    Ok(id)
}

fn require_empty_repicks(request: &EditFeatureArgs) -> Result<(), Error> {
    if request.repick.as_deref().unwrap_or_default().is_empty() {
        return Ok(());
    }
    Err(Error(format!(
        "features.edit id {} repick: expected no repicks for this feature",
        request.id
    )))
}

fn single_distance_edit(request: &EditFeatureArgs) -> Result<&str, Error> {
    let edits = request.scalars.as_deref().unwrap_or_default();
    if edits.len() != 1 || edits[0].index != 0 || edits[0].value.trim().is_empty() {
        return Err(Error(format!(
            "features.edit id {} scalars {edits:?}: expected one nonempty length edit at slot 0",
            request.id
        )));
    }
    Ok(&edits[0].value)
}

fn rename<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    document: u64,
    request: RenameFeatureArgs,
) -> Result<u64, Error> {
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "document {document}: expected a feature replay owner"
        ))
    })?;
    let (id, _) = crate::checkpoint_replay::edit_with_owner(
        files,
        owner,
        document,
        "Rename Feature",
        |snapshot, _| {
            let features = &mut snapshot.work_geometry;
            require_rename(features, request.id, &request.name)?;
            features
                .rename(FeatureId(request.id), &request.name)
                .map_err(api_error)?;
            Ok::<_, Error>(request.id)
        },
    )?;
    Ok(id)
}

fn reorder<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    document: u64,
    request: ReorderFeatureArgs,
) -> Result<u64, Error> {
    let index = history_index(request.new_index)?;
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "document {document}: expected a feature replay owner"
        ))
    })?;
    let (id, _) = crate::checkpoint_replay::edit_with_owner(
        files,
        owner,
        document,
        "Reorder Feature",
        |snapshot, _| {
            snapshot
                .work_geometry
                .reorder(FeatureId(request.id), index)
                .map_err(api_error)?;
            Ok::<_, Error>(request.id)
        },
    )?;
    Ok(id)
}

fn history_index(index: i64) -> Result<usize, Error> {
    usize::try_from(index).map_err(|cause| {
        Error(format!(
            "features.reorder index {index}: expected a non-negative history index: {cause}"
        ))
    })
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
        feature: feature_detail(feature, index, &errors, snapshot)?,
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
    snapshot: &DocumentSnapshot,
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
        scalars: feature_scalars(feature, snapshot)?,
        slots: None,
    })
}

pub(crate) fn feature_scalars(
    feature: &AuthoredFeature<PartFeatureDefinition>,
    snapshot: &DocumentSnapshot,
) -> Result<Option<Vec<FeatureScalar>>, Error> {
    let (length, label, second) = match feature.definition() {
        PartFeatureDefinition::SketchPrism(definition) if definition.extent().is_none() => (
            definition.displacement()[2],
            if definition.second_depth().is_some() {
                "Distance A"
            } else {
                "Distance"
            },
            definition.second_depth(),
        ),
        PartFeatureDefinition::Fillet(definition) => (definition.radius(), "Radius", None),
        PartFeatureDefinition::Chamfer(definition) => (definition.distance(), "Distance", None),
        _ => return Ok(None),
    };
    let mut scalars = vec![length_scalar(snapshot, feature.id(), 0, length, label)?];
    if let Some(length) = second {
        scalars.push(length_scalar(
            snapshot,
            feature.id(),
            1,
            length,
            "Distance B",
        )?);
    }
    Ok(Some(scalars))
}

fn length_scalar(
    snapshot: &DocumentSnapshot,
    feature: FeatureId,
    index: i64,
    length: oblikovati_model::feature::work_geometry::DatumLength,
    label: &str,
) -> Result<FeatureScalar, Error> {
    let distance = length.resolve(&snapshot.parameters).map_err(api_error)?;
    let units = snapshot.parameters.units();
    let value = units.to_preferred(Quantity::new(distance, Dimension::Length));
    if !value.is_finite() {
        return Err(Error(format!(
            "feature {} distance {distance}: expected a finite display value",
            feature.0
        )));
    }
    Ok(FeatureScalar {
        index,
        label: label.into(),
        unit: units.preferred_name(Dimension::Length).into(),
        value,
        integer: false,
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
    owner.ensure_part(files, document).map_err(Error)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let evaluation = owner.current_part(files, document).map_err(Error)?;
    crate::part_feature_errors(session.snapshot(), evaluation.health()).map_err(Error)
}

fn require_owner<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    document: u64,
) -> Result<(), Error> {
    access.as_deref_mut().map(|_| ()).ok_or_else(|| {
        Error(format!(
            "feature lifecycle document {document}: expected an injected exact topology owner"
        ))
    })
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
