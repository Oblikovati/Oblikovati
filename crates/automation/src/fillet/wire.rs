// SPDX-License-Identifier: GPL-2.0-only
//! Typed Fillet command with validation before publication (#4045).
use super::*;
use oblikovati_api::wire::{AddFeatureArgs, featureargs::Fillet};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn FilletAccess>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input: Fillet = serde_json::from_value(
        request.args.0.unwrap_or_else(|| serde_json::json!({})),
    )
    .map_err(|cause| {
        Error(format!(
            "fillet arguments: expected the canonical Fillet object: {cause}"
        ))
    })?;
    let edges = parse_edges(&input)?;
    let document = crate::part_owner::active_part(files)?;
    let backend = backend
        .ok_or_else(|| Error("fillet: expected an injected certified geometry service".into()))?;
    let workspace = files.workspace();
    let session = workspace.session(document).map_err(api_error)?;
    let guid = workspace
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let body = backend
        .body(session.history(), guid, &edges)
        .map_err(Error)?;
    let prepared = prepare_fillet(session.snapshot(), body, edges, &input.radius)?;
    create(files, document, guid, backend, prepared)
}
fn parse_edges(input: &Fillet) -> Result<Vec<Vec<u8>>, Error> {
    require_supported(input)?;
    let refs = input
        .edge_refs
        .as_ref()
        .filter(|refs| !refs.is_empty())
        .ok_or_else(|| {
            Error(format!(
                "fillet edgeRefs {:?}: expected one or more exact edge keys",
                input.edge_refs
            ))
        })?;
    refs.iter()
        .map(|key| crate::reference_bytes::hex_bytes(key))
        .collect()
}
fn require_supported(input: &Fillet) -> Result<(), Error> {
    let unsupported = input
        .edge_sets
        .as_ref()
        .is_some_and(|value| !value.is_empty())
        || input
            .face_refs_a
            .as_ref()
            .is_some_and(|value| !value.is_empty())
        || input
            .face_refs_b
            .as_ref()
            .is_some_and(|value| !value.is_empty())
        || input
            .edges_geom
            .as_ref()
            .is_some_and(|value| !value.is_empty())
        || !input.width.is_empty()
        || input.rho != 0.;
    if unsupported {
        return Err(Error(format!(
            "fillet options {input:?}: expected flat edgeRefs and one constant radius without face or geometric selectors"
        )));
    }
    for (name, value, default) in [
        ("cornerType", &input.corner_type, "miter"),
        ("crossSection", &input.cross_section, "arc"),
        ("concaveStrategy", &input.concave_strategy, "outward"),
    ] {
        if !value.is_empty() && !value.eq_ignore_ascii_case(default) {
            return Err(Error(format!(
                "fillet {name} {value:?}: expected {default:?} or an empty default"
            )));
        }
    }
    Ok(())
}
fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    guid: oblikovati_model::identity::DocumentGuid,
    backend: &dyn FilletAccess,
    prepared: PreparedFillet,
) -> Result<Vec<u8>, Error> {
    let name = prepared.name.clone();
    files.commands(document).map_err(api_error)?.run_projected(
        "Create fillet",
        |candidate, ids| prepared.create(candidate, ids),
        |candidate, id| {
            let bodies = backend.validate(guid, candidate, id).map_err(Error)?;
            let feature = candidate.work_geometry.get(id).map_err(api_error)?;
            crate::encode(&serde_json::json!({"id": id.0, "name": feature.name(), "kind": feature.definition().kind(), "suppressed": candidate.feature_suppressed(id).map_err(api_error)?, "health": "", "feature": name, "bodies": bodies, "healthy": true}))
        },
    ).map_err(api_error)
}
fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
