// SPDX-License-Identifier: GPL-2.0-only
use crate::{Error, PartTopologyAccess, encode, part_owner::active_part};
use oblikovati_api::wire::*;
use oblikovati_model::feature::part::PartFeatureDefinition;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Read the active native part tree with refreshed feature health and body count.
pub(crate) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    no_arguments(encoded)?;
    let document = active_part(files)?;
    let owner = access
        .as_deref_mut()
        .ok_or_else(|| missing_owner(document))?;
    let errors = owner.update(files, document, false).map_err(Error)?;
    let bodies = owner.list_bodies(files, document).map_err(Error)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    encode(&ModelTreeResult {
        document: files.document_info(document).map_err(api_error)?.name,
        parameters: Some(parameter_names(&snapshot.parameters)),
        sketches: count(snapshot.sketches.sketches().len(), "sketches")?,
        features: Some(feature_rows(&snapshot.work_geometry, &errors)),
        bodies: count(bodies.len(), "bodies")?,
    })
}

fn parameter_names(
    parameters: &oblikovati_model::parameter::ParameterCollection<u64>,
) -> Vec<String> {
    parameters
        .ids()
        .iter()
        .filter_map(|id| parameters.get(*id))
        .map(|state| state.name.clone())
        .collect()
}

fn feature_rows(
    features: &oblikovati_model::feature::FeatureSequence<PartFeatureDefinition>,
    errors: &[FeatureError],
) -> Vec<FeatureInfo> {
    features
        .features()
        .iter()
        .map(|feature| FeatureInfo {
            id: feature.id().0,
            name: feature.name().into(),
            kind: feature.definition().kind().into(),
            suppressed: feature.suppressed(),
            health: health(feature.suppressed(), feature.id().0, errors),
        })
        .collect()
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

fn count(value: usize, field: &str) -> Result<i64, Error> {
    i64::try_from(value).map_err(|cause| {
        Error(format!(
            "model.tree {field} {value}: expected an i64: {cause}"
        ))
    })
}

fn no_arguments(encoded: &[u8]) -> Result<(), Error> {
    if encoded.is_empty() {
        return Ok(());
    }
    let _: oblikovati_api::json::EmptyObject = crate::decode(METHOD_MODEL_TREE, encoded)?;
    Ok(())
}

fn missing_owner(document: u64) -> Error {
    Error(format!(
        "model.tree document {document}: expected an injected exact topology owner"
    ))
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
