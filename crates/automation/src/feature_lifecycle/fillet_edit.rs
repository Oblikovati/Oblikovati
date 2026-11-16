// SPDX-License-Identifier: GPL-2.0-only
//! Validate edited fillets before adopting their recipe (#4045).
use super::*;
use oblikovati_model::feature::fillet::FilletDefinition;

pub(super) fn is_fillet<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    id: u64,
) -> Result<bool, Error> {
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let (feature, _) = locate(&snapshot.work_geometry, id)?;
    Ok(matches!(
        feature.definition(),
        PartFeatureDefinition::Fillet(_)
    ))
}

pub(super) fn edit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    backend: Option<&dyn crate::FilletAccess>,
    request: &EditFeatureArgs,
) -> Result<u64, Error> {
    let backend = backend
        .ok_or_else(|| Error("fillet edit: expected a certified geometry service".into()))?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Edit fillet",
            |snapshot, _| replace_radius(snapshot, request),
            |snapshot, id| {
                backend.validate(guid, snapshot, id).map_err(Error)?;
                Ok(id.0)
            },
        )
        .map_err(api_error)
}

fn replace_radius(
    snapshot: &mut DocumentSnapshot,
    request: &EditFeatureArgs,
) -> Result<FeatureId, Error> {
    require_empty_repicks(request)?;
    let value = single_distance_edit(request)?;
    let radius = crate::extrusion::positive_length(snapshot, value, "fillet radius")?;
    let (feature, _) = locate(&snapshot.work_geometry, request.id)?;
    let PartFeatureDefinition::Fillet(previous) = feature.definition() else {
        return Err(Error(format!("feature {}: expected a fillet", request.id)));
    };
    let definition =
        FilletDefinition::new(previous.body().to_vec(), previous.edges().to_vec(), radius)
            .map_err(api_error)?;
    let reads = definition.reads();
    let id = FeatureId(request.id);
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        )
        .map_err(api_error)?;
    Ok(id)
}
